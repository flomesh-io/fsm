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

// Package remote contains the remote connector for the FSM multi-cluster
package remote

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/mcs/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/announcements"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	multiclusterClientset "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	cctx "github.com/flomesh-io/fsm/pkg/mcs/context"
	conn "github.com/flomesh-io/fsm/pkg/mcs/context"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/version"
)

// Connector is the main struct for the remote connector
type Connector struct {
	context            context.Context
	kubeClient         kubernetes.Interface
	configClient       configClientset.Interface
	mcsClient          multiclusterClientset.Interface
	cfg                *configurator.Client
	controlPlaneBroker *messaging.Broker
}

// Background is the background struct for the remote connector
type Background struct {
	Context   *cctx.ConnectorContext
	Connector *Connector
}

var (
	log = logger.New("mcs-connector")
)

// NewConnector creates a new remote connector
func NewConnector(ctx context.Context, controlPlaneBroker *messaging.Broker) (*Connector, error) {
	connectorCtx := ctx.(*conn.ConnectorContext)
	stop := connectorCtx.StopCh
	kubeConfig := connectorCtx.KubeConfig
	clusterKey := connectorCtx.ClusterKey
	fsmNamespace := connectorCtx.FsmNamespace
	fsmMeshConfigName := connectorCtx.FsmMeshConfigName

	log.Debug().Msgf("Creating remote connector for cluster %s, fsmNamespace=%s, fsmMeshConfigName=%s", clusterKey, fsmNamespace, fsmMeshConfigName)

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	if !version.IsSupportedK8sVersion(kubeClient) {
		err := fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String())
		log.Err(err)

		return nil, err
	}

	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	multiclusterClient, err := multiclusterClientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	workerBroker := messaging.NewBroker(stop)

	informerCollection, err := informers.NewInformerCollection(clusterKey, stop,
		//informers.WithKubeClientWithoutNamespace(kubeClient),
		informers.WithConfigClient(configClient, fsmMeshConfigName, fsmNamespace),
		informers.WithMultiClusterClient(multiclusterClient),
	)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}

	mc := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, workerBroker)

	log.Debug().Msgf("Checking if FSM Control Plane is installed in cluster %s ...", clusterKey)
	// checks if fsm is installed in the cluster, this's a MUST otherwise it doesn't work
	_, err = kubeClient.AppsV1().
		Deployments(mc.GetFSMNamespace()).
		Get(context.TODO(), constants.FSMControllerName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error().Msgf("[%s] FSM Control Plane is not installed or not in a proper state, please check it.", clusterKey)
			return nil, err
		}

		log.Error().Msgf("[%s] Get FSM controller component %s/%s error: %s", clusterKey, mc.GetFSMNamespace(), constants.FSMControllerName, err)
		return nil, err
	}

	if err := updateConfigsOfManagedCluster(configClient, connectorCtx.ConnectorConfig, mc); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error updating mesh config of managed cluster %s", clusterKey)
		return nil, err
	}

	// wait for the config to be updated
	log.Debug().Msgf("[%s] Waiting for the config to be updated ...", clusterKey)
	time.Sleep(1 * time.Second)

	connector := &Connector{
		context:            connectorCtx,
		kubeClient:         kubeClient,
		configClient:       configClient,
		mcsClient:          multiclusterClient,
		cfg:                mc,
		controlPlaneBroker: controlPlaneBroker,
	}

	for _, informerKey := range []informers.InformerKey{
		informers.InformerKeyServiceExport,
	} {
		if eventTypes := getEventTypesByInformerKey(informerKey); eventTypes != nil {
			informerCollection.AddEventHandler(informerKey, connector.getEventHandlerFuncs(eventTypes))
		}
	}

	return connector, nil
}

func updateConfigsOfManagedCluster(configClient configClientset.Interface, connectorCfg *config.ConnectorConfig, cfg configurator.Configurator) error {
	log.Debug().Msgf("[%s] updating config .... ", connectorCfg.Key())

	if cfg.IsManaged() && cfg.GetMultiClusterControlPlaneUID() != "" {
		if cfg.GetMultiClusterControlPlaneUID() != connectorCfg.ControlPlaneUID() {
			return fmt.Errorf("cluster %s is already managed, cannot join the MultiCluster", connectorCfg.Key())
		}

		log.Debug().Msgf("[%s] Rejoining ClusterSet ...", connectorCfg.Key())
	} else {
		mc := cfg.GetMeshConfig()
		mc.Spec.ClusterSet.IsManaged = true
		mc.Spec.ClusterSet.Region = connectorCfg.Region()
		mc.Spec.ClusterSet.Zone = connectorCfg.Zone()
		mc.Spec.ClusterSet.Group = connectorCfg.Group()
		mc.Spec.ClusterSet.Name = connectorCfg.Name()
		mc.Spec.ClusterSet.ControlPlaneUID = connectorCfg.ControlPlaneUID()

		if _, err := configClient.ConfigV1alpha3().
			MeshConfigs(mc.Namespace).
			Update(context.TODO(), &mc, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func getEventTypesByInformerKey(informerKey informers.InformerKey) *k8s.EventTypes {
	switch informerKey {
	case informers.InformerKeyServiceExport:
		return &k8s.EventTypes{
			Add:    announcements.ServiceExportAdded,
			Update: announcements.ServiceExportUpdated,
			Delete: announcements.ServiceExportDeleted,
		}
	}

	return nil
}

func (c *Connector) getEventHandlerFuncs(eventTypes *k8s.EventTypes) k8scache.ResourceEventHandlerFuncs {
	return k8scache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAddFunc(eventTypes),
		UpdateFunc: c.onUpdateFunc(eventTypes),
		DeleteFunc: c.onDeleteFunc(eventTypes),
	}
}

func (c *Connector) onAddFunc(eventTypes *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		switch obj := obj.(type) {
		case *mcsv1alpha1.ServiceExport:
			connectorCtx := c.context.(*conn.ConnectorContext)
			connectorConfig := connectorCtx.ConnectorConfig

			log.Debug().Msgf("[%s] ServiceExport %s added", connectorConfig.Key(), client.ObjectKeyFromObject(obj))

			c.onUpdateFunc(eventTypes)(nil, obj)
		}
	}
}

func (c *Connector) onUpdateFunc(_ *k8s.EventTypes) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		switch obj := newObj.(type) {
		case *mcsv1alpha1.ServiceExport:
			connectorCtx := c.context.(*conn.ConnectorContext)
			connectorConfig := connectorCtx.ConnectorConfig

			log.Debug().Msgf("[%s] ServiceExport %s updated", connectorConfig.Key(), client.ObjectKeyFromObject(obj))

			if !c.cfg.IsManaged() {
				log.Warn().Msgf("[%s] Cluster is not managed, ignore processing ServiceExport %s", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			svc, err := c.getService(obj)
			if err != nil {
				log.Error().Msgf("[%s] Ignore processing ServiceExport %s due to get service failed", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			c.controlPlaneBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   announcements.MultiClusterServiceExportCreated,
				OldObj: nil,
				NewObj: &mcsevent.ServiceExportEvent{
					Geo:           connectorConfig,
					ServiceExport: obj,
					Service:       svc,
				},
			})
		}
	}
}

func (c *Connector) onDeleteFunc(_ *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		switch obj := obj.(type) {
		case *mcsv1alpha1.ServiceExport:
			connectorCtx := c.context.(*conn.ConnectorContext)
			connectorConfig := connectorCtx.ConnectorConfig

			log.Debug().Msgf("[%s] ServiceExport %s deleted", connectorConfig.Key(), client.ObjectKeyFromObject(obj))

			if !c.cfg.IsManaged() {
				log.Warn().Msgf("[%s] Cluster is not managed, ignore processing ServiceExport %s", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			svc, err := c.getService(obj)
			if err != nil {
				log.Error().Msgf("[%s] Ignore processing ServiceExport %s due to get service failed", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			c.controlPlaneBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   announcements.MultiClusterServiceExportDeleted,
				NewObj: nil,
				OldObj: &mcsevent.ServiceExportEvent{
					Geo:           connectorConfig,
					ServiceExport: obj,
					Service:       svc,
				},
			})
		}
	}
}

const externalNameServiceErrorMsg = "[%s] ExternalName service %s/%s cannot be exported"

func (c *Connector) getService(export *mcsv1alpha1.ServiceExport) (*corev1.Service, error) {
	connectorCtx := c.context.(*conn.ConnectorContext)
	connectorConfig := connectorCtx.ConnectorConfig
	log.Debug().Msgf("[%s] Getting service %s/%s", connectorConfig.Key(), export.Namespace, export.Name)

	svc, err := c.kubeClient.CoreV1().
		Services(export.Namespace).
		Get(context.TODO(), export.Name, metav1.GetOptions{})

	if err != nil {
		log.Error().Msgf("[%s] Failed to get svc %s/%s, %s", connectorConfig.Key(), export.Namespace, export.Name, err)
		return nil, err
	}

	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		log.Error().Msg(fmt.Sprintf(externalNameServiceErrorMsg, connectorConfig.Key(), export.Namespace, export.Name))
		return nil, fmt.Errorf(externalNameServiceErrorMsg, connectorConfig.Key(), export.Namespace, export.Name)
	}

	return svc, nil
}
