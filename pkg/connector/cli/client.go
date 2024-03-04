package cli

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/announcements"
	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

// NewConnectorController returns a new Connector.Controller which means to provide access to locally-cached connector resources
func NewConnectorController(provider, connector string,
	context context.Context,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	configClient configClientset.Interface,
	machineClient machineClientset.Interface,
	informerCollection *fsminformers.InformerCollection,
	msgBroker *messaging.Broker,
	selectInformers ...InformerKey) connector.ConnectorController {
	return newClient(provider, connector, context, kubeConfig, kubeClient, configClient, machineClient, informerCollection, msgBroker, selectInformers...)
}

func newClient(provider, connector string,
	context context.Context,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	configClient configClientset.Interface,
	machineClient machineClientset.Interface,
	informerCollection *fsminformers.InformerCollection,
	msgBroker *messaging.Broker,
	selectInformers ...InformerKey) *client {
	// Initialize client object
	c := &client{
		connectorProvider: provider,
		connectorName:     connector,

		context:       context,
		kubeConfig:    kubeConfig,
		kubeClient:    kubeClient,
		configClient:  configClient,
		machineClient: machineClient,

		informers:         informerCollection,
		msgBroker:         msgBroker,
		msgWorkerPoolSize: 0,
		msgWorkQueues:     workerpool.NewWorkerPool(0),
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]func(){
		ConsulConnectors:  c.initConsulConnectorMonitor,
		EurekaConnectors:  c.initEurekaConnectorMonitor,
		NacosConnectors:   c.initNacosConnectorMonitor,
		MachineConnectors: c.initMachineConnectorMonitor,
		GatewayConnectors: c.initGatewayConnectorMonitor,
	}

	// If specific informers are not selected to be initialized, initialize all informers
	if len(selectInformers) == 0 {
		selectInformers = []InformerKey{
			ConsulConnectors,
			EurekaConnectors,
			NacosConnectors,
			MachineConnectors,
			GatewayConnectors}
	}

	for _, informer := range selectInformers {
		informerInitHandlerMap[informer]()
	}

	return c
}

func (c *client) initConsulConnectorMonitor() {
	consulConnectorEventTypes := k8s.EventTypes{
		Add:    announcements.ConsulConnectorAdded,
		Update: announcements.ConsulConnectorUpdated,
		Delete: announcements.ConsulConnectorDeleted,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyConsulConnector,
		k8s.GetEventHandlerFuncs(nil, consulConnectorEventTypes, c.msgBroker))
}

func (c *client) initEurekaConnectorMonitor() {
	eurekaConnectorEventTypes := k8s.EventTypes{
		Add:    announcements.EurekaConnectorAdded,
		Update: announcements.EurekaConnectorUpdated,
		Delete: announcements.EurekaConnectorDeleted,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyEurekaConnector,
		k8s.GetEventHandlerFuncs(nil, eurekaConnectorEventTypes, c.msgBroker))
}

func (c *client) initNacosConnectorMonitor() {
	nacosConnectorEventTypes := k8s.EventTypes{
		Add:    announcements.NacosConnectorAdded,
		Update: announcements.NacosConnectorUpdated,
		Delete: announcements.NacosConnectorDeleted,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyNacosConnector,
		k8s.GetEventHandlerFuncs(nil, nacosConnectorEventTypes, c.msgBroker))
}

func (c *client) initMachineConnectorMonitor() {
	machineConnectorEventTypes := k8s.EventTypes{
		Add:    announcements.MachineConnectorAdded,
		Update: announcements.MachineConnectorUpdated,
		Delete: announcements.MachineConnectorDeleted,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyMachineConnector,
		k8s.GetEventHandlerFuncs(nil, machineConnectorEventTypes, c.msgBroker))
}

func (c *client) initGatewayConnectorMonitor() {
	gatewayConnectorEventTypes := k8s.EventTypes{
		Add:    announcements.GatewayConnectorAdded,
		Update: announcements.GatewayConnectorUpdated,
		Delete: announcements.GatewayConnectorDeleted,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyGatewayConnector,
		k8s.GetEventHandlerFuncs(nil, gatewayConnectorEventTypes, c.msgBroker))
}

// GetConsulConnector returns a ConsulConnector resource if found, nil otherwise.
func (c *client) GetConsulConnector(connector string) *connectorv1alpha1.ConsulConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyConsulConnector, connector)
	if exists && err == nil {
		return connectorIf.(*connectorv1alpha1.ConsulConnector)
	}
	return nil
}

// GetEurekaConnector returns a EurekaConnector resource if found, nil otherwise.
func (c *client) GetEurekaConnector(connector string) *connectorv1alpha1.EurekaConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyEurekaConnector, connector)
	if exists && err == nil {
		return connectorIf.(*connectorv1alpha1.EurekaConnector)
	}
	return nil
}

// GetNacosConnector returns a NacosConnector resource if found, nil otherwise.
func (c *client) GetNacosConnector(connector string) *connectorv1alpha1.NacosConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyNacosConnector, connector)
	if exists && err == nil {
		return connectorIf.(*connectorv1alpha1.NacosConnector)
	}
	return nil
}

// GetMachineConnector returns a MachineConnector resource if found, nil otherwise.
func (c *client) GetMachineConnector(connector string) *connectorv1alpha1.MachineConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyMachineConnector, connector)
	if exists && err == nil {
		return connectorIf.(*connectorv1alpha1.MachineConnector)
	}
	return nil
}

// GetGatewayConnector returns a GatewayConnector resource if found, nil otherwise.
func (c *client) GetGatewayConnector(connector string) *connectorv1alpha1.GatewayConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyGatewayConnector, connector)
	if exists && err == nil {
		return connectorIf.(*connectorv1alpha1.GatewayConnector)
	}
	return nil
}

// GetConnector returns a Connector resource if found, nil otherwise.
func (c *client) GetConnector() (spec interface{}, uid string, ok bool) {
	switch c.GetConnectorProvider() {
	case connectorv1alpha1.ConsulDiscoveryService:
		if connector := c.GetConsulConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case connectorv1alpha1.EurekaDiscoveryService:
		if connector := c.GetEurekaConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case connectorv1alpha1.NacosDiscoveryService:
		if connector := c.GetNacosConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case connectorv1alpha1.MachineDiscoveryService:
		if connector := c.GetMachineConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case connectorv1alpha1.GatewayDiscoveryService:
		if connector := c.GetGatewayConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	default:
	}
	return
}

// GetConnectorProvider returns connector provider.
func (c *client) GetConnectorProvider() connectorv1alpha1.DiscoveryServiceProvider {
	return connectorv1alpha1.DiscoveryServiceProvider(c.connectorProvider)
}

// GetConnectorName returns connector name.
func (c *client) GetConnectorName() string {
	return c.connectorName
}
