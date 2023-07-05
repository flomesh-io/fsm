package gateway

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/gateway/cache"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logger.New("controller-gatewayapi")
)

// NewGatewayAPIController returns a gateway.Controller interface related to functionality provided by the resources in the plugin.flomesh.io API group
func NewGatewayAPIController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker) Controller {
	return newClient(informerCollection, kubeClient, msgBroker)
}
func newClient(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker) *Client {
	c := &Client{
		informers:  informerCollection,
		kubeClient: kubeClient,
		msgBroker:  msgBroker,
		cache:      cache.NewGatewayCache(informerCollection),
	}

	// Initialize informers
	informerInitHandlerMap := map[fsminformers.InformerKey]func(fsminformers.InformerKey){
		fsminformers.InformerKeyService:                c.initServicesMonitor,
		fsminformers.InformerKeyEndpointSlices:         c.initEndpointSlicesMonitor,
		fsminformers.InformerKeySecret:                 c.initSecretMonitor,
		fsminformers.InformerKeyGatewayApiGatewayClass: c.initGatewayClassesMonitor,
		fsminformers.InformerKeyGatewayApiGateway:      c.initGatewaysMonitor,
		fsminformers.InformerKeyGatewayApiHTTPRoute:    c.initHTTPRoutesMonitor,
		fsminformers.InformerKeyGatewayApiGRPCRoute:    c.initGRPCRoutesMonitor,
		fsminformers.InformerKeyGatewayApiTLSRoute:     c.initTLSRoutesMonitor,
		fsminformers.InformerKeyGatewayApiTCPRoute:     c.initTCPRoutesMonitor,
	}

	for key, initFunc := range informerInitHandlerMap {
		initFunc(key)
	}

	return c
}

func (c *Client) shouldObserveUpsert(obj interface{}) bool {
	return c.cache.Insert(obj)
}

func (c *Client) shouldObserveDelete(obj interface{}) bool {
	return c.cache.Delete(obj)
}

// Initializes Service monitoring
func (c *Client) initServicesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.ServiceAdded,
		Update: announcements.ServiceUpdated,
		Delete: announcements.ServiceDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

// Initializes EndpointSlices monitoring
func (c *Client) initEndpointSlicesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.EndpointSlicesAdded,
		Update: announcements.EndpointSlicesUpdated,
		Delete: announcements.EndpointSlicesDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initSecretMonitor(informerKey informers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.SecretAdded,
		Update: announcements.SecretUpdated,
		Delete: announcements.SecretDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initGatewayClassesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.GatewayApiGatewayClassAdded,
		Update: announcements.GatewayApiGatewayClassUpdated,
		Delete: announcements.GatewayApiGatewayClassDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initGatewaysMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.GatewayApiGatewayAdded,
		Update: announcements.GatewayApiGatewayUpdated,
		Delete: announcements.GatewayApiGatewayDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initHTTPRoutesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.GatewayApiHTTPRouteAdded,
		Update: announcements.GatewayApiHTTPRouteUpdated,
		Delete: announcements.GatewayApiHTTPRouteDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initGRPCRoutesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.GatewayApiGRPCRouteAdded,
		Update: announcements.GatewayApiGRPCRouteUpdated,
		Delete: announcements.GatewayApiGRPCRouteDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initTLSRoutesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.GatewayApiTLSRouteAdded,
		Update: announcements.GatewayApiTLSRouteUpdated,
		Delete: announcements.GatewayApiTLSRouteDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}

func (c *Client) initTCPRoutesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.GatewayApiTCPRouteAdded,
		Update: announcements.GatewayApiTCPRouteUpdated,
		Delete: announcements.GatewayApiTCPRouteDeleted,
	}
	c.informers.AddEventHandler(informerKey, GetEventHandlerFuncs(c.shouldObserveUpsert, c.shouldObserveDelete, eventTypes, c.msgBroker))
}
