package cli

import (
	"fmt"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"golang.org/x/time/rate"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/utils/cidr"
)

const (
	MinSyncPeriod = 2 * time.Second
)

type protocolPort struct {
	httpPort uint
	grpcPort uint
}

type config struct {
	flock sync.RWMutex

	auth struct {
		consul struct {
			username string
			password string
		}
		nacos struct {
			username    string
			password    string
			accessKey   string
			secretKey   string
			namespaceId string
		}
		zookeeper struct {
			password string
		}
	}

	httpAddr           string
	deriveNamespace    string
	purge              bool
	asInternalServices bool

	// syncPeriod is the interval between full catalog syncs. These will
	// re-register all services to prevent overwrites of data. This should
	// happen relatively infrequently and default to 5 seconds.
	syncPeriod time.Duration

	c2kCfg struct {
		enable           bool
		clusterId        string
		passingOnly      bool
		filterTag        string
		filterMetadatas  []ctv1.Metadata
		filterIPRanges   []string
		excludeMetadatas []ctv1.Metadata
		excludeIPRanges  []string

		prefixTag      string
		suffixTag      string
		prefixMetadata string
		suffixMetadata string

		fixedHTTPServicePort *uint32

		metadataStrategy *ctv1.MetadataStrategy

		enableConversions  bool
		serviceConversions map[string]ctv1.ServiceConversion

		withGateway   bool
		multiGateways bool

		nacos2kCfg struct {
			clusterSet []string
			groupSet   []string
		}
	}

	k2cCfg struct {
		enable bool

		// defaultSync should set to be false to require explicit enabling
		// using annotations. If this is true, then services are implicitly
		// enabled (aka default enabled).
		defaultSync bool

		// syncClusterIPServices set to true (the default) syncs ClusterIP-type services.
		// Setting this to false will ignore ClusterIP services during the sync.
		syncClusterIPServices bool

		// syncLoadBalancerEndpoints set to true (default false) will sync ServiceTypeLoadBalancer endpoints.
		syncLoadBalancerEndpoints bool

		// NodeExternalIPSync set to true (the default) syncs NodePort services
		// using the node's external ip address. When false, the node's internal
		// ip address will be used instead.
		nodePortSyncType ctv1.NodePortSyncType

		// syncIngress enables syncing of the hostname from an Ingress resource
		// to the service registration if an Ingress rule matches the service.
		syncIngress bool

		// syncIngressLoadBalancerIPs enables syncing the IP of the Ingress LoadBalancer
		// if we do not want to sync the hostname from the Ingress resource.
		syncIngressLoadBalancerIPs bool

		//addServicePrefix prepends K8s services in cloud with a prefix
		addServicePrefix string

		// addK8SNamespaceAsServiceSuffix set to true appends Kubernetes namespace
		// to the service name being synced to Cloud separated by a dash.
		// For example, service 'foo' in the 'default' namespace will be synced
		// as 'foo-default'.
		addK8SNamespaceAsServiceSuffix bool

		appendTagSet      mapset.Set
		appendMetadataSet mapset.Set

		// allowK8sNamespacesSet is a set of k8s namespaces to explicitly allow for
		// syncing. It supports the special character `*` which indicates that
		// all k8s namespaces are eligible unless explicitly denied. This filter
		// is applied before checking pod annotations.
		allowK8sNamespacesSet mapset.Set

		// denyK8sNamespacesSet is a set of k8s namespaces to explicitly deny
		// syncing and thus service registration with Consul. An empty set
		// means that no namespaces are removed from consideration. This filter
		// takes precedence over allowK8sNamespacesSet.
		denyK8sNamespacesSet mapset.Set

		filterIPRanges  []string
		excludeIPRanges []string

		withGateway bool

		withGatewayMode ctv1.WithGatewayMode

		consulCfg struct {
			// The Consul node name to register services with.
			consulNodeName string

			// enableNamespaces indicates that a user is running Consul Enterprise
			// with version 1.7+ which is namespace aware. It enables Consul namespaces,
			// with syncing into either a single Consul namespace or mirrored from
			// k8s namespaces.
			consulEnableNamespaces bool

			// destinationNamespace is the name of the Consul namespace to register all
			// synced services into if Consul namespaces are enabled and mirroring
			// is disabled. This will not be used if mirroring is enabled.
			consulDestinationNamespace string

			// enableK8SNSMirroring causes Consul namespaces to be created to match the
			// organization within k8s. Services are registered into the Consul
			// namespace that mirrors their k8s namespace.
			consulEnableK8SNSMirroring bool

			// k8sNSMirroringPrefix is an optional prefix that can be added to the Consul
			// namespaces created while mirroring. For example, if it is set to "k8s-",
			// then the k8s `default` namespace will be mirrored in Consul's
			// `k8s-default` namespace.
			consulK8SNSMirroringPrefix string

			// crossNamespaceACLPolicy is the name of the ACL policy to attach to
			// any created Consul namespaces to allow cross namespace service discovery.
			// Only necessary if ACLs are enabled.
			consulCrossNamespaceACLPolicy string

			consulGenerateInternalServiceHealthCheck bool
		}

		eurekaCfg struct {
			heartBeatInstance      bool
			heartBeatPeriod        time.Duration
			checkServiceInstanceID bool
		}

		nacosCfg struct {
			clusterId string
			groupId   string
		}

		zookeeperCfg struct {
			basePath string
			category string
			adaptor  string
		}
	}

	k2gCfg struct {
		enable bool

		// defaultSync should set to be false to require explicit enabling
		// using annotations. If this is true, then services are implicitly
		// enabled (aka default enabled).
		defaultSync bool

		// allowK8sNamespacesSet is a set of k8s namespaces to explicitly allow for
		// syncing. It supports the special character `*` which indicates that
		// all k8s namespaces are eligible unless explicitly denied. This filter
		// is applied before checking pod annotations.
		allowK8sNamespacesSet mapset.Set

		// denyK8sNamespacesSet is a set of k8s namespaces to explicitly deny
		// syncing and thus service registration with Consul. An empty set
		// means that no namespaces are removed from consideration. This filter
		// takes precedence over allowK8sNamespacesSet.
		denyK8sNamespacesSet mapset.Set
	}

	viaCfg struct {
		fgwName           string
		ingressIPSelector ctv1.AddrSelector
		egressIPSelector  ctv1.AddrSelector

		ingressAddr string
		egressAddr  string

		ingress protocolPort
		egress  protocolPort
	}
}

func (c *config) GetViaFgwName() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.fgwName
}

func (c *config) GetViaIngressIPSelector() ctv1.AddrSelector {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.ingressIPSelector
}

func (c *config) GetViaEgressIPSelector() ctv1.AddrSelector {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.egressIPSelector
}

func (c *config) GetViaIngressAddr() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.ingressAddr
}

func (c *config) SetViaIngressAddr(ingressAddr string) {
	c.flock.Lock()
	defer c.flock.Unlock()
	c.viaCfg.ingressAddr = ingressAddr
}

func (c *config) GetViaEgressAddr() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.egressAddr
}

func (c *config) SetViaEgressAddr(egressAddr string) {
	c.flock.Lock()
	defer c.flock.Unlock()
	c.viaCfg.egressAddr = egressAddr
}

func (c *config) GetViaIngressHTTPPort() uint {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.ingress.httpPort
}

func (c *config) SetViaIngressHTTPPort(httpPort uint) {
	c.flock.Lock()
	defer c.flock.Unlock()
	c.viaCfg.ingress.httpPort = httpPort
}

func (c *config) GetViaIngressGRPCPort() uint {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.ingress.grpcPort
}

func (c *config) SetViaIngressGRPCPort(grpcPort uint) {
	c.flock.Lock()
	defer c.flock.Unlock()
	c.viaCfg.ingress.grpcPort = grpcPort
}

func (c *config) GetViaEgressHTTPPort() uint {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.egress.httpPort
}

func (c *config) SetViaEgressHTTPPort(httpPort uint) {
	c.flock.Lock()
	defer c.flock.Unlock()
	c.viaCfg.egress.httpPort = httpPort
}

func (c *config) GetViaEgressGRPCPort() uint {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.viaCfg.egress.grpcPort
}

func (c *config) SetViaEgressGRPCPort(grpcPort uint) {
	c.flock.Lock()
	defer c.flock.Unlock()
	c.viaCfg.egress.grpcPort = grpcPort
}

func (c *config) GetK2GDefaultSync() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2gCfg.defaultSync
}

func (c *config) GetK2GAllowK8SNamespaceSet() mapset.Set {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2gCfg.allowK8sNamespacesSet
}

func (c *config) GetK2GDenyK8SNamespaceSet() mapset.Set {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2gCfg.denyK8sNamespacesSet
}

func (c *config) GetDefaultSync() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.defaultSync
}

func (c *config) GetSyncClusterIPServices() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.syncClusterIPServices
}

func (c *config) GetSyncLoadBalancerEndpoints() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.syncLoadBalancerEndpoints
}

func (c *config) GetNodePortSyncType() ctv1.NodePortSyncType {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.nodePortSyncType
}

func (c *config) GetSyncIngress() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.syncIngress
}

func (c *config) GetSyncIngressLoadBalancerIPs() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.syncIngressLoadBalancerIPs
}

func (c *config) GetAddServicePrefix() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.addServicePrefix
}

func (c *config) GetAddK8SNamespaceAsServiceSuffix() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.addK8SNamespaceAsServiceSuffix
}

func (c *config) GetAppendTagSet() mapset.Set {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.appendTagSet
}

func (c *config) GetAppendMetadataSet() mapset.Set {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.appendMetadataSet
}

func (c *config) GetAllowK8SNamespaceSet() mapset.Set {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.allowK8sNamespacesSet
}

func (c *config) GetDenyK8SNamespaceSet() mapset.Set {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.denyK8sNamespacesSet
}

func (c *config) GetK2CWithGateway() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.withGateway
}

func (c *config) GetK2CWithGatewayMode() ctv1.WithGatewayMode {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.withGatewayMode
}

func (c *config) GetConsulNodeName() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulNodeName
}

func (c *config) GetConsulEnableNamespaces() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulEnableNamespaces
}

func (c *config) GetConsulDestinationNamespace() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulDestinationNamespace
}

func (c *config) GetConsulEnableK8SNSMirroring() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulEnableK8SNSMirroring
}

func (c *config) GetConsulK8SNSMirroringPrefix() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulK8SNSMirroringPrefix
}

func (c *config) GetConsulCrossNamespaceACLPolicy() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulCrossNamespaceACLPolicy
}

func (c *config) GetConsulGenerateInternalServiceHealthCheck() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.consulCfg.consulGenerateInternalServiceHealthCheck
}

func (c *config) GetEurekaHeartBeatInstance() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.eurekaCfg.heartBeatInstance
}

func (c *config) GetEurekaHeartBeatPeriod() time.Duration {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.eurekaCfg.heartBeatPeriod
}

func (c *config) GetEurekaCheckServiceInstanceID() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.eurekaCfg.checkServiceInstanceID
}

func (c *config) GetNacosGroupId() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.nacosCfg.groupId
}

func (c *config) GetNacosClusterId() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.nacosCfg.clusterId
}

func (c *config) GetZookeeperBasePath() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.zookeeperCfg.basePath
}

func (c *config) GetZookeeperCategory() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.zookeeperCfg.category
}

func (c *config) GetZookeeperAdaptor() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.zookeeperCfg.adaptor
}

func (c *config) GetClusterId() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.clusterId
}

func (c *config) GetPassingOnly() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.passingOnly
}

func (c *config) GetC2KFilterTag() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.filterTag
}

func (c *config) GetC2KFilterMetadatas() []ctv1.Metadata {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.filterMetadatas
}

func (c *config) GetC2KFilterIPRanges() []*cidr.CIDR {
	c.flock.RLock()
	defer c.flock.RUnlock()
	var cidrs []*cidr.CIDR
	for _, ipRange := range c.c2kCfg.filterIPRanges {
		if len(ipRange) > 0 {
			if net, err := cidr.ParseCIDR(ipRange); err == nil {
				cidrs = append(cidrs, net)
			}
		}
	}
	return cidrs
}

func (c *config) GetK2CFilterIPRanges() []*cidr.CIDR {
	c.flock.RLock()
	defer c.flock.RUnlock()

	var cidrs []*cidr.CIDR
	for _, ipRange := range c.k2cCfg.filterIPRanges {
		if len(ipRange) > 0 {
			if net, err := cidr.ParseCIDR(ipRange); err == nil {
				cidrs = append(cidrs, net)
			}
		}
	}
	return cidrs
}

func (c *config) GetC2KExcludeMetadatas() []ctv1.Metadata {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.excludeMetadatas
}

func (c *config) GetC2KExcludeIPRanges() []*cidr.CIDR {
	c.flock.RLock()
	defer c.flock.RUnlock()
	var cidrs []*cidr.CIDR
	for _, ipRange := range c.c2kCfg.excludeIPRanges {
		if len(ipRange) > 0 {
			if net, err := cidr.ParseCIDR(ipRange); err == nil {
				cidrs = append(cidrs, net)
			}
		}
	}
	return cidrs
}

func (c *config) GetK2CExcludeIPRanges() []*cidr.CIDR {
	c.flock.RLock()
	defer c.flock.RUnlock()
	var cidrs []*cidr.CIDR
	for _, ipRange := range c.k2cCfg.excludeIPRanges {
		if len(ipRange) > 0 {
			if net, err := cidr.ParseCIDR(ipRange); err == nil {
				cidrs = append(cidrs, net)
			}
		}
	}
	return cidrs
}

func (c *config) GetPrefixTag() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.prefixTag
}

func (c *config) GetSuffixTag() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.suffixTag
}

func (c *config) GetPrefixMetadata() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.prefixMetadata
}

func (c *config) GetSuffixMetadata() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.suffixMetadata
}

func (c *config) EnableC2KConversions() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.enableConversions
}

func (c *config) GetC2KServiceConversions() map[string]ctv1.ServiceConversion {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.serviceConversions
}

func (c *config) GetC2KFixedHTTPServicePort() *uint32 {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.fixedHTTPServicePort
}

func (c *config) EnableC2KMetadataStrategy() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	if c.c2kCfg.metadataStrategy != nil {
		return c.c2kCfg.metadataStrategy.Enable
	}
	return false
}

func (c *config) GetC2KMetadataToLabelConversions() map[string]string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	if c.c2kCfg.metadataStrategy != nil {
		return c.c2kCfg.metadataStrategy.LabelConversions
	}
	return nil
}

func (c *config) GetC2KMetadataToAnnotationConversions() map[string]string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	if c.c2kCfg.metadataStrategy != nil {
		return c.c2kCfg.metadataStrategy.AnnotationConversions
	}
	return nil
}

func (c *config) GetC2KWithGateway() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.withGateway
}

func (c *config) GetC2KMultiGateways() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.multiGateways
}

func (c *config) GetNacos2KClusterSet() []string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.nacos2kCfg.clusterSet
}

func (c *config) GetNacos2KGroupSet() []string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.nacos2kCfg.groupSet
}

func (c *config) GetAuthNacosUsername() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.nacos.username
}

func (c *config) GetAuthNacosPassword() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.nacos.password
}

func (c *config) GetAuthNacosAccessKey() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.nacos.accessKey
}

func (c *config) GetAuthNacosSecretKey() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.nacos.secretKey
}

func (c *config) GetAuthNacosNamespaceId() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.nacos.namespaceId
}

func (c *config) GetAuthConsulUsername() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.consul.username
}

func (c *config) GetAuthConsulPassword() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.auth.consul.password
}

func (c *config) GetHTTPAddr() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.httpAddr
}

func (c *config) GetDeriveNamespace() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.deriveNamespace
}

func (c *config) Purge() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.purge
}

func (c *config) AsInternalServices() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.asInternalServices
}

func (c *config) GetSyncPeriod() time.Duration {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.syncPeriod
}

func (c *config) SyncCloudToK8s() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.enable
}

func (c *config) SyncK8sToCloud() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.enable
}
func (c *config) SyncK8sToGateway() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2gCfg.enable
}

func (c *client) initGatewayConnectorConfig(spec ctv1.GatewaySpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.purge = spec.SyncToFgw.Purge
	c.syncPeriod = spec.SyncToFgw.SyncPeriod.Duration
	if c.syncPeriod < MinSyncPeriod {
		c.syncPeriod = MinSyncPeriod
	}

	c.k2gCfg.enable = spec.SyncToFgw.Enable
	c.k2gCfg.defaultSync = spec.SyncToFgw.DefaultSync
	c.k2gCfg.allowK8sNamespacesSet = ToSet(spec.SyncToFgw.AllowK8sNamespaces)
	c.k2gCfg.denyK8sNamespacesSet = ToSet(spec.SyncToFgw.DenyK8sNamespaces)

	c.viaCfg.fgwName = spec.GatewayName

	c.viaCfg.ingressIPSelector = spec.Ingress.IPSelector
	c.viaCfg.egressIPSelector = spec.Egress.IPSelector

	c.viaCfg.ingress.httpPort = uint(spec.Ingress.HTTPPort)
	c.viaCfg.ingress.grpcPort = uint(spec.Ingress.GRPCPort)
	c.viaCfg.egress.httpPort = uint(spec.Egress.HTTPPort)
	c.viaCfg.egress.grpcPort = uint(spec.Egress.GRPCPort)
}

func (c *client) initMachineConnectorConfig(spec ctv1.MachineSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.deriveNamespace = spec.DeriveNamespace
	c.purge = spec.Purge
	c.asInternalServices = spec.AsInternalServices

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.passingOnly = spec.SyncToK8S.PassingOnly
	c.config.c2kCfg.filterTag = spec.SyncToK8S.FilterLabel
	c.config.c2kCfg.filterIPRanges = append([]string{}, spec.SyncToK8S.FilterIPRanges...)
	c.config.c2kCfg.excludeIPRanges = append([]string{}, spec.SyncToK8S.ExcludeIPRanges...)

	c.config.c2kCfg.prefixTag = spec.SyncToK8S.PrefixLabel
	c.config.c2kCfg.suffixTag = spec.SyncToK8S.SuffixLabel
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway.Enable
	c.config.c2kCfg.multiGateways = spec.SyncToK8S.WithGateway.MultiGateways

	if spec.SyncToK8S.ConversionStrategy != nil {
		c.config.c2kCfg.enableConversions = spec.SyncToK8S.ConversionStrategy.Enable
		c.config.c2kCfg.serviceConversions = make(map[string]ctv1.ServiceConversion)
		if len(spec.SyncToK8S.ConversionStrategy.ServiceConversions) > 0 {
			for _, serviceConversion := range spec.SyncToK8S.ConversionStrategy.ServiceConversions {
				c.config.c2kCfg.serviceConversions[fmt.Sprintf("%s/%s", serviceConversion.Namespace, serviceConversion.Service)] = serviceConversion
			}
		}
	} else {
		c.config.c2kCfg.enableConversions = false
		c.config.c2kCfg.serviceConversions = nil
	}
}

func (c *client) initNacosConnectorConfig(spec ctv1.NacosSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.httpAddr = spec.HTTPAddr
	c.deriveNamespace = spec.DeriveNamespace
	c.purge = spec.Purge
	c.asInternalServices = spec.AsInternalServices
	c.syncPeriod = spec.SyncPeriod.Duration
	if c.syncPeriod < MinSyncPeriod {
		c.syncPeriod = MinSyncPeriod
	}

	c.auth.nacos.username = spec.Auth.Username
	c.auth.nacos.password = spec.Auth.Password
	c.auth.nacos.accessKey = spec.Auth.AccessKey
	c.auth.nacos.secretKey = spec.Auth.SecretKey
	c.auth.nacos.namespaceId = spec.Auth.NamespaceId

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.passingOnly = spec.SyncToK8S.PassingOnly
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.filterIPRanges = append([]string{}, spec.SyncToK8S.FilterIPRanges...)
	c.config.c2kCfg.excludeMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.ExcludeMetadatas...)
	c.config.c2kCfg.excludeIPRanges = append([]string{}, spec.SyncToK8S.ExcludeIPRanges...)
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.fixedHTTPServicePort = spec.SyncToK8S.FixedHTTPServicePort
	c.config.c2kCfg.metadataStrategy = spec.SyncToK8S.MetadataStrategy
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway.Enable
	c.config.c2kCfg.multiGateways = spec.SyncToK8S.WithGateway.MultiGateways
	if len(spec.SyncToK8S.ClusterSet) == 0 {
		c.config.c2kCfg.nacos2kCfg.clusterSet = []string{connector.NACOS_DEFAULT_CLUSTER}
	} else {
		c.config.c2kCfg.nacos2kCfg.clusterSet = append([]string{}, spec.SyncToK8S.ClusterSet...)
	}
	if len(spec.SyncToK8S.GroupSet) == 0 {
		c.config.c2kCfg.nacos2kCfg.groupSet = []string{constant.DEFAULT_GROUP}
	} else {
		c.config.c2kCfg.nacos2kCfg.groupSet = append([]string{}, spec.SyncToK8S.GroupSet...)
	}

	if spec.SyncToK8S.ConversionStrategy != nil {
		c.config.c2kCfg.enableConversions = spec.SyncToK8S.ConversionStrategy.Enable
		c.config.c2kCfg.serviceConversions = make(map[string]ctv1.ServiceConversion)
		if len(spec.SyncToK8S.ConversionStrategy.ServiceConversions) > 0 {
			for _, serviceConversion := range spec.SyncToK8S.ConversionStrategy.ServiceConversions {
				c.config.c2kCfg.serviceConversions[fmt.Sprintf("%s/%s", serviceConversion.Namespace, serviceConversion.Service)] = serviceConversion
			}
		}
	} else {
		c.config.c2kCfg.enableConversions = false
		c.config.c2kCfg.serviceConversions = nil
	}

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.defaultSync = spec.SyncFromK8S.DefaultSync
	c.k2cCfg.syncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	c.k2cCfg.syncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	c.k2cCfg.nodePortSyncType = spec.SyncFromK8S.NodePortSyncType
	c.k2cCfg.syncIngress = spec.SyncFromK8S.SyncIngress
	c.k2cCfg.syncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	c.k2cCfg.addServicePrefix = spec.SyncFromK8S.AddServicePrefix
	c.k2cCfg.addK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix
	c.k2cCfg.appendMetadataSet = ToMetaSet(spec.SyncFromK8S.AppendMetadatas)
	c.k2cCfg.allowK8sNamespacesSet = ToSet(spec.SyncFromK8S.AllowK8sNamespaces)
	c.k2cCfg.denyK8sNamespacesSet = ToSet(spec.SyncFromK8S.DenyK8sNamespaces)
	c.k2cCfg.filterIPRanges = append([]string{}, spec.SyncFromK8S.FilterIPRanges...)
	c.k2cCfg.excludeIPRanges = append([]string{}, spec.SyncFromK8S.ExcludeIPRanges...)
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway.Enable
	c.k2cCfg.withGatewayMode = spec.SyncFromK8S.WithGateway.GatewayMode

	c.k2cCfg.nacosCfg.clusterId = spec.SyncFromK8S.ClusterId
	c.k2cCfg.nacosCfg.groupId = spec.SyncFromK8S.GroupId

	c.limiter.SetLimit(rate.Limit(spec.Limiter.Limit))
	c.limiter.SetBurst(int(spec.Limiter.Limit))
}

func (c *client) initEurekaConnectorConfig(spec ctv1.EurekaSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.httpAddr = spec.HTTPAddr
	c.deriveNamespace = spec.DeriveNamespace
	c.purge = spec.Purge
	c.asInternalServices = spec.AsInternalServices
	c.syncPeriod = spec.SyncPeriod.Duration
	if c.syncPeriod < MinSyncPeriod {
		c.syncPeriod = MinSyncPeriod
	}

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.filterIPRanges = append([]string{}, spec.SyncToK8S.FilterIPRanges...)
	c.config.c2kCfg.excludeMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.ExcludeMetadatas...)
	c.config.c2kCfg.excludeIPRanges = append([]string{}, spec.SyncToK8S.ExcludeIPRanges...)
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.fixedHTTPServicePort = spec.SyncToK8S.FixedHTTPServicePort
	c.config.c2kCfg.metadataStrategy = spec.SyncToK8S.MetadataStrategy
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway.Enable
	c.config.c2kCfg.multiGateways = spec.SyncToK8S.WithGateway.MultiGateways

	if spec.SyncToK8S.ConversionStrategy != nil {
		c.config.c2kCfg.enableConversions = spec.SyncToK8S.ConversionStrategy.Enable
		c.config.c2kCfg.serviceConversions = make(map[string]ctv1.ServiceConversion)
		if len(spec.SyncToK8S.ConversionStrategy.ServiceConversions) > 0 {
			for _, serviceConversion := range spec.SyncToK8S.ConversionStrategy.ServiceConversions {
				c.config.c2kCfg.serviceConversions[fmt.Sprintf("%s/%s", serviceConversion.Namespace, serviceConversion.Service)] = serviceConversion
			}
		}
	} else {
		c.config.c2kCfg.enableConversions = false
		c.config.c2kCfg.serviceConversions = nil
	}

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.defaultSync = spec.SyncFromK8S.DefaultSync
	c.k2cCfg.syncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	c.k2cCfg.syncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	c.k2cCfg.nodePortSyncType = spec.SyncFromK8S.NodePortSyncType
	c.k2cCfg.syncIngress = spec.SyncFromK8S.SyncIngress
	c.k2cCfg.syncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	c.k2cCfg.addServicePrefix = spec.SyncFromK8S.AddServicePrefix
	c.k2cCfg.addK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix
	c.k2cCfg.appendMetadataSet = ToMetaSet(spec.SyncFromK8S.AppendMetadatas)
	c.k2cCfg.allowK8sNamespacesSet = ToSet(spec.SyncFromK8S.AllowK8sNamespaces)
	c.k2cCfg.denyK8sNamespacesSet = ToSet(spec.SyncFromK8S.DenyK8sNamespaces)
	c.k2cCfg.filterIPRanges = append([]string{}, spec.SyncFromK8S.FilterIPRanges...)
	c.k2cCfg.excludeIPRanges = append([]string{}, spec.SyncFromK8S.ExcludeIPRanges...)
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway.Enable
	c.k2cCfg.withGatewayMode = spec.SyncFromK8S.WithGateway.GatewayMode

	c.k2cCfg.eurekaCfg.checkServiceInstanceID = spec.SyncFromK8S.CheckServiceInstanceID
	c.k2cCfg.eurekaCfg.heartBeatInstance = spec.SyncFromK8S.HeartBeatInstance
	c.k2cCfg.eurekaCfg.heartBeatPeriod = spec.SyncFromK8S.HeartBeatPeriod.Duration
	if c.k2cCfg.eurekaCfg.heartBeatPeriod < MinSyncPeriod {
		c.k2cCfg.eurekaCfg.heartBeatPeriod = MinSyncPeriod
	}

	c.limiter.SetLimit(rate.Limit(spec.Limiter.Limit))
	c.limiter.SetBurst(int(spec.Limiter.Limit))
}

func (c *client) initConsulConnectorConfig(spec ctv1.ConsulSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.httpAddr = spec.HTTPAddr
	c.deriveNamespace = spec.DeriveNamespace
	c.purge = spec.Purge
	c.asInternalServices = spec.AsInternalServices
	c.syncPeriod = spec.SyncPeriod.Duration
	if c.syncPeriod < MinSyncPeriod {
		c.syncPeriod = MinSyncPeriod
	}

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.passingOnly = spec.SyncToK8S.PassingOnly
	c.config.c2kCfg.filterTag = spec.SyncToK8S.FilterTag
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.filterIPRanges = append([]string{}, spec.SyncToK8S.FilterIPRanges...)
	c.config.c2kCfg.excludeMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.ExcludeMetadatas...)
	c.config.c2kCfg.excludeIPRanges = append([]string{}, spec.SyncToK8S.ExcludeIPRanges...)
	c.config.c2kCfg.prefixTag = spec.SyncToK8S.PrefixTag
	c.config.c2kCfg.suffixTag = spec.SyncToK8S.SuffixTag
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.fixedHTTPServicePort = spec.SyncToK8S.FixedHTTPServicePort
	c.config.c2kCfg.metadataStrategy = spec.SyncToK8S.MetadataStrategy
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway.Enable
	c.config.c2kCfg.multiGateways = spec.SyncToK8S.WithGateway.MultiGateways

	if spec.SyncToK8S.ConversionStrategy != nil {
		c.config.c2kCfg.enableConversions = spec.SyncToK8S.ConversionStrategy.Enable
		c.config.c2kCfg.serviceConversions = make(map[string]ctv1.ServiceConversion)
		if len(spec.SyncToK8S.ConversionStrategy.ServiceConversions) > 0 {
			for _, serviceConversion := range spec.SyncToK8S.ConversionStrategy.ServiceConversions {
				c.config.c2kCfg.serviceConversions[fmt.Sprintf("%s/%s", serviceConversion.Namespace, serviceConversion.Service)] = serviceConversion
			}
		}
	} else {
		c.config.c2kCfg.enableConversions = false
		c.config.c2kCfg.serviceConversions = nil
	}

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.defaultSync = spec.SyncFromK8S.DefaultSync
	c.k2cCfg.syncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	c.k2cCfg.syncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	c.k2cCfg.nodePortSyncType = spec.SyncFromK8S.NodePortSyncType
	c.k2cCfg.syncIngress = spec.SyncFromK8S.SyncIngress
	c.k2cCfg.syncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	c.k2cCfg.addServicePrefix = spec.SyncFromK8S.AddServicePrefix
	c.k2cCfg.addK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix
	c.k2cCfg.appendTagSet = ToSet(spec.SyncFromK8S.AppendTags)
	c.k2cCfg.appendMetadataSet = ToMetaSet(spec.SyncFromK8S.AppendMetadatas)
	c.k2cCfg.allowK8sNamespacesSet = ToSet(spec.SyncFromK8S.AllowK8sNamespaces)
	c.k2cCfg.denyK8sNamespacesSet = ToSet(spec.SyncFromK8S.DenyK8sNamespaces)
	c.k2cCfg.filterIPRanges = append([]string{}, spec.SyncFromK8S.FilterIPRanges...)
	c.k2cCfg.excludeIPRanges = append([]string{}, spec.SyncFromK8S.ExcludeIPRanges...)
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway.Enable
	c.k2cCfg.withGatewayMode = spec.SyncFromK8S.WithGateway.GatewayMode

	c.k2cCfg.consulCfg.consulNodeName = spec.SyncFromK8S.ConsulNodeName
	c.k2cCfg.consulCfg.consulEnableNamespaces = spec.SyncFromK8S.ConsulEnableNamespaces
	c.k2cCfg.consulCfg.consulDestinationNamespace = spec.SyncFromK8S.ConsulDestinationNamespace
	c.k2cCfg.consulCfg.consulEnableK8SNSMirroring = spec.SyncFromK8S.ConsulEnableK8SNSMirroring
	c.k2cCfg.consulCfg.consulK8SNSMirroringPrefix = spec.SyncFromK8S.ConsulK8SNSMirroringPrefix
	c.k2cCfg.consulCfg.consulCrossNamespaceACLPolicy = spec.SyncFromK8S.ConsulCrossNamespaceACLPolicy
	c.k2cCfg.consulCfg.consulGenerateInternalServiceHealthCheck = spec.SyncToK8S.GenerateInternalServiceHealthCheck

	c.limiter.SetLimit(rate.Limit(spec.Limiter.Limit))
	c.limiter.SetBurst(int(spec.Limiter.Limit))
}

func (c *client) initZookeeperConnectorConfig(spec ctv1.ZookeeperSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.httpAddr = spec.HTTPAddr
	c.deriveNamespace = spec.DeriveNamespace
	c.purge = spec.Purge
	c.asInternalServices = spec.AsInternalServices
	c.syncPeriod = spec.SyncPeriod.Duration
	if c.syncPeriod < MinSyncPeriod {
		c.syncPeriod = MinSyncPeriod
	}

	c.auth.zookeeper.password = spec.Auth.Password

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.filterIPRanges = append([]string{}, spec.SyncToK8S.FilterIPRanges...)
	c.config.c2kCfg.excludeMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.ExcludeMetadatas...)
	c.config.c2kCfg.excludeIPRanges = append([]string{}, spec.SyncToK8S.ExcludeIPRanges...)
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.fixedHTTPServicePort = spec.SyncToK8S.FixedHTTPServicePort
	c.config.c2kCfg.metadataStrategy = spec.SyncToK8S.MetadataStrategy
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway.Enable
	c.config.c2kCfg.multiGateways = spec.SyncToK8S.WithGateway.MultiGateways

	if spec.SyncToK8S.ConversionStrategy != nil {
		c.config.c2kCfg.enableConversions = spec.SyncToK8S.ConversionStrategy.Enable
		c.config.c2kCfg.serviceConversions = make(map[string]ctv1.ServiceConversion)
		if len(spec.SyncToK8S.ConversionStrategy.ServiceConversions) > 0 {
			for _, serviceConversion := range spec.SyncToK8S.ConversionStrategy.ServiceConversions {
				c.config.c2kCfg.serviceConversions[fmt.Sprintf("%s/%s", serviceConversion.Namespace, serviceConversion.Service)] = serviceConversion
			}
		}
	} else {
		c.config.c2kCfg.enableConversions = false
		c.config.c2kCfg.serviceConversions = nil
	}

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.defaultSync = spec.SyncFromK8S.DefaultSync
	c.k2cCfg.syncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	c.k2cCfg.syncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	c.k2cCfg.nodePortSyncType = spec.SyncFromK8S.NodePortSyncType
	c.k2cCfg.syncIngress = spec.SyncFromK8S.SyncIngress
	c.k2cCfg.syncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	c.k2cCfg.addServicePrefix = spec.SyncFromK8S.AddServicePrefix
	c.k2cCfg.addK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix
	c.k2cCfg.appendMetadataSet = ToMetaSet(spec.SyncFromK8S.AppendMetadatas)
	c.k2cCfg.allowK8sNamespacesSet = ToSet(spec.SyncFromK8S.AllowK8sNamespaces)
	c.k2cCfg.denyK8sNamespacesSet = ToSet(spec.SyncFromK8S.DenyK8sNamespaces)
	c.k2cCfg.filterIPRanges = append([]string{}, spec.SyncFromK8S.FilterIPRanges...)
	c.k2cCfg.excludeIPRanges = append([]string{}, spec.SyncFromK8S.ExcludeIPRanges...)
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway.Enable
	c.k2cCfg.withGatewayMode = spec.SyncFromK8S.WithGateway.GatewayMode

	c.k2cCfg.zookeeperCfg.basePath = spec.BasePath
	c.k2cCfg.zookeeperCfg.category = spec.Category
	c.k2cCfg.zookeeperCfg.adaptor = spec.Adaptor

	c.limiter.SetLimit(rate.Limit(spec.Limiter.Limit))
	c.limiter.SetBurst(int(spec.Limiter.Limit))
}
