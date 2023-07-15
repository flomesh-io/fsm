package pipy

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logger.New("controller-gatewayapi")
)

// NewGatewayAPIController returns a gateway.Controller interface related to functionality provided by the resources in the plugin.flomesh.io API group
func NewGatewayAPIController(informerCollection *fsminformers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator) Controller {
	return newClient(informerCollection, kubeClient, msgBroker, cfg)
}
func newClient(informerCollection *fsminformers.InformerCollection, kubeClient kubernetes.Interface, msgBroker *messaging.Broker, cfg configurator.Configurator) *client {
	c := &client{
		informers:  informerCollection,
		kubeClient: kubeClient,
		msgBroker:  msgBroker,
		cfg:        cfg,
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

// Initializes Service monitoring
func (c *client) initServicesMonitor(informerKey fsminformers.InformerKey) {
	eventTypes := k8s.EventTypes{
		Add:    announcements.ServiceAdded,
		Update: announcements.ServiceUpdated,
		Delete: announcements.ServiceDeleted,
	}
	c.informers.AddEventHandler(informerKey, getEventHandlerFuncs(c.shouldObserve, eventTypes, c.msgBroker))
}

func (c *client) Start() error {
	// Start broadcast listener thread
	//s := repo.NewServer(c.cfg, c.msgBroker, c.cache)
	//go s.BroadcastListener()

	return nil
}
