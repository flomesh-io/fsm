package gateway

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/gateway/cache"
	"github.com/flomesh-io/fsm/pkg/gateway/repo"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var (
	log = logger.New("controller-gatewayapi")
)

// NewGatewayAPIController returns a gateway.Controller interface related to functionality provided by the resources in the plugin.flomesh.io API group
func NewGatewayAPIController(informerCollection *fsminformers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator) Controller {
	return newClient(informerCollection, kubeClient, msgBroker, cfg)
}
func newClient(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator) *client {
	c := &client{
		informers:  informerCollection,
		kubeClient: kubeClient,
		msgBroker:  msgBroker,
		cfg:        cfg,
		cache:      cache.NewGatewayCache(informerCollection, kubeClient, cfg),
	}

	// Initialize informers
	for _, informerKey := range []fsminformers.InformerKey{
		fsminformers.InformerKeyService,
		fsminformers.InformerKeyEndpointSlices,
		fsminformers.InformerKeySecret,
		fsminformers.InformerKeyGatewayApiGatewayClass,
		fsminformers.InformerKeyGatewayApiGateway,
		fsminformers.InformerKeyGatewayApiHTTPRoute,
		fsminformers.InformerKeyGatewayApiGRPCRoute,
		fsminformers.InformerKeyGatewayApiTLSRoute,
		fsminformers.InformerKeyGatewayApiTCPRoute,
	} {
		if eventTypes := getEventTypesByInformerKey(informerKey); eventTypes != nil {
			c.informers.AddEventHandler(informerKey, c.getEventHandlerFuncs(eventTypes))
		}
	}

	return c
}

func (c *client) getEventHandlerFuncs(eventTypes *k8s.EventTypes) k8scache.ResourceEventHandlerFuncs {
	return k8scache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAddFunc(eventTypes),
		UpdateFunc: c.onUpdateFunc(eventTypes),
		DeleteFunc: c.onDeleteFunc(eventTypes),
	}
}

func (c *client) onAddFunc(eventTypes *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		if !c.shouldObserve(nil, obj) {
			return
		}
		logResourceEvent(log, eventTypes.Add, obj)
		c.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
			Kind:   eventTypes.Add,
			NewObj: obj,
			OldObj: nil,
		})
	}
}

func (c *client) onUpdateFunc(eventTypes *k8s.EventTypes) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if !c.shouldObserve(oldObj, newObj) {
			return
		}
		logResourceEvent(log, eventTypes.Update, newObj)
		c.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
			Kind:   eventTypes.Update,
			NewObj: newObj,
			OldObj: oldObj,
		})
	}
}

func (c *client) onDeleteFunc(eventTypes *k8s.EventTypes) func(obj interface{}) {
	return func(obj interface{}) {
		if !c.shouldObserve(obj, nil) {
			return
		}
		logResourceEvent(log, eventTypes.Delete, obj)
		c.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
			Kind:   eventTypes.Delete,
			NewObj: nil,
			OldObj: obj,
		})
	}
}

func (c *client) shouldObserve(oldObj, newObj interface{}) bool {
	return c.onChange(oldObj, newObj)
}

func (c *client) onChange(oldObj, newObj interface{}) bool {
	if newObj == nil {
		return c.cache.Delete(oldObj)
	} else {
		if oldObj == nil {
			return c.cache.Insert(newObj)
		} else {
			if cmp.Equal(oldObj, newObj) {
				return false
			}

			del := c.cache.Delete(oldObj)
			ins := c.cache.Insert(newObj)

			return del || ins
		}
	}
}

func (c *client) OnAdd(obj interface{}) {
	if eventTypes := getEventTypesByObjectType(obj); eventTypes != nil {
		c.onAddFunc(eventTypes)(obj)
	}
}

func (c *client) OnUpdate(oldObj, newObj interface{}) {
	if eventTypes := getEventTypesByObjectType(newObj); eventTypes != nil {
		c.onUpdateFunc(eventTypes)(oldObj, newObj)
	}
}

func (c *client) OnDelete(obj interface{}) {
	if eventTypes := getEventTypesByObjectType(obj); eventTypes != nil {
		c.onDeleteFunc(eventTypes)(obj)
	}
}

func getEventTypesByObjectType(obj interface{}) *k8s.EventTypes {
	switch obj.(type) {
	case *corev1.Service:
		return getEventTypesByInformerKey(fsminformers.InformerKeyService)
	case *discoveryv1.EndpointSlice:
		return getEventTypesByInformerKey(fsminformers.InformerKeyEndpointSlices)
	case *corev1.Secret:
		return getEventTypesByInformerKey(fsminformers.InformerKeySecret)
	case *gwv1beta1.GatewayClass:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayApiGatewayClass)
	case *gwv1beta1.Gateway:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayApiGateway)
	case *gwv1beta1.HTTPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayApiHTTPRoute)
	case *gwv1alpha2.GRPCRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayApiGRPCRoute)
	case *gwv1alpha2.TLSRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayApiTLSRoute)
	case *gwv1alpha2.TCPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayApiTCPRoute)
	}

	return nil
}

func getEventTypesByInformerKey(informerKey fsminformers.InformerKey) *k8s.EventTypes {
	switch informerKey {
	case fsminformers.InformerKeyService:
		return &k8s.EventTypes{
			Add:    announcements.ServiceAdded,
			Update: announcements.ServiceUpdated,
			Delete: announcements.ServiceDeleted,
		}
	case fsminformers.InformerKeyEndpointSlices:
		return &k8s.EventTypes{
			Add:    announcements.EndpointSlicesAdded,
			Update: announcements.EndpointSlicesUpdated,
			Delete: announcements.EndpointSlicesDeleted,
		}
	case fsminformers.InformerKeySecret:
		return &k8s.EventTypes{
			Add:    announcements.SecretAdded,
			Update: announcements.SecretUpdated,
			Delete: announcements.SecretDeleted,
		}
	case fsminformers.InformerKeyGatewayApiGatewayClass:
		return &k8s.EventTypes{
			Add:    announcements.GatewayApiGatewayClassAdded,
			Update: announcements.GatewayApiGatewayClassUpdated,
			Delete: announcements.GatewayApiGatewayClassDeleted,
		}
	case fsminformers.InformerKeyGatewayApiGateway:
		return &k8s.EventTypes{
			Add:    announcements.GatewayApiGatewayAdded,
			Update: announcements.GatewayApiGatewayUpdated,
			Delete: announcements.GatewayApiGatewayDeleted,
		}
	case fsminformers.InformerKeyGatewayApiHTTPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayApiHTTPRouteAdded,
			Update: announcements.GatewayApiHTTPRouteUpdated,
			Delete: announcements.GatewayApiHTTPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayApiGRPCRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayApiGRPCRouteAdded,
			Update: announcements.GatewayApiGRPCRouteUpdated,
			Delete: announcements.GatewayApiGRPCRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayApiTLSRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayApiTLSRouteAdded,
			Update: announcements.GatewayApiTLSRouteUpdated,
			Delete: announcements.GatewayApiTLSRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayApiTCPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayApiTCPRouteAdded,
			Update: announcements.GatewayApiTCPRouteUpdated,
			Delete: announcements.GatewayApiTCPRouteDeleted,
		}
	}

	return nil
}

func (c *client) Start() error {
	// Start broadcast listener thread
	s := repo.NewServer(c.cfg, c.msgBroker, c.cache)
	go s.BroadcastListener()

	return nil
}

func logResourceEvent(parent zerolog.Logger, event announcements.Kind, obj interface{}) {
	log := parent.With().Str("event", event.String()).Logger()
	o, err := meta.Accessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("error parsing object, ignoring")
		return
	}
	name := o.GetName()
	if o.GetNamespace() != "" {
		name = o.GetNamespace() + "/" + name
	}
	log.Debug().Str("resource_name", name).Msg("received kubernetes resource event")
}
