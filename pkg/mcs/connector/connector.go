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

package connector

import (
	"context"
	"fmt"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	conn "github.com/flomesh-io/fsm/pkg/mcs/context"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	retry "github.com/sethvargo/go-retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"net"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func (c *Connector) Run(stopCh <-chan struct{}) error {
	ctx := c.context.(*conn.ConnectorContext)
	connectorCfg := ctx.ConnectorConfig
	errCh := make(chan error)

	err := c.updateConfigsOfManagedCluster()
	if err != nil {
		return err
	}

	if c.cache.GetBroadcaster() != nil && c.k8sAPI.EventClient != nil {
		klog.V(3).Infof("[%s] Starting broadcaster ......", connectorCfg.Key())
		c.cache.GetBroadcaster().StartRecordingToSink(stopCh)
	}

	// register event handlers
	klog.V(3).Infof("[%s] Registering event handlers ......", connectorCfg.Key())
	controllers := c.cache.GetControllers()
	go controllers.ServiceExport.Run(stopCh)

	// start the ServiceExport Informer
	klog.V(3).Infof("[%s] Starting ServiceExport informer ......", connectorCfg.Key())
	go controllers.ServiceExport.Informer.Run(stopCh)
	if !k8scache.WaitForCacheSync(stopCh, controllers.ServiceExport.HasSynced) {
		runtime.HandleError(fmt.Errorf("[%s] timed out waiting for ServiceExport to sync", connectorCfg.Key()))
	}

	// Sleep for a while, so that there's enough time for processing
	klog.V(5).Infof("[%s] Sleep for a while ......", connectorCfg.Key())
	time.Sleep(1 * time.Second)

	// register event handler
	mc := c.clusterCfg.MeshConfig.GetConfig()
	if mc.IsManaged {
		go c.processEvent(c.broker, stopCh)
	}

	// start the cache runner
	go c.cache.SyncLoop(stopCh)

	return <-errCh
}

func (c *Connector) updateConfigsOfManagedCluster() error {
	ctx := c.context.(*conn.ConnectorContext)
	connectorCfg := ctx.ConnectorConfig
	klog.V(5).Infof("[%s] updating config .... ", connectorCfg.Key())

	//if !connectorCfg.IsInCluster() {
	mcClient := c.clusterCfg.MeshConfig
	mc := mcClient.GetConfig()

	if mc.IsManaged && mc.Cluster.ControlPlaneUID != "" {
		if mc.Cluster.ControlPlaneUID != connectorCfg.ControlPlaneUID() {
			return fmt.Errorf("cluster %s is already managed, cannot join the MultiCluster", connectorCfg.Key())
		} else {
			klog.Infof("[%s] Rejoining ClusterSet ...", connectorCfg.Key())
		}
	} else {
		mc.IsManaged = true
		mc.Cluster.Region = connectorCfg.Region()
		mc.Cluster.Zone = connectorCfg.Zone()
		mc.Cluster.Group = connectorCfg.Group()
		mc.Cluster.Name = connectorCfg.Name()
		mc.Cluster.ControlPlaneUID = connectorCfg.ControlPlaneUID()

		if _, err := mcClient.UpdateConfig(mc); err != nil {
			return err
		}

	}
	//}

	return nil
}

func (c *Connector) processEvent(broker *mcsevent.Broker, stopCh <-chan struct{}) {
	ctx := c.context.(*conn.ConnectorContext)
	connectorCfg := ctx.ConnectorConfig
	klog.V(5).Infof("[%s] start to processing events .... ", connectorCfg.Key())

	msgBus := broker.GetMessageBus()

	svcExportDeletedCh := msgBus.Sub(string(mcsevent.ServiceExportDeleted))
	defer broker.Unsub(msgBus, svcExportDeletedCh)
	svcExportAcceptedCh := msgBus.Sub(string(mcsevent.ServiceExportAccepted))
	defer broker.Unsub(msgBus, svcExportAcceptedCh)
	svcExportRejectedCh := msgBus.Sub(string(mcsevent.ServiceExportRejected))
	defer broker.Unsub(msgBus, svcExportRejectedCh)

	for {
		// FIXME: refine it later
		select {
		case msg, ok := <-svcExportDeletedCh:
			if !ok {
				klog.Warningf("[%s] Channel closed for ServiceExport", connectorCfg.Key())
				continue
			}
			klog.V(5).Infof("[%s] received event ServiceExportDeleted %v", connectorCfg.Key(), msg)

			e, ok := msg.(mcsevent.Message)
			if !ok {
				klog.Errorf("[%s] Received unexpected message %T on channel, expected Message", connectorCfg.Key(), e)
				continue
			}

			svcExportEvt, ok := e.OldObj.(*mcsevent.ServiceExportEvent)
			if !ok {
				klog.Errorf("[%s] Received unexpected object %T, expected *mcsevent.ServiceExportEvent", connectorCfg.Key(), svcExportEvt)
				continue
			}

			go func() {
				if err := retry.Fibonacci(c.context, 1*time.Second, func(ctx context.Context) error {
					if err := c.deleteServiceImport(svcExportEvt); err != nil {
						// This marks the error as retryable
						return retry.RetryableError(err)
					}

					return nil
				}); err != nil {
					klog.Errorf("[%s] Failed to delete ServiceImport %s/%s", connectorCfg.Key(), svcExportEvt.ServiceExport.Namespace, svcExportEvt.ServiceExport.Name)
				}
			}()
		case msg, ok := <-svcExportAcceptedCh:
			if !ok {
				klog.Warningf("[%s] Channel closed for ServiceExport", connectorCfg.Key())
				continue
			}
			klog.V(5).Infof("[%s] received event ServiceExportAccepted %v", connectorCfg.Key(), msg)

			e, ok := msg.(mcsevent.Message)
			if !ok {
				klog.Errorf("[%s] Received unexpected message %T on channel, expected Message", connectorCfg.Key(), e)
				continue
			}

			svcExportEvt, ok := e.NewObj.(*mcsevent.ServiceExportEvent)
			if !ok {
				klog.Errorf("[%s] Received unexpected object %T, expected *mcsevent.ServiceExportEvent", connectorCfg.Key(), svcExportEvt)
				continue
			}

			go func() {
				if err := retry.Fibonacci(c.context, 1*time.Second, func(ctx context.Context) error {
					if err := c.upsertServiceImport(svcExportEvt); err != nil {
						// This marks the error as retryable
						return retry.RetryableError(err)
					}

					return nil
				}); err != nil {
					klog.Errorf("[%s] Failed to upsert ServiceImport %s/%s", connectorCfg.Key(), svcExportEvt.ServiceExport.Namespace, svcExportEvt.ServiceExport.Name)
				}
			}()
		case msg, ok := <-svcExportRejectedCh:
			if !ok {
				klog.Warningf("[%s] Channel closed for ServiceExport", connectorCfg.Key())
				continue
			}
			klog.V(5).Infof("[%s] received event ServiceExportRejected %v", connectorCfg.Key(), msg)

			e, ok := msg.(mcsevent.Message)
			if !ok {
				klog.Errorf("[%s] Received unexpected message %T on channel, expected Message", connectorCfg.Key(), e)
				continue
			}

			svcExportEvt, ok := e.NewObj.(*mcsevent.ServiceExportEvent)
			if !ok {
				klog.Errorf("[%s] Received unexpected object %T, expected *mcsevent.ServiceExportEvent", connectorCfg.Key(), svcExportEvt)
				continue
			}

			go func() {
				if err := retry.Fibonacci(c.context, 1*time.Second, func(ctx context.Context) error {
					if err := c.rejectServiceExport(svcExportEvt); err != nil {
						// This marks the error as retryable
						return retry.RetryableError(err)
					}

					return nil
				}); err != nil {
					klog.Errorf("[%s] Failed to handle Reject Event of ServiceExport %s/%s: %s", connectorCfg.Key(), svcExportEvt.ServiceExport.Namespace, svcExportEvt.ServiceExport.Name, err)
				}
			}()
		case <-stopCh:
			klog.Infof("[%s] Received stop signal.", connectorCfg.Key())
			return
		}
	}
}

func (c *Connector) ServiceImportExists(svcExp *mcsv1alpha1.ServiceExport) bool {
	ctx := c.context.(*conn.ConnectorContext)

	if _, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Get(context.TODO(), svcExp.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			klog.V(5).Infof("[%s] ServiceImport %s/%s doesn't exist", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
			return false
		}

		klog.Errorf("[%s] Failed to get of ServiceImport %s/%s: %s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, err)
		return true
	}

	klog.V(5).Infof("[%s] ServiceImport %s/%s already exists", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
	return true
}

func (c *Connector) ValidateServiceExport(svcExp *mcsv1alpha1.ServiceExport, service *corev1.Service) error {
	ctx := c.context.(*conn.ConnectorContext)
	clusterKey := ctx.ClusterKey
	localSvc, err := c.kubeClient.CoreV1().
		Services(svcExp.Namespace).
		Get(context.TODO(), svcExp.Name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			// If not found this svc in the cluster, then there' no conflict possibility
			klog.V(5).Infof("[%s] Service %s/%s doesn't exist, no conflict", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
			return nil
		}
		return fmt.Errorf("[%s] Failed get Service %s/%s: %s", clusterKey, svcExp.Namespace, svcExp.Name, err)
	}

	if service.Spec.Type != localSvc.Spec.Type {
		return fmt.Errorf("[%s] service type doesn't match: %s vs %s", clusterKey, service.Spec.Type, localSvc.Spec.Type)
	}

	if !reflect.DeepEqual(service.Spec.Ports, localSvc.Spec.Ports) {
		return fmt.Errorf("[%s] spec.ports conflict, please check service spec", clusterKey)
	}

	return nil
}

func (c *Connector) upsertServiceImport(export *mcsevent.ServiceExportEvent) error {
	ctx := c.context.(*conn.ConnectorContext)
	exportClusterKey := export.ClusterKey()
	svcExp := export.ServiceExport
	if exportClusterKey == ctx.ClusterKey {
		klog.Warningf("[%s] ServiceExport %s/%s is ignored as it occurs in same cluster", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
		return nil
	}

	imp, err := c.getOrCreateServiceImport(export)
	if err != nil {
		return err
	}
	klog.V(5).Infof("[%s] Created/Found ServiceImport %s/%s: %v", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, imp)

	//ports := make([]svcimpv1alpha1.ServicePort, 0)
	for idx, p := range imp.Spec.Ports {
		klog.V(5).Infof("[%s] processing port %d, len(endpoints)=%d", ctx.ClusterKey, p.Port, len(p.Endpoints))
		endpoints := make([]svcimpv1alpha1.Endpoint, 0)
		if len(p.Endpoints) == 0 {
			for _, r := range svcExp.Spec.Rules {
				if r.PortNumber == p.Port {
					ep := newEndpoint(export, r, export.Geo.GatewayHost(), export.Geo.GatewayIP(), export.Geo.GatewayPort())
					klog.V(5).Infof("[%s] processing port %d, ep=%v", ctx.ClusterKey, p.Port, ep)
					endpoints = append(endpoints, ep)
				}
			}
		} else {
			epMap := make(map[string]svcimpv1alpha1.Endpoint)
			for _, r := range svcExp.Spec.Rules {
				if r.PortNumber == p.Port {
					// copy
					for _, ep := range p.Endpoints {
						klog.V(5).Infof("[%s] processing port %d, existing ep=%v", ctx.ClusterKey, p.Port, ep)
						epMap[ep.ClusterKey] = *ep.DeepCopy()
					}

					// insert/update
					epMap[exportClusterKey] = newEndpoint(export, r, export.Geo.GatewayHost(), export.Geo.GatewayIP(), export.Geo.GatewayPort())
				}
			}

			for _, ep := range epMap {
				klog.V(5).Infof("[%s] port %d, endpoint entry=%v", ctx.ClusterKey, p.Port, ep)
				endpoints = append(endpoints, ep)
			}
		}

		imp.Spec.Ports[idx].Endpoints = endpoints
		klog.V(5).Infof("[%s] len of endpoints of port %d is %d", ctx.ClusterKey, p.Port, len(imp.Spec.Ports[idx].Endpoints))
	}
	imp.Spec.ServiceAccountName = svcExp.Spec.ServiceAccountName
	klog.V(5).Infof("[%s] After merging, ServiceImport %s/%s: %v", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, imp)

	klog.V(5).Infof("[%s] updating ServiceImport %s/%s ...", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
	if _, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Update(context.TODO(), imp, metav1.UpdateOptions{}); err != nil {
		klog.Errorf("[%s] Failed to update ServiceImport %s/%s: %s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, err)
		return err
	}

	return nil
}

func (c *Connector) getOrCreateServiceImport(export *mcsevent.ServiceExportEvent) (*svcimpv1alpha1.ServiceImport, error) {
	ctx := c.context.(*conn.ConnectorContext)
	svcExp := export.ServiceExport

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: svcExp.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
	}
	if _, err := c.k8sAPI.Client.CoreV1().
		Namespaces().
		Create(context.TODO(), ns, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			klog.V(5).Infof("[%s] Namespace %q exists", ctx.ClusterKey, svcExp.Namespace)
		} else {
			klog.Errorf("[%s] Failed to create Namespace %q: %s", ctx.ClusterKey, svcExp.Namespace, err)
			return nil, err
		}
	}

	imp := c.newServiceImport(export)
	if imp == nil {
		return nil, fmt.Errorf("[%s] Failed to new instance of ServiceImport %s/%s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
	}

	imp, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Create(context.TODO(), imp, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.V(5).Infof("[%s] ServiceImport %s/%s already exists, getting it ...", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
			imp, err = c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
				ServiceImports(svcExp.Namespace).
				Get(context.TODO(), svcExp.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("[%s] Failed to get ServiceImport %s/%s: %s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, err)
				return nil, err
			}

			return imp, nil
		}

		klog.Errorf("[%s] Failed to create ServiceImport %s/%s: %s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, err)
		return nil, err
	}

	klog.V(5).Infof("[%s] ServiceImport %s/%s is created successfully", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
	return imp, nil
}

func (c *Connector) newServiceImport(export *mcsevent.ServiceExportEvent) *svcimpv1alpha1.ServiceImport {
	svcExp := export.ServiceExport
	service := export.Service

	ports := make([]svcimpv1alpha1.ServicePort, 0)
	for _, r := range svcExp.Spec.Rules {
		for _, p := range service.Spec.Ports {
			if r.PortNumber == p.Port {
				ports = append(ports, svcimpv1alpha1.ServicePort{
					Name:        p.Name,
					Port:        p.Port,
					Protocol:    p.Protocol,
					AppProtocol: p.AppProtocol,
					Endpoints: []svcimpv1alpha1.Endpoint{
						newEndpoint(export, r, export.Geo.GatewayHost(), export.Geo.GatewayIP(), export.Geo.GatewayPort()),
					},
				})
			}
		}
	}

	return &svcimpv1alpha1.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcExp.Name,
			Namespace: svcExp.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "flomesh.io/v1alpha1",
			Kind:       "ServiceImport",
		},
		Spec: svcimpv1alpha1.ServiceImportSpec{
			Type:               svcimpv1alpha1.ClusterSetIP, // ONLY set the value, there's no any logic to handle the type yet
			Ports:              ports,
			ServiceAccountName: svcExp.Spec.ServiceAccountName,
		},
	}
}

func newEndpoint(export *mcsevent.ServiceExportEvent, r mcsv1alpha1.ServiceExportRule, host string, ip net.IP, port int32) svcimpv1alpha1.Endpoint {
	return svcimpv1alpha1.Endpoint{
		ClusterKey: export.ClusterKey(),
		//Targets: []string{
		//	fmt.Sprintf("%s%s", export.Geo.Gateway(), r.Path),
		//},
		Target: svcimpv1alpha1.Target{
			Host: host,
			IP:   ip.String(),
			Port: port,
			Path: r.Path,
		},
	}
}

func (c *Connector) deleteServiceImport(export *mcsevent.ServiceExportEvent) error {
	ctx := c.context.(*conn.ConnectorContext)
	exportClusterKey := export.ClusterKey()
	svcExp := export.ServiceExport
	if exportClusterKey == ctx.ClusterKey {
		klog.Warningf("[%s] ServiceExport %s/%s is ignored as it occurs in same cluster", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
		return nil
	}

	imp, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
		ServiceImports(svcExp.Namespace).
		Get(context.TODO(), svcExp.Name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			klog.Warningf("[%s] ServiceImport %s had been deleted.", ctx.ClusterKey, client.ObjectKeyFromObject(svcExp))
			return nil
		}

		return err
	}

	if imp.DeletionTimestamp != nil {
		klog.Warningf("[%s] ServiceImport %s/%s is being deleted, ignore it", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
		return nil
	}

	// update service import, remove the export entry
	ports := make([]svcimpv1alpha1.ServicePort, 0)
	for _, r := range svcExp.Spec.Rules {
		for _, p := range imp.Spec.Ports {
			if r.PortNumber == p.Port {
				endpoints := make([]svcimpv1alpha1.Endpoint, 0)
				for _, ep := range p.Endpoints {
					if ep.ClusterKey == exportClusterKey {
						continue
					} else {
						endpoints = append(endpoints, *ep.DeepCopy())
					}
				}

				if len(endpoints) > 0 {
					p.Endpoints = endpoints
					ports = append(ports, *p.DeepCopy())
				}
			}
		}
	}

	if len(ports) > 0 {
		imp.Spec.Ports = ports
		if _, err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
			ServiceImports(svcExp.Namespace).
			Update(context.TODO(), imp, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("[%s] Failed to update ServiceImport %s/%s: %s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, err)
			return err
		}
		klog.V(5).Infof("[%s] ServiceImport %s/%s is updated successfully", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
	} else {
		if err := c.k8sAPI.FlomeshClient.ServiceimportV1alpha1().
			ServiceImports(svcExp.Namespace).
			Delete(context.TODO(), svcExp.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("[%s] Failed to delete ServiceImport %s/%s: %s", ctx.ClusterKey, svcExp.Namespace, svcExp.Name, err)
			return err
		}
		klog.V(5).Infof("[%s] ServiceImport %s/%s is deleted successfully", ctx.ClusterKey, svcExp.Namespace, svcExp.Name)
	}

	return nil
}

func (c *Connector) rejectServiceExport(svcExportEvt *mcsevent.ServiceExportEvent) error {
	ctx := c.context.(*conn.ConnectorContext)
	export := svcExportEvt.ServiceExport
	//reason := svcExportEvt.Data["reason"]
	reason := svcExportEvt.Error

	if ctx.ClusterKey == svcExportEvt.ClusterKey() {
		exp, err := c.k8sAPI.FlomeshClient.ServiceexportV1alpha1().
			ServiceExports(export.Namespace).
			Get(context.TODO(), export.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("[%s] Failed to get ServiceExport %s/%s: %s", ctx.ClusterKey, export.Namespace, export.Name, err)
			return err
		}

		c.cache.GetRecorder().Eventf(exp, nil, corev1.EventTypeWarning, "Rejected", "ServiceExport %s/%s is invalid, %s", exp.Namespace, exp.Name, reason)

		metautil.SetStatusCondition(&exp.Status.Conditions, metav1.Condition{
			Type:               string(mcsv1alpha1.ServiceExportConflict),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: exp.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Conflict",
			Message:            fmt.Sprintf("ServiceExport %s/%s conflicts, %s", exp.Namespace, exp.Name, reason),
		})

		if _, err := c.k8sAPI.FlomeshClient.ServiceexportV1alpha1().
			ServiceExports(export.Namespace).
			UpdateStatus(context.TODO(), exp, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("[%s] Failed to update status of ServiceExport %s/%s: %s", ctx.ClusterKey, exp.Namespace, exp.Name, err)
			return err
		}
	}

	return nil
}
