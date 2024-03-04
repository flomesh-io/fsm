package connector

import connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"

// ConnectorController is the controller interface for K8s connectors
type ConnectorController interface {
	BroadcastListener()
	GetConnectorProvider() connectorv1alpha1.DiscoveryServiceProvider
	GetConnectorName() string
	GetConsulConnector(connector string) *connectorv1alpha1.ConsulConnector
	GetEurekaConnector(connector string) *connectorv1alpha1.EurekaConnector
	GetNacosConnector(connector string) *connectorv1alpha1.NacosConnector
	GetMachineConnector(connector string) *connectorv1alpha1.MachineConnector
	GetGatewayConnector(connector string) *connectorv1alpha1.GatewayConnector
	GetConnector() (spec interface{}, uid string, ok bool)
	Refresh()
}
