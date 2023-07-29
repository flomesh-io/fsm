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

// Package v1alpha1 contains controller logic for the Cluster API v1alpha1.
package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/logger"
	cp "github.com/flomesh-io/fsm/pkg/mcs/ctrl"
	"github.com/flomesh-io/fsm/pkg/mcs/remote"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterReconciler reconciles a Cluster object
type reconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	stopCh   chan struct{}
	mu       sync.Mutex
	server   *cp.ControlPlaneServer
}

var (
	log = logger.New("cluster-controller/v1alpha1")
)

// NewReconciler returns a new reconciler for Cluster objects
func NewReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &reconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Cluster"),
		fctx:     ctx,
		stopCh:   utils.RegisterOSExitHandlers(),
		server:   cp.NewControlPlaneServer(ctx.Config, ctx.Broker),
	}

	go r.server.Run(r.stopCh)

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
			log.Info().Msgf("Cluster resource not found. Stopping the connector and remove the reference.")
			r.destroyConnector(cluster)

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get Cluster, %v", err)
		return ctrl.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		r.destroyConnector(cluster)
	}

	mc := r.fctx.Config

	result, err := r.deriveCodebases(cluster, mc)
	if err != nil {
		return result, err
	}

	key := cluster.Key()
	log.Info().Msgf("Cluster key is %s", key)
	bg, exists := r.server.GetBackground(key)
	if exists && bg.Context.Hash != clusterHash(cluster) {
		log.Info().Msgf("Background context of cluster [%s] exists, ", key)
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
		log.Warn().Msgf("The connector %s already exists and the spec doesn't change", key)
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

func (r *reconciler) deriveCodebases(cluster *mcsv1alpha1.Cluster, _ configurator.Configurator) (ctrl.Result, error) {
	repoClient := r.fctx.RepoClient

	bytes, jsonErr := json.Marshal(cluster)
	if jsonErr != nil {
		return ctrl.Result{}, jsonErr
	}
	version := utils.Hash(bytes)

	defaultServicesPath := utils.GetDefaultServicesPath()
	if _, err := repoClient.DeriveCodebase(defaultServicesPath, constants.DefaultServiceBasePath, version); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	defaultIngressPath := utils.GetDefaultIngressPath()
	if _, err := repoClient.DeriveCodebase(defaultIngressPath, constants.DefaultIngressBasePath, version); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) createConnector(ctx context.Context, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.newConnector(ctx, cluster, mc)
}

func (r *reconciler) recreateConnector(ctx context.Context, _ *remote.Background, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.server.DestroyBackground(cluster.Key())

	return r.newConnector(ctx, cluster, mc)
}

func (r *reconciler) destroyConnector(cluster *mcsv1alpha1.Cluster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := cluster.Key()
	r.server.DestroyBackground(key)
}

func (r *reconciler) newConnector(ctx context.Context, cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (ctrl.Result, error) {
	key := cluster.Key()

	kubeconfig, result, err := getKubeConfig(cluster)
	if err != nil {
		log.Error().Msgf("Failed to get kubeconfig for cluster %q: %s", cluster.Key(), err)
		return result, err
	}

	background, err := remote.NewBackground(cluster, kubeconfig, r.fctx.Config, r.fctx.Broker)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.server.AddBackground(key, background)

	success := true
	errorMsg := ""
	go func() {
		if err := background.Run(); err != nil {
			success = false
			errorMsg = err.Error()
			log.Error().Msgf("Failed to run connector for cluster %q: %s", cluster.Key(), err)
			r.server.DestroyBackground(key)
		}
	}()

	if success {
		return r.successJoinClusterSet(ctx, cluster, mc)
	}

	return r.failedJoinClusterSet(ctx, cluster, errorMsg)
}

func getKubeConfig(cluster *mcsv1alpha1.Cluster) (*rest.Config, ctrl.Result, error) {
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

func (r *reconciler) successJoinClusterSet(ctx context.Context, cluster *mcsv1alpha1.Cluster, _ configurator.Configurator) (ctrl.Result, error) {
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
