package connector

import (
	"time"

	mapset "github.com/deckarep/golang-set"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/utils/cidr"
)

// ConnectController is the controller interface for K8s connectors
type ConnectController interface {
	BroadcastListener(stopCh <-chan struct{})

	GetConnectorProvider() ctv1.DiscoveryServiceProvider
	GetConnectorNamespace() string
	GetConnectorName() string
	GetConnectorUID() string

	GetConsulConnector(namespace, name string) *ctv1.ConsulConnector
	GetEurekaConnector(namespace, name string) *ctv1.EurekaConnector
	GetNacosConnector(namespace, name string) *ctv1.NacosConnector
	GetZookeeperConnector(namespace, name string) *ctv1.ZookeeperConnector
	GetMachineConnector(namespace, name string) *ctv1.MachineConnector
	GetGatewayConnector(namespace, name string) *ctv1.GatewayConnector
	GetConnector() (connector, spec interface{}, uid string, ok bool)

	Refresh()

	WaitLimiter()

	GetC2KContext() *C2KContext
	GetK2CContext() *K2CContext
	GetK2GContext() *K2GContext

	GetClusterSet() string
	SetClusterSet(name, group, zone, region string)

	SetServiceInstanceIDFunc(f ServiceInstanceIDFunc)
	GetServiceInstanceID(name, addr string, port MicroServicePort, protocol MicroServiceProtocol) string

	/* config for ctok source */

	GetClusterId() string
	GetPassingOnly() bool
	GetC2KFilterTag() string
	GetC2KFilterMetadatas() []ctv1.Metadata
	GetC2KExcludeMetadatas() []ctv1.Metadata
	GetC2KFilterIPRanges() []*cidr.CIDR
	GetC2KExcludeIPRanges() []*cidr.CIDR
	GetK2CFilterIPRanges() []*cidr.CIDR
	GetK2CExcludeIPRanges() []*cidr.CIDR
	GetPrefixTag() string
	GetSuffixTag() string
	GetPrefixMetadata() string
	GetSuffixMetadata() string

	GetC2KFixedHTTPServicePort() *uint32

	EnableC2KMetadataStrategy() bool
	GetC2KMetadataToLabelConversions() map[string]string
	GetC2KMetadataToAnnotationConversions() map[string]string

	EnableC2KConversions() bool
	GetC2KServiceConversions() map[string]ctv1.ServiceConversion

	GetC2KWithGateway() bool
	GetC2KMultiGateways() bool

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
	GetK2CWithGatewayMode() ctv1.WithGatewayMode

	GetConsulNodeName() string
	GetConsulEnableNamespaces() bool
	GetConsulDestinationNamespace() string
	GetConsulEnableK8SNSMirroring() bool
	GetConsulK8SNSMirroringPrefix() string
	GetConsulCrossNamespaceACLPolicy() string
	GetConsulGenerateInternalServiceHealthCheck() bool

	GetEurekaHeartBeatInstance() bool
	GetEurekaHeartBeatPeriod() time.Duration
	GetEurekaCheckServiceInstanceID() bool

	GetNacosGroupId() string
	GetNacosClusterId() string

	GetZookeeperBasePath() string
	GetZookeeperCategory() string
	GetZookeeperAdaptor() string

	/* config for ktog source */

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
	Purge() bool
	AsInternalServices() bool

	CacheCatalogInstances(key string, catalogFunc func() (interface{}, error)) (interface{}, error)
	CacheRegisterInstance(key string, instance interface{}, registerFunc func() error) error
	CacheDeregisterInstance(key string, deregisterFunc func() error) error
	CacheCleaner(stopCh <-chan struct{})
}

type ServiceInstanceIDFunc func(name, addr string, port MicroServicePort, protocol MicroServiceProtocol) string
