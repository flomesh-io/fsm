package cli

import (
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"golang.org/x/time/rate"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	// SyncPeriod is how often the syncer will attempt to
	// reconcile the expected service states with the remote cloud server.
	SyncPeriod = 5 * time.Second
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
	}

	httpAddr           string
	deriveNamespace    string
	asInternalServices bool

	c2kCfg struct {
		enable          bool
		clusterId       string
		passingOnly     bool
		filterTag       string
		filterMetadatas []ctv1.Metadata

		prefix         string // prefix is a prefix to prepend to services
		prefixTag      string
		suffixTag      string
		prefixMetadata string
		suffixMetadata string

		withGateway bool

		nacos2kCfg struct {
			clusterSet []string
			groupSet   []string
		}
	}

	k2cCfg struct {
		enable bool

		// syncPeriod is the interval between full catalog syncs. These will
		// re-register all services to prevent overwrites of data. This should
		// happen relatively infrequently and default to 5 seconds.
		syncPeriod time.Duration

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

		withGateway bool

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
		}

		nacosCfg struct {
			clusterId string
			groupId   string
		}
	}

	k2gCfg struct {
		enable bool

		syncPeriod time.Duration

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
		ingressIPSelector ctv1.AddrSelector
		egressIPSelector  ctv1.AddrSelector

		ingressAddr string
		egressAddr  string

		ingress protocolPort
		egress  protocolPort
	}
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

func (c *config) GetK2GSyncPeriod() time.Duration {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2gCfg.syncPeriod
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

func (c *config) GetSyncPeriod() time.Duration {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.k2cCfg.syncPeriod
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

func (c *config) GetFilterTag() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.filterTag
}

func (c *config) GetFilterMetadatas() []ctv1.Metadata {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.filterMetadatas
}

func (c *config) GetPrefix() string {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.prefix
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

func (c *config) GetC2KWithGateway() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.c2kCfg.withGateway
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

func (c *config) AsInternalServices() bool {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.asInternalServices
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

	c.k2gCfg.enable = spec.SyncToFgw.Enable
	c.k2gCfg.syncPeriod = SyncPeriod
	c.k2gCfg.defaultSync = spec.SyncToFgw.DefaultSync
	c.k2gCfg.allowK8sNamespacesSet = ToSet(spec.SyncToFgw.AllowK8sNamespaces)
	c.k2gCfg.denyK8sNamespacesSet = ToSet(spec.SyncToFgw.DenyK8sNamespaces)

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
	c.asInternalServices = spec.AsInternalServices

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.passingOnly = spec.SyncToK8S.PassingOnly
	c.config.c2kCfg.filterTag = spec.SyncToK8S.FilterLabel
	c.config.c2kCfg.prefixTag = spec.SyncToK8S.PrefixLabel
	c.config.c2kCfg.suffixTag = spec.SyncToK8S.SuffixLabel
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway
}

func (c *client) initNacosConnectorConfig(spec ctv1.NacosSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.httpAddr = spec.HTTPAddr
	c.deriveNamespace = spec.DeriveNamespace
	c.asInternalServices = spec.AsInternalServices

	c.auth.nacos.username = spec.Auth.Username
	c.auth.nacos.password = spec.Auth.Password
	c.auth.nacos.accessKey = spec.Auth.AccessKey
	c.auth.nacos.secretKey = spec.Auth.SecretKey
	c.auth.nacos.namespaceId = spec.Auth.NamespaceId

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.passingOnly = spec.SyncToK8S.PassingOnly
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway
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

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.syncPeriod = SyncPeriod
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
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway

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
	c.asInternalServices = spec.AsInternalServices

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.syncPeriod = SyncPeriod
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
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway

	c.limiter.SetLimit(rate.Limit(spec.Limiter.Limit))
	c.limiter.SetBurst(int(spec.Limiter.Limit))
}

func (c *client) initConsulConnectorConfig(spec ctv1.ConsulSpec) {
	c.flock.Lock()
	defer c.flock.Unlock()

	c.httpAddr = spec.HTTPAddr
	c.deriveNamespace = spec.DeriveNamespace
	c.asInternalServices = spec.AsInternalServices

	c.config.c2kCfg.enable = spec.SyncToK8S.Enable
	c.config.c2kCfg.clusterId = spec.SyncToK8S.ClusterId
	c.config.c2kCfg.passingOnly = spec.SyncToK8S.PassingOnly
	c.config.c2kCfg.filterTag = spec.SyncToK8S.FilterTag
	c.config.c2kCfg.prefixTag = spec.SyncToK8S.PrefixTag
	c.config.c2kCfg.suffixTag = spec.SyncToK8S.SuffixTag
	c.config.c2kCfg.filterMetadatas = append([]ctv1.Metadata{}, spec.SyncToK8S.FilterMetadatas...)
	c.config.c2kCfg.prefixMetadata = spec.SyncToK8S.PrefixMetadata
	c.config.c2kCfg.suffixMetadata = spec.SyncToK8S.SuffixMetadata
	c.config.c2kCfg.withGateway = spec.SyncToK8S.WithGateway

	c.k2cCfg.enable = spec.SyncFromK8S.Enable
	c.k2cCfg.syncPeriod = SyncPeriod
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
	c.k2cCfg.withGateway = spec.SyncFromK8S.WithGateway

	c.k2cCfg.consulCfg.consulNodeName = spec.SyncFromK8S.ConsulNodeName
	c.k2cCfg.consulCfg.consulEnableNamespaces = spec.SyncFromK8S.ConsulEnableNamespaces
	c.k2cCfg.consulCfg.consulDestinationNamespace = spec.SyncFromK8S.ConsulDestinationNamespace
	c.k2cCfg.consulCfg.consulEnableK8SNSMirroring = spec.SyncFromK8S.ConsulEnableK8SNSMirroring
	c.k2cCfg.consulCfg.consulK8SNSMirroringPrefix = spec.SyncFromK8S.ConsulK8SNSMirroringPrefix
	c.k2cCfg.consulCfg.consulCrossNamespaceACLPolicy = spec.SyncFromK8S.ConsulCrossNamespaceACLPolicy

	c.limiter.SetLimit(rate.Limit(spec.Limiter.Limit))
	c.limiter.SetBurst(int(spec.Limiter.Limit))
}
