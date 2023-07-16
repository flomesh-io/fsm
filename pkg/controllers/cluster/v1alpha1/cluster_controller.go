/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package v1alpha1

import (
	"context"
	_ "embed"
	"fmt"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	mcscfg "github.com/flomesh-io/fsm/pkg/mcs/config"
	conn "github.com/flomesh-io/fsm/pkg/mcs/connector"
	cctx "github.com/flomesh-io/fsm/pkg/mcs/context"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	"github.com/flomesh-io/fsm/pkg/utils"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

// ClusterReconciler reconciles a Cluster object
type reconciler struct {
	recorder    record.EventRecorder
	fctx        *fctx.ControllerContext
	backgrounds map[string]*connectorBackground
	stopCh      chan struct{}
	mu          sync.Mutex
}

type connectorBackground struct {
	context   cctx.ConnectorContext
	connector *conn.Connector
}

func NewReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &reconciler{
		recorder:    ctx.Manager.GetEventRecorderFor("Cluster"),
		fctx:        ctx,
		backgrounds: make(map[string]*connectorBackground),
		stopCh:      utils.RegisterOSExitHandlers(),
	}

	go r.processEvent(r.fctx.Broker, r.stopCh)

	return r
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Cluster instance
	cluster := &mcsv1alpha1.Cluster{}
	if err := r.fctx.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		cluster,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("Cluster resource not found. Stopping the connector and remove the reference.")
			r.destroyConnector(cluster)

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Cluster, %v", err)
		return ctrl.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		r.destroyConnector(cluster)
	}

	mc := r.fctx.Config

	result, err := r.deriveCodebases(mc)
	if err != nil {
		return result, err
	}

	key := cluster.Key()
	klog.V(5).Infof("Cluster key is %s", key)
	bg, exists := r.backgrounds[key]
	if exists && bg.context.Hash != clusterHash(cluster) {
		klog.V(5).Infof("Background context of cluster [%s] exists, ")
		// exists and the spec changed, then stop it and start a new one
		if result, err = r.recreateConnector(ctx, bg, cluster, mc); err != nil {
			return result, err
		}
	} else if !exists {
		// doesn't exist, just create a new one
		if result, err = r.createConnector(ctx, cluster, mc); err != nil {
			return result, err
		}
	} else {
		klog.V(2).Infof("The connector %s already exists and the spec doesn't change", key)
	}

	return ctrl.Result{}, nil
}

func clusterHash(cluster *mcsv1alpha1.Cluster) string {
	return utils.SimpleHash(
		struct {
			spec            mcsv1alpha1.ClusterSpec
			resourceVersion string
			generation      int64
			uuid            string
		}{
			spec:            cluster.Spec,
			resourceVersion: cluster.ResourceVersion,
			generation:      cluster.Generation,
			uuid:            string(cluster.UID),
		},
	)
}

func (r *reconciler) deriveCodebases(mc configurator.Configurator) (ctrl.Result, error) {
	repoClient := r.fctx.RepoClient

	defaultServicesPath := utils.GetDefaultServicesPath()
	if _, err := repoClient.DeriveCodebase(defaultServicesPath, constants.DefaultServiceBasePath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	defaultIngressPath := utils.GetDefaultIngressPath()
	if _, err := repoClient.DeriveCodebase(defaultIngressPath, constants.DefaultIngressBasePath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) createConnector(ctx context.Context, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.newConnector(ctx, cluster, mc)
}

func (r *reconciler) recreateConnector(ctx context.Context, bg *connectorBackground, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	close(bg.context.StopCh)
	delete(r.backgrounds, cluster.Key())

	return r.newConnector(ctx, cluster, mc)
}

func (r *reconciler) destroyConnector(cluster *mcsv1alpha1.Cluster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := cluster.Key()
	if bg, exists := r.backgrounds[key]; exists {
		close(bg.context.StopCh)
		delete(r.backgrounds, key)
	}
}

func (r *reconciler) newConnector(ctx context.Context, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	key := cluster.Key()

	kubeconfig, result, err := getKubeConfig(cluster)
	if err != nil {
		klog.Errorf("Failed to get kubeconfig for cluster %q: %s", cluster.Key(), err)
		return result, err
	}

	connCfg, err := r.connectorConfig(cluster, mc)
	if err != nil {
		return ctrl.Result{}, err
	}

	background := cctx.ConnectorContext{
		ClusterKey:      key,
		KubeConfig:      kubeconfig,
		ConnectorConfig: connCfg,
		Hash:            clusterHash(cluster),
	}
	_, cancel := context.WithCancel(&background)
	stop := utils.RegisterExitHandlers(cancel)
	background.Cancel = cancel
	background.StopCh = stop

	connector, err := conn.NewConnector(&background, r.fctx.Broker, 15*time.Minute)
	if err != nil {
		klog.Errorf("Failed to create connector for cluster %q: %s", cluster.Key(), err)
		return ctrl.Result{}, err
	}

	r.backgrounds[key] = &connectorBackground{
		//isInCluster: cluster.Spec.IsInCluster,
		context:   background,
		connector: connector,
	}

	success := true
	errorMsg := ""
	go func() {
		if err := connector.Run(stop); err != nil {
			success = false
			errorMsg = err.Error()
			klog.Errorf("Failed to run connector for cluster %q: %s", cluster.Key(), err)
			close(stop)
			delete(r.backgrounds, key)
		}
	}()

	//if !cluster.Spec.IsInCluster {
	if success {
		return r.successJoinClusterSet(ctx, cluster, mc)
	} else {
		return r.failedJoinClusterSet(ctx, cluster, errorMsg)
	}
	//}

	//return ctrl.Result{}, nil
}

func getKubeConfig(cluster *mcsv1alpha1.Cluster) (*rest.Config, ctrl.Result, error) {
	//if cluster.Spec.IsInCluster {
	//	kubeconfig, err := rest.InClusterConfig()
	//	if err != nil {
	//		return nil, ctrl.Result{}, err
	//	}
	//
	//	return kubeconfig, ctrl.Result{}, nil
	//} else {
	return remoteKubeConfig(cluster)
	//}
}

func remoteKubeConfig(cluster *mcsv1alpha1.Cluster) (*rest.Config, ctrl.Result, error) {
	// use the current context in kubeconfig
	kubeconfig, err := clientcmd.BuildConfigFromKubeconfigGetter("", func() (*clientcmdapi.Config, error) {
		cfg, err := clientcmd.Load([]byte(cluster.Spec.Kubeconfig))
		if err != nil {
			return nil, err
		}

		return cfg, nil
	})

	if err != nil {
		return nil, ctrl.Result{}, err
	}

	return kubeconfig, ctrl.Result{}, nil
}

func (r *reconciler) connectorConfig(cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (*mcscfg.ConnectorConfig, error) {
	//if cluster.Spec.IsInCluster {
	//	return config.NewConnectorConfig(
	//		mc.Cluster.Region,
	//		mc.Cluster.Zone,
	//		mc.Cluster.Group,
	//		mc.Cluster.Name,
	//		cluster.Spec.GatewayHost,
	//		cluster.Spec.GatewayPort,
	//		cluster.Spec.IsInCluster,
	//		"",
	//	)
	//} else {
	return mcscfg.NewConnectorConfig(
		cluster.Spec.Region,
		cluster.Spec.Zone,
		cluster.Spec.Group,
		cluster.Name,
		cluster.Spec.GatewayHost,
		cluster.Spec.GatewayPort,
		mc.GetClusterUID(),
	)
	//}
}

func (r *reconciler) processEvent(broker *mcsevent.Broker, stop <-chan struct{}) {
	msgBus := broker.GetMessageBus()
	svcExportCreatedCh := msgBus.Sub(string(mcsevent.ServiceExportCreated))
	defer broker.Unsub(msgBus, svcExportCreatedCh)

	for {
		// FIXME: refine it later

		select {
		case msg, ok := <-svcExportCreatedCh:
			mc := r.fctx.Config
			// ONLY Control Plane takes care of the federation of service export/import
			if mc.IsManaged() && mc.GetMultiClusterControlPlaneUID() != "" && mc.GetClusterUID() != mc.GetMultiClusterControlPlaneUID() {
				klog.V(5).Infof("Ignore processing ServiceExportCreated event due to cluster is managed and not a control plane ...")
				continue
			}

			if !ok {
				klog.Warningf("Channel closed for ServiceExport")
				continue
			}
			klog.V(5).Infof("Received event ServiceExportCreated %v", msg)

			e, ok := msg.(mcsevent.Message)
			if !ok {
				klog.Errorf("Received unexpected message %T on channel, expected Message", e)
				continue
			}

			svcExportEvt, ok := e.NewObj.(*mcsevent.ServiceExportEvent)
			if !ok {
				klog.Errorf("Received unexpected object %T, expected *event.ServiceExportEvent", svcExportEvt)
				continue
			}

			// check ServiceExport Status, Invalid and Conflict ServiceExport is ignored
			export := svcExportEvt.ServiceExport
			if metautil.IsStatusConditionFalse(export.Status.Conditions, string(mcsv1alpha1.ServiceExportValid)) {
				klog.Warningf("ServiceExport %v is ignored due to Valid status is false", export)
				continue
			}
			if metautil.IsStatusConditionTrue(export.Status.Conditions, string(mcsv1alpha1.ServiceExportConflict)) {
				klog.Warningf("ServiceExport %v is ignored due to Conflict status is true", export)
				continue
			}

			r.processServiceExportCreatedEvent(svcExportEvt)
		case <-stop:
			klog.Infof("Received stop signal.")
			return
		}
	}
}

func (r *reconciler) processServiceExportCreatedEvent(svcExportEvt *mcsevent.ServiceExportEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	export := svcExportEvt.ServiceExport
	if r.isFirstTimeExport(svcExportEvt) {
		klog.V(5).Infof("[%s] ServiceExport %s/%s is exported first in the cluster set, will be accepted", svcExportEvt.Geo.Key(), export.Namespace, export.Name)
		r.acceptServiceExport(svcExportEvt)
	} else {
		valid, err := r.isValidServiceExport(svcExportEvt)
		if valid {
			klog.V(5).Infof("[%s] ServiceExport %s/%s is valid, will be accepted", svcExportEvt.Geo.Key(), export.Namespace, export.Name)
			r.acceptServiceExport(svcExportEvt)
		} else {
			klog.V(5).Infof("[%s] ServiceExport %s/%s is invalid, will be rejected", svcExportEvt.Geo.Key(), export.Namespace, export.Name)
			r.rejectServiceExport(svcExportEvt, err)
		}
	}
}

func (r *reconciler) isFirstTimeExport(event *mcsevent.ServiceExportEvent) bool {
	export := event.ServiceExport
	for _, bg := range r.backgrounds {
		//if bg.isInCluster {
		//	continue
		//}
		if bg.connector.ServiceImportExists(export) {
			klog.Warningf("[%s] ServiceExport %s/%s exists in Cluster %s", event.Geo.Key(), export.Namespace, export.Name, bg.context.ClusterKey)
			return false
		}
	}

	return true
}

func (r *reconciler) isValidServiceExport(svcExportEvt *mcsevent.ServiceExportEvent) (bool, error) {
	export := svcExportEvt.ServiceExport
	for _, bg := range r.backgrounds {
		//if bg.isInCluster {
		//	continue
		//}

		connectorContext := bg.context
		if connectorContext.ClusterKey == svcExportEvt.ClusterKey() {
			// no need to test against itself
			continue
		}

		if err := bg.connector.ValidateServiceExport(svcExportEvt.ServiceExport, svcExportEvt.Service); err != nil {
			klog.Warningf("[%s] ServiceExport %s/%s has conflict in Cluster %s", svcExportEvt.Geo.Key(), export.Namespace, export.Name, connectorContext.ClusterKey)
			return false, err
		}
	}

	return true, nil
}

func (r *reconciler) acceptServiceExport(svcExportEvt *mcsevent.ServiceExportEvent) {
	r.fctx.Broker.Enqueue(
		mcsevent.Message{
			Kind:   mcsevent.ServiceExportAccepted,
			OldObj: nil,
			NewObj: svcExportEvt,
		},
	)
}

func (r *reconciler) rejectServiceExport(svcExportEvt *mcsevent.ServiceExportEvent, err error) {
	svcExportEvt.Error = err.Error()

	r.fctx.Broker.Enqueue(
		mcsevent.Message{
			Kind:   mcsevent.ServiceExportRejected,
			OldObj: nil,
			NewObj: svcExportEvt,
		},
	)
}

func (r *reconciler) successJoinClusterSet(ctx context.Context, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	metautil.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ClusterManaged),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: cluster.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Success",
		Message:            fmt.Sprintf("Cluster %s joined ClusterSet successfully.", cluster.Key()),
	})

	if err := r.fctx.Status().Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) failedJoinClusterSet(ctx context.Context, cluster *mcsv1alpha1.Cluster, err string) (ctrl.Result, error) {
	metautil.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ClusterManaged),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: cluster.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("Cluster %s failed to join ClusterSet: %s.", cluster.Key(), err),
	})

	if err := r.fctx.Status().Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcsv1alpha1.Cluster{}).
		Owns(&corev1.Secret{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
