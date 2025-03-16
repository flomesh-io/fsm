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
	GetConnectorName() string
	GetConnectorUID() string

	GetConsulConnector(connector string) *ctv1.ConsulConnector
	GetEurekaConnector(connector string) *ctv1.EurekaConnector
	GetNacosConnector(connector string) *ctv1.NacosConnector
	GetMachineConnector(connector string) *ctv1.MachineConnector
	GetGatewayConnector(connector string) *ctv1.GatewayConnector
	GetConnector() (connector, spec interface{}, uid string, ok bool)

	Refresh()

	WaitLimiter()

	GetSyncPeriod() time.Duration

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
	GetC2KFilterTag() string
	GetC2KFilterMetadatas() []ctv1.Metadata
	GetC2KExcludeMetadatas() []ctv1.Metadata
	GetC2KFilterIPRanges() []*cidr.CIDR
	GetC2KExcludeIPRanges() []*cidr.CIDR
	GetK2CFilterIPRanges() []*cidr.CIDR
	GetK2CExcludeIPRanges() []*cidr.CIDR
	GetPrefix() string
	GetPrefixTag() string
	GetSuffixTag() string
	GetPrefixMetadata() string
	GetSuffixMetadata() string

	GetC2KFixedHTTPServicePort() *uint32
	GetC2KFixedGRPCServicePort() *uint32

	GetC2KAppendLabels() map[string]string
	GetC2KAppendAnnotations() map[string]string

	EnableC2KTagStrategy() bool
	GetC2KTagToLabelConversions() map[string]string
	GetC2KTagToAnnotationConversions() map[string]string

	EnableC2KMetadataStrategy() bool
	GetC2KMetadataToLabelConversions() map[string]string
	GetC2KMetadataToAnnotationConversions() map[string]string

	GetC2KWithGateway() bool
	GetC2KMultiGateways() bool

	GetNacos2KClusterSet() []string
	GetNacos2KGroupSet() []string

	/* config for ktoc source */

	GetK2CDefaultSync() bool
	GetK2CSyncClusterIPServices() bool
	GetK2CSyncLoadBalancerEndpoints() bool
	GetK2CNodePortSyncType() ctv1.NodePortSyncType

	GetK2CSyncIngress() bool
	GetK2CSyncIngressLoadBalancerIPs() bool

	GetK2CAddServicePrefix() string
	GetK2CAddK8SNamespaceAsServiceSuffix() bool

	GetK2CAppendTagSet() mapset.Set
	GetK2CAppendMetadataSet() mapset.Set

	EnableK2CTagStrategy() bool
	GetK2CTagToLabelConversions() map[string]string
	GetK2CTagToAnnotationConversions() map[string]string

	EnableK2CMetadataStrategy() bool
	GetK2CMetadataToLabelConversions() map[string]string
	GetK2CMetadataToAnnotationConversions() map[string]string

	GetK2CAllowK8SNamespaceSet() mapset.Set
	GetK2CDenyK8SNamespaceSet() mapset.Set

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

type ServiceInstanceIDFunc func(name, addr string, httpPort, grpcPort int) string
