package cli

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/time/rate"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/announcements"
	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

// NewConnectController returns a new Connector.Controller which means to provide access to locally-cached connector resources
func NewConnectController(provider, connector string,
	context context.Context,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	configClient configClientset.Interface,
	machineClient machineClientset.Interface,
	gatewayClient gwapi.Interface,
	informerCollection *fsminformers.InformerCollection,
	msgBroker *messaging.Broker,
	selectInformers ...InformerKey) connector.ConnectController {
	return newClient(provider, connector,
		context,
		kubeConfig,
		kubeClient,
		configClient,
		machineClient,
		gatewayClient,
		informerCollection,
		msgBroker,
		selectInformers...)
}

func newClient(provider, connectorName string,
	context context.Context,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	configClient configClientset.Interface,
	machineClient machineClientset.Interface,
	gatewayClient gwapi.Interface,
	informerCollection *fsminformers.InformerCollection,
	msgBroker *messaging.Broker,
	selectInformers ...InformerKey) *client {
	// Initialize client object
	c := &client{
		connectorProvider: provider,
		connectorName:     connectorName,

		context:       context,
		kubeConfig:    kubeConfig,
		kubeClient:    kubeClient,
		configClient:  configClient,
		machineClient: machineClient,
		gatewayClient: gatewayClient,

		c2kContext: connector.NewC2KContext(),
		k2cContext: connector.NewK2CContext(),
		k2gContext: connector.NewK2GContext(),

		informers:         informerCollection,
		msgBroker:         msgBroker,
		msgWorkerPoolSize: 0,
		msgWorkQueues:     workerpool.NewWorkerPool(0),

		limiter: rate.NewLimiter(0, 0),
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
func (c *client) GetConsulConnector(connector string) *ctv1.ConsulConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyConsulConnector, connector)
	if exists && err == nil {
		return connectorIf.(*ctv1.ConsulConnector)
	}
	return nil
}

// GetEurekaConnector returns a EurekaConnector resource if found, nil otherwise.
func (c *client) GetEurekaConnector(connector string) *ctv1.EurekaConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyEurekaConnector, connector)
	if exists && err == nil {
		return connectorIf.(*ctv1.EurekaConnector)
	}
	return nil
}

// GetNacosConnector returns a NacosConnector resource if found, nil otherwise.
func (c *client) GetNacosConnector(connector string) *ctv1.NacosConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyNacosConnector, connector)
	if exists && err == nil {
		return connectorIf.(*ctv1.NacosConnector)
	}
	return nil
}

// GetMachineConnector returns a MachineConnector resource if found, nil otherwise.
func (c *client) GetMachineConnector(connector string) *ctv1.MachineConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyMachineConnector, connector)
	if exists && err == nil {
		return connectorIf.(*ctv1.MachineConnector)
	}
	return nil
}

// GetGatewayConnector returns a GatewayConnector resource if found, nil otherwise.
func (c *client) GetGatewayConnector(connector string) *ctv1.GatewayConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyGatewayConnector, connector)
	if exists && err == nil {
		return connectorIf.(*ctv1.GatewayConnector)
	}
	return nil
}

// GetConnector returns a Connector resource if found, nil otherwise.
func (c *client) GetConnector() (spec interface{}, uid string, ok bool) {
	switch c.GetConnectorProvider() {
	case ctv1.ConsulDiscoveryService:
		if connector := c.GetConsulConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case ctv1.EurekaDiscoveryService:
		if connector := c.GetEurekaConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case ctv1.NacosDiscoveryService:
		if connector := c.GetNacosConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case ctv1.MachineDiscoveryService:
		if connector := c.GetMachineConnector(c.GetConnectorName()); connector != nil {
			spec = connector.Spec
			uid = string(connector.UID)
			ok = true
		}
	case ctv1.GatewayDiscoveryService:
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
func (c *client) GetConnectorProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.DiscoveryServiceProvider(c.connectorProvider)
}

// GetConnectorName returns connector name.
func (c *client) GetConnectorName() string {
	return c.connectorName
}

// GetConnectorUID returns connector uid.
func (c *client) GetConnectorUID() string {
	return c.connectorUID
}

// GetClusterSet returns cluster set.
func (c *client) GetClusterSet() string {
	return c.clusterSet
}

// SetClusterSet sets cluster set.
func (c *client) SetClusterSet(name, group, zone, region string) {
	c.clusterSet = fmt.Sprintf("%s.%s.%s.%s", name, group, zone, region)
}

func (c *client) SetServiceInstanceIDFunc(f connector.ServiceInstanceIDFunc) {
	c.serviceInstanceIDFunc = f
}

// GetServiceInstanceID generates a unique ID for a service. This ID is not meant
// to be particularly human-friendly.
func (c *client) GetServiceInstanceID(name, addr string, httpPort, grpcPort int) string {
	if c.serviceInstanceIDFunc != nil {
		return c.serviceInstanceIDFunc(name, addr, httpPort, grpcPort)
	}
	if grpcPort > 0 {
		return strings.ToLower(fmt.Sprintf("%s-%s-%d-%d-%s", name, addr, httpPort, grpcPort, c.clusterSet))
	}
	return strings.ToLower(fmt.Sprintf("%s-%s-%d-%s", name, addr, httpPort, c.clusterSet))
}
