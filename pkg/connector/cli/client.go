package cli

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/time/rate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/announcements"
	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/utils/chm"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

// NewConnectController returns a new Connector.Controller which means to provide access to locally-cached connector resources
func NewConnectController(provider, connectorNamespace, connectorName string,
	context context.Context,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	configClient configClientset.Interface,
	connectorClient connectorClientset.Interface,
	machineClient machineClientset.Interface,
	gatewayClient gwapi.Interface,
	informerCollection *fsminformers.InformerCollection,
	msgBroker *messaging.Broker,
	selectInformers ...InformerKey) connector.ConnectController {
	return newClient(provider, connectorNamespace, connectorName,
		context,
		kubeConfig,
		kubeClient,
		configClient,
		connectorClient,
		machineClient,
		gatewayClient,
		informerCollection,
		msgBroker,
		selectInformers...)
}

func newClient(provider, connectorNamespace, connectorName string,
	context context.Context,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	configClient configClientset.Interface,
	connectorClient connectorClientset.Interface,
	machineClient machineClientset.Interface,
	gatewayClient gwapi.Interface,
	informerCollection *fsminformers.InformerCollection,
	msgBroker *messaging.Broker,
	selectInformers ...InformerKey) *client {
	// Initialize client object
	c := &client{
		connectorProvider:  provider,
		connectorNamespace: connectorNamespace,
		connectorName:      connectorName,

		context:         context,
		kubeConfig:      kubeConfig,
		kubeClient:      kubeClient,
		configClient:    configClient,
		connectorClient: connectorClient,
		machineClient:   machineClient,
		gatewayClient:   gatewayClient,

		c2kContext: connector.NewC2KContext(),
		k2cContext: connector.NewK2CContext(),
		k2gContext: connector.NewK2GContext(),

		informers:         informerCollection,
		msgBroker:         msgBroker,
		msgWorkerPoolSize: 0,
		msgWorkQueues:     workerpool.NewWorkerPool(0),

		limiter: rate.NewLimiter(0, 0),

		cache: cache{
			catalogInstances:    chm.NewConcurrentMap[*catalogTimeScale](),
			registeredInstances: chm.NewConcurrentMap[*registerTimeScale](),
		},
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]func(){
		ConsulConnectors:    c.initConsulConnectorMonitor,
		EurekaConnectors:    c.initEurekaConnectorMonitor,
		NacosConnectors:     c.initNacosConnectorMonitor,
		ZookeeperConnectors: c.initZookeeperConnectorMonitor,
		MachineConnectors:   c.initMachineConnectorMonitor,
		GatewayConnectors:   c.initGatewayConnectorMonitor,
		GatewayHTTPRoutes:   c.initGatewayHTTPRouteMonitor,
		GatewayGRPCRoutes:   c.initGatewayGRPCRouteMonitor,
		GatewayTCPRoutes:    c.initGatewayTCPRouteMonitor,
	}

	// If specific informers are not selected to be initialized, initialize all informers
	if len(selectInformers) == 0 {
		selectInformers = []InformerKey{
			ConsulConnectors,
			EurekaConnectors,
			NacosConnectors,
			ZookeeperConnectors,
			MachineConnectors,
			GatewayConnectors,
			GatewayHTTPRoutes,
			GatewayGRPCRoutes,
			GatewayTCPRoutes}
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

func (c *client) initZookeeperConnectorMonitor() {
	zookeeperConnectorEventTypes := k8s.EventTypes{
		Add:    announcements.ZookeeperConnectorAdded,
		Update: announcements.ZookeeperConnectorUpdated,
		Delete: announcements.ZookeeperConnectorDeleted,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyZookeeperConnector,
		k8s.GetEventHandlerFuncs(nil, zookeeperConnectorEventTypes, c.msgBroker))
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

func (c *client) initGatewayHTTPRouteMonitor() {
	gatewayHTTPRouteEventTypes := k8s.EventTypes{
		Add:    announcements.GatewayAPIHTTPRouteAdded,
		Update: announcements.GatewayAPIHTTPRouteDeleted,
		Delete: announcements.GatewayAPIHTTPRouteUpdated,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyGatewayAPIHTTPRoute,
		k8s.GetEventHandlerFuncs(nil, gatewayHTTPRouteEventTypes, c.msgBroker))
}

func (c *client) initGatewayGRPCRouteMonitor() {
	gatewayGRPCRouteEventTypes := k8s.EventTypes{
		Add:    announcements.GatewayAPIGRPCRouteAdded,
		Update: announcements.GatewayAPIGRPCRouteDeleted,
		Delete: announcements.GatewayAPIGRPCRouteUpdated,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyGatewayAPIGRPCRoute,
		k8s.GetEventHandlerFuncs(nil, gatewayGRPCRouteEventTypes, c.msgBroker))
}

func (c *client) initGatewayTCPRouteMonitor() {
	gatewayTCPRouteEventTypes := k8s.EventTypes{
		Add:    announcements.GatewayAPITCPRouteAdded,
		Update: announcements.GatewayAPITCPRouteDeleted,
		Delete: announcements.GatewayAPITCPRouteUpdated,
	}
	c.informers.AddEventHandler(fsminformers.InformerKeyGatewayAPITCPRoute,
		k8s.GetEventHandlerFuncs(nil, gatewayTCPRouteEventTypes, c.msgBroker))
}

// GetConsulConnector returns a ConsulConnector resource if found, nil otherwise.
func (c *client) GetConsulConnector(namespace, name string) *ctv1.ConsulConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyConsulConnector, fmt.Sprintf("%s/%s", namespace, name))
	if exists && err == nil {
		return connectorIf.(*ctv1.ConsulConnector)
	}
	return nil
}

// GetEurekaConnector returns a EurekaConnector resource if found, nil otherwise.
func (c *client) GetEurekaConnector(namespace, name string) *ctv1.EurekaConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyEurekaConnector, fmt.Sprintf("%s/%s", namespace, name))
	if exists && err == nil {
		return connectorIf.(*ctv1.EurekaConnector)
	}
	return nil
}

// GetNacosConnector returns a NacosConnector resource if found, nil otherwise.
func (c *client) GetNacosConnector(namespace, name string) *ctv1.NacosConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyNacosConnector, fmt.Sprintf("%s/%s", namespace, name))
	if exists && err == nil {
		return connectorIf.(*ctv1.NacosConnector)
	}
	return nil
}

// GetZookeeperConnector returns a ZookeeperConnector resource if found, nil otherwise.
func (c *client) GetZookeeperConnector(namespace, name string) *ctv1.ZookeeperConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyZookeeperConnector, fmt.Sprintf("%s/%s", namespace, name))
	if exists && err == nil {
		return connectorIf.(*ctv1.ZookeeperConnector)
	}
	return nil
}

// GetMachineConnector returns a MachineConnector resource if found, nil otherwise.
func (c *client) GetMachineConnector(namespace, name string) *ctv1.MachineConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyMachineConnector, fmt.Sprintf("%s/%s", namespace, name))
	if exists && err == nil {
		return connectorIf.(*ctv1.MachineConnector)
	}
	return nil
}

// GetGatewayConnector returns a GatewayConnector resource if found, nil otherwise.
func (c *client) GetGatewayConnector(namespace, name string) *ctv1.GatewayConnector {
	connectorIf, exists, err := c.informers.GetByKey(fsminformers.InformerKeyGatewayConnector, fmt.Sprintf("%s/%s", namespace, name))
	if exists && err == nil {
		return connectorIf.(*ctv1.GatewayConnector)
	}
	return nil
}

// GetConnector returns a Connector resource if found, nil otherwise.
func (c *client) GetConnector() (connector, spec interface{}, uid string, ok bool) {
	switch c.GetConnectorProvider() {
	case ctv1.ConsulDiscoveryService:
		if consulConnector := c.GetConsulConnector(c.GetConnectorNamespace(), c.GetConnectorName()); consulConnector != nil {
			connector = consulConnector
			spec = consulConnector.Spec
			uid = string(consulConnector.UID)
			ok = true
		}
	case ctv1.EurekaDiscoveryService:
		if eurekaConnector := c.GetEurekaConnector(c.GetConnectorNamespace(), c.GetConnectorName()); eurekaConnector != nil {
			connector = eurekaConnector
			spec = eurekaConnector.Spec
			uid = string(eurekaConnector.UID)
			ok = true
		}
	case ctv1.NacosDiscoveryService:
		if nacosConnector := c.GetNacosConnector(c.GetConnectorNamespace(), c.GetConnectorName()); nacosConnector != nil {
			connector = nacosConnector
			spec = nacosConnector.Spec
			uid = string(nacosConnector.UID)
			ok = true
		}
	case ctv1.ZookeeperDiscoveryService:
		if zookeeperConnector := c.GetZookeeperConnector(c.GetConnectorNamespace(), c.GetConnectorName()); zookeeperConnector != nil {
			connector = zookeeperConnector
			spec = zookeeperConnector.Spec
			uid = string(zookeeperConnector.UID)
			ok = true
		}
	case ctv1.MachineDiscoveryService:
		if machineConnector := c.GetMachineConnector(c.GetConnectorNamespace(), c.GetConnectorName()); machineConnector != nil {
			connector = machineConnector
			spec = machineConnector.Spec
			uid = string(machineConnector.UID)
			ok = true
		}
	case ctv1.GatewayDiscoveryService:
		if gatewayConnector := c.GetGatewayConnector(c.GetConnectorNamespace(), c.GetConnectorName()); gatewayConnector != nil {
			connector = gatewayConnector
			spec = gatewayConnector.Spec
			uid = string(gatewayConnector.UID)
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

// GetConnectorNamespace returns connector namespace.
func (c *client) GetConnectorNamespace() string {
	return c.connectorNamespace
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
func (c *client) GetServiceInstanceID(name, addr string, port connector.MicroServicePort, protocol connector.MicroServiceProtocol) string {
	if c.serviceInstanceIDFunc != nil {
		return c.serviceInstanceIDFunc(name, addr, port, protocol)
	}
	return strings.ToLower(fmt.Sprintf("%s-%s-%d-%s", name, addr, port, c.clusterSet))
}

func (c *client) updateConnectorStatus() {
	if connector, _, _, exists := c.GetConnector(); exists {
		if consulConnector, ok := connector.(*ctv1.ConsulConnector); ok {
			if update := c.checkConnectorStatus(&consulConnector.Status); update {
				if _, err := c.connectorClient.ConnectorV1alpha1().ConsulConnectors(consulConnector.Namespace).
					UpdateStatus(c.context, consulConnector, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err).Msgf("fail to update status for connector: %s/%s", consulConnector.Namespace, consulConnector.Name)
				}
			}
			return
		}
		if eurekaConnector, ok := connector.(*ctv1.EurekaConnector); ok {
			if update := c.checkConnectorStatus(&eurekaConnector.Status); update {
				if _, err := c.connectorClient.ConnectorV1alpha1().EurekaConnectors(eurekaConnector.Namespace).
					UpdateStatus(c.context, eurekaConnector, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err).Msgf("fail to update status for connector: %s/%s", eurekaConnector.Namespace, eurekaConnector.Name)
				}
			}
			return
		}
		if nacosConnector, ok := connector.(*ctv1.NacosConnector); ok {
			if update := c.checkConnectorStatus(&nacosConnector.Status); update {
				if _, err := c.connectorClient.ConnectorV1alpha1().NacosConnectors(nacosConnector.Namespace).
					UpdateStatus(c.context, nacosConnector, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err).Msgf("fail to update status for connector: %s/%s", nacosConnector.Namespace, nacosConnector.Name)
				}
			}
			return
		}
		if zookeeperConnector, ok := connector.(*ctv1.ZookeeperConnector); ok {
			if update := c.checkConnectorStatus(&zookeeperConnector.Status); update {
				if _, err := c.connectorClient.ConnectorV1alpha1().ZookeeperConnectors(zookeeperConnector.Namespace).
					UpdateStatus(c.context, zookeeperConnector, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err).Msgf("fail to update status for connector: %s/%s", zookeeperConnector.Namespace, zookeeperConnector.Name)
				}
			}
			return
		}
		if machineConnector, ok := connector.(*ctv1.MachineConnector); ok {
			if update := c.checkConnectorStatus(&machineConnector.Status); update {
				if _, err := c.connectorClient.ConnectorV1alpha1().MachineConnectors(machineConnector.Namespace).
					UpdateStatus(c.context, machineConnector, metav1.UpdateOptions{}); err != nil {
					log.Error().Err(err).Msgf("fail to update status for connector: %s/%s", machineConnector.Namespace, machineConnector.Name)
				}
			}
			return
		}
	}
}

func (c *client) checkConnectorStatus(connectorStatus *ctv1.ConnectorStatus) bool {
	toK8sServiceCnt := len(c.c2kContext.KubeServiceCache)
	fromK8sServiceCnt := c.k2cContext.ServiceMap.Count() + c.k2cContext.IngressServiceMap.Count()
	update := false

	if toK8sServiceCnt != connectorStatus.ToK8SServiceCnt {
		connectorStatus.ToK8SServiceCnt = toK8sServiceCnt
		update = true
	}
	if fromK8sServiceCnt != connectorStatus.FromK8SServiceCnt {
		connectorStatus.FromK8SServiceCnt = fromK8sServiceCnt
		update = true
	}
	if hexHash := fmt.Sprintf("%x", c.c2kContext.CatalogServicesHash); connectorStatus.CatalogServicesHash != hexHash {
		connectorStatus.CatalogServicesHash = hexHash
		connectorStatus.CatalogServices = c.c2kContext.CatalogServices
		update = true
	}
	return update
}
