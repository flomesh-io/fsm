package connector

import (
	"time"

	mapset "github.com/deckarep/golang-set"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
)

// ConnectController is the controller interface for K8s connectors
type ConnectController interface {
	BroadcastListener()

	GetConnectorProvider() ctv1.DiscoveryServiceProvider
	GetConnectorName() string
	GetConnectorUID() string

	GetConsulConnector(connector string) *ctv1.ConsulConnector
	GetEurekaConnector(connector string) *ctv1.EurekaConnector
	GetNacosConnector(connector string) *ctv1.NacosConnector
	GetMachineConnector(connector string) *ctv1.MachineConnector
	GetGatewayConnector(connector string) *ctv1.GatewayConnector
	GetConnector() (spec interface{}, uid string, ok bool)

	Refresh()

	WaitLimiter()

	GetC2KContext() *C2KContext
	GetK2CContext() *K2CContext
	GetK2GContext() *K2GContext

	GetClusterSet() string
	SetClusterSet(name, group, zone, region string)

	SetServiceInstanceIDFunc(f ServiceInstanceIDFunc)
	GetServiceInstanceID(name, addr string, httpPort, grpcPort int) string

	/* config for ctok source */

	GetClusterId() string
	GetPassingOnly() bool
	GetFilterTag() string
	GetFilterMetadatas() []ctv1.Metadata
	GetPrefix() string
	GetPrefixTag() string
	GetSuffixTag() string
	GetPrefixMetadata() string
	GetSuffixMetadata() string

	GetC2KWithGateway() bool

	GetNacos2KClusterSet() []string
	GetNacos2KGroupSet() []string

	/* config for ktoc source */

	GetSyncPeriod() time.Duration
	GetDefaultSync() bool
	GetSyncClusterIPServices() bool
	GetSyncLoadBalancerEndpoints() bool
	GetNodePortSyncType() ctv1.NodePortSyncType

	GetSyncIngress() bool
	GetSyncIngressLoadBalancerIPs() bool

	GetAddServicePrefix() string
	GetAddK8SNamespaceAsServiceSuffix() bool

	GetAppendTagSet() mapset.Set
	GetAppendMetadataSet() mapset.Set
	GetAllowK8SNamespaceSet() mapset.Set
	GetDenyK8SNamespaceSet() mapset.Set

	GetK2CWithGateway() bool

	GetConsulNodeName() string
	GetConsulEnableNamespaces() bool
	GetConsulDestinationNamespace() string
	GetConsulEnableK8SNSMirroring() bool
	GetConsulK8SNSMirroringPrefix() string
	GetConsulCrossNamespaceACLPolicy() string

	GetNacosGroupId() string
	GetNacosClusterId() string

	/* config for ktog source */

	GetK2GSyncPeriod() time.Duration
	GetK2GDefaultSync() bool
	GetK2GAllowK8SNamespaceSet() mapset.Set
	GetK2GDenyK8SNamespaceSet() mapset.Set

	/* config for via gateway */

	GetViaIngressIPSelector() ctv1.AddrSelector
	GetViaEgressIPSelector() ctv1.AddrSelector

	GetViaIngressAddr() string
	SetViaIngressAddr(ingressAddr string)

	GetViaEgressAddr() string
	SetViaEgressAddr(egressAddr string)

	GetViaIngressHTTPPort() uint
	SetViaIngressHTTPPort(httpPort uint)

	GetViaIngressGRPCPort() uint
	SetViaIngressGRPCPort(grpcPort uint)

	GetViaEgressHTTPPort() uint
	SetViaEgressHTTPPort(httpPort uint)

	GetViaEgressGRPCPort() uint
	SetViaEgressGRPCPort(grpcPort uint)

	GetAuthConsulUsername() string
	GetAuthConsulPassword() string

	GetAuthNacosUsername() string
	GetAuthNacosPassword() string
	GetAuthNacosAccessKey() string
	GetAuthNacosSecretKey() string
	GetAuthNacosNamespaceId() string

	SyncCloudToK8s() bool
	SyncK8sToCloud() bool
	SyncK8sToGateway() bool

	GetHTTPAddr() string
	GetDeriveNamespace() string
	AsInternalServices() bool
}

type ServiceInstanceIDFunc func(name, addr string, httpPort, grpcPort int) string
