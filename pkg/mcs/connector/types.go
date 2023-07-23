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
	"github.com/flomesh-io/fsm/pkg/announcements"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	multiclusterClientset "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	conn "github.com/flomesh-io/fsm/pkg/mcs/context"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Connector struct {
	context      context.Context
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
	cfg          *configurator.Client
	broker       *messaging.Broker
}

var (
	log = logger.New("mcs-connector")
)

func NewConnector(ctx context.Context, broker *messaging.Broker) (*Connector, error) {
	connectorCtx := ctx.(*conn.ConnectorContext)

	kubeClient, err := kubernetes.NewForConfig(connectorCtx.KubeConfig)
	if err != nil {
		return nil, err
	}

	if !version.IsSupportedK8sVersion(kubeClient) {
		err := fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String())
		log.Err(err)

		return nil, err
	}

	configClient, err := configClientset.NewForConfig(connectorCtx.KubeConfig)
	if err != nil {
		return nil, err
	}

	multiclusterClient, err := multiclusterClientset.NewForConfig(connectorCtx.KubeConfig)
	if err != nil {
		return nil, err
	}

	informerCollection, err := informers.NewInformerCollection(meshName, connectorCtx.StopCh,
		informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, fsmMeshConfigName, fsmNamespace),
		informers.WithMultiClusterClient(multiclusterClient),
	)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}

	mc := configurator.NewConfigurator(informerCollection, fsmNamespace, fsmMeshConfigName, broker)

	connector := &Connector{
		context:      connectorCtx,
		kubeClient:   kubeClient,
		configClient: configClient,
		cfg:          mc,
		broker:       broker,
	}

	for _, informerKey := range []informers.InformerKey{
		informers.InformerKeyServiceExport,
	} {
		if eventTypes := getEventTypesByInformerKey(informerKey); eventTypes != nil {
			informerCollection.AddEventHandler(informerKey, connector.getEventHandlerFuncs(eventTypes))
		}
	}

	// checks if fsm is installed in the cluster, this's a MUST otherwise it doesn't work
	_, err = kubeClient.AppsV1().
		Deployments(mc.GetFSMNamespace()).
		Get(context.TODO(), constants.FSMControllerName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error().Msgf("FSM Control Plane is not installed or not in a proper state, please check it.")
			return nil, err
		}

		log.Error().Msgf("Get FSM controller component %s/%s error: %s", mc.GetFSMNamespace(), constants.FSMControllerName, err)
		return nil, err
	}

	return connector, nil
}

func getEventTypesByInformerKey(informerKey informers.InformerKey) *k8s.EventTypes {
	switch informerKey {
	case informers.InformerKeyService:
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
			c.onUpdateFunc(eventTypes)(nil, obj)
		}
	}
}

func (c *Connector) onUpdateFunc(eventTypes *k8s.EventTypes) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		switch obj := newObj.(type) {
		case *mcsv1alpha1.ServiceExport:
			connectorCtx := c.context.(*conn.ConnectorContext)
			connectorConfig := connectorCtx.ConnectorConfig

			if !c.cfg.IsManaged() {
				log.Warn().Msgf("[%s] Cluster is not managed, ignore processing ServiceExport %s", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			svc, err := c.getService(obj)
			if err != nil {
				log.Error().Msgf("[%s] Ignore processing ServiceExport %s due to get service failed", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			c.broker.GetQueue().AddRateLimited(events.PubSubMessage{
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

func (c *Connector) onDeleteFunc(eventTypes *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		switch obj := obj.(type) {
		case *mcsv1alpha1.ServiceExport:
			connectorCtx := c.context.(*conn.ConnectorContext)
			connectorConfig := connectorCtx.ConnectorConfig

			if !c.cfg.IsManaged() {
				log.Warn().Msgf("[%s] Cluster is not managed, ignore processing ServiceExport %s", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			svc, err := c.getService(obj)
			if err != nil {
				log.Error().Msgf("[%s] Ignore processing ServiceExport %s due to get service failed", connectorConfig.Key(), client.ObjectKeyFromObject(obj))
				return
			}

			c.broker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   announcements.MultiClusterServiceExportDeleted,
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

func (c *Connector) getService(export *mcsv1alpha1.ServiceExport) (*corev1.Service, error) {
	connectorCtx := c.context.(*conn.ConnectorContext)
	connectorConfig := connectorCtx.ConnectorConfig
	log.Info().Msgf("[%s] Getting service %s/%s", connectorConfig.Key(), export.Namespace, export.Name)

	svc, err := c.kubeClient.CoreV1().
		Services(export.Namespace).
		Get(context.TODO(), export.Name, metav1.GetOptions{})

	if err != nil {
		log.Error().Msgf("[%s] Failed to get svc %s/%s, %s", connectorConfig.Key(), export.Namespace, export.Name, err)
		return nil, err
	}

	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		msg := fmt.Sprintf("[%s] ExternalName service %s/%s cannot be exported", connectorConfig.Key(), export.Namespace, export.Name)
		log.Error().Msgf(msg)
		return nil, fmt.Errorf(msg)
	}

	return svc, nil
}
