package repo

import (
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/registry"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

var (
	log = logger.New("flomesh-pipy")
)

// Server implements the Aggregate Discovery Services
type Server struct {
	catalog        catalog.MeshCataloger
	proxyRegistry  *registry.ProxyRegistry
	fsmNamespace   string
	cfg            configurator.Configurator
	certManager    *certificate.Manager
	ready          bool
	workQueues     *workerpool.WorkerPool
	kubeController k8s.Controller

	// When snapshot cache is enabled, we (currently) don't keep track of proxy information, however different
	// config versions have to be provided to the cache as we keep adding snapshots. The following map
	// tracks at which version we are at given a proxy UUID
	configVerMutex sync.Mutex
	configVersion  map[string]uint64

	pluginSet        mapset.Set
	pluginPri        map[string]float32
	pluginSetVersion string
	pluginMutex      sync.RWMutex

	msgBroker *messaging.Broker

	repoClient *client.PipyRepoClient

	retryProxiesJob func()
}

// Protocol is a string wrapper type
type Protocol string

// Address is a string wrapper type
type Address string

// Port is a uint16 wrapper type
type Port uint16

// Weight is a uint32 wrapper type
type Weight uint32

// ClusterName is a string wrapper type
type ClusterName string

// WeightedEndpoint is a wrapper type of map[HTTPHostPort]Weight
type WeightedEndpoint map[HTTPHostPort]Weight

// Header is a string wrapper type
type Header string

// HeaderRegexp is a string wrapper type
type HeaderRegexp string

// Headers is a wrapper type of map[Header]HeaderRegexp
type Headers map[Header]HeaderRegexp

// Method is a string wrapper type
type Method string

// Methods is a wrapper type of []Method
type Methods []Method

// WeightedClusters is a wrapper type of map[ClusterName]Weight
type WeightedClusters map[ClusterName]Weight

// URIPathValue is a uri value wrapper
type URIPathValue string

// URIMatchType is a match type wrapper
type URIMatchType string

const (
	// PathMatchRegex is the type used to specify regex based path matching
	PathMatchRegex URIMatchType = "Regex"

	// PathMatchExact is the type used to specify exact path matching
	PathMatchExact URIMatchType = "Exact"

	// PathMatchPrefix is the type used to specify prefix based path matching
	PathMatchPrefix URIMatchType = "Prefix"
)

// PluginSlice plugin array
type PluginSlice []trafficpolicy.Plugin

// Pluggable is the base struct supported plugin
type Pluggable struct {
	Plugins map[string]*runtime.RawExtension `json:"Plugins,omitempty"`
}

// URIPath is a uri wrapper type
type URIPath struct {
	Value URIPathValue
	Type  URIMatchType
}

// ServiceName is a string wrapper type
type ServiceName string

// Services is a wrapper type of []ServiceName
type Services []ServiceName

// HTTPMatchRule http match rule
type HTTPMatchRule struct {
	Path              URIPathValue
	Type              URIMatchType
	Headers           Headers `json:"Headers,omitempty"`
	Methods           Methods `json:"Methods,omitempty"`
	allowedAnyService bool
	allowedAnyMethod  bool
}

// HTTPRouteRule http route rule
type HTTPRouteRule struct {
	HTTPMatchRule
	TargetClusters  WeightedClusters `json:"TargetClusters"`
	AllowedServices Services         `json:"AllowedServices,omitempty"`
}

// HTTPRouteRuleName is a string wrapper type
type HTTPRouteRuleName string

// HTTPRouteRuleRef http route rule name
type HTTPRouteRuleRef struct {
	RuleName HTTPRouteRuleName `json:"RuleName"`
	Service  string            `json:"Service,omitempty"`
}

// HTTPHostPort is a string wrapper type
type HTTPHostPort string

// HTTPHostPort2Service is a wrapper type of map[HTTPHostPort]HTTPRouteRuleRef
type HTTPHostPort2Service map[HTTPHostPort]*HTTPRouteRuleRef

// DestinationIPRange is a string wrapper type
type DestinationIPRange string

// DestinationSecuritySpec is the security spec of destination
type DestinationSecuritySpec struct {
	SourceCert *Certificate `json:"SourceCert,omitempty"`
}

// DestinationIPRanges is a wrapper type of map[DestinationIPRange]*DestinationSecuritySpec
type DestinationIPRanges map[DestinationIPRange]*DestinationSecuritySpec

// SourceIPRange is a string wrapper type
type SourceIPRange string

// SourceSecuritySpec is the security spec of source
type SourceSecuritySpec struct {
	MTLS                     bool `json:"mTLS"`
	SkipClientCertValidation bool
	AuthenticatedPrincipals  []string
}

// SourceIPRanges is a wrapper type of map[SourceIPRange]*SourceSecuritySpec
type SourceIPRanges map[SourceIPRange]*SourceSecuritySpec

// AllowedEndpoints is a wrapper type of map[Address]ServiceName
type AllowedEndpoints map[Address]ServiceName

// FeatureFlags represents the flags of feature
type FeatureFlags struct {
	EnableSidecarActiveHealthChecks bool
	EnableAutoDefaultRoute          bool
}

// TrafficSpec represents the spec of traffic
type TrafficSpec struct {
	EnableEgress                      bool
	enablePermissiveTrafficPolicyMode bool
	HTTP1PerRequestLoadBalancing      bool
	HTTP2PerRequestLoadBalancing      bool
}

// TracingSpec is the type to represent tracing configuration.
type TracingSpec struct {
	// Address defines the tracing collectio's hostname.
	Address string `json:"address,omitempty"`

	// Endpoint defines the API endpoint for tracing requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`

	// SampledFraction defines the sampled fraction.
	SampledFraction string `json:"sampledFraction,omitempty"`
}

// RemoteLoggingSpec is the type to represent remote logging configuration.
type RemoteLoggingSpec struct {
	// Level defines the remote logging's level.
	Level uint16 `json:"level,omitempty"`
	// Address defines the remote logging's hostname.
	Address string `json:"address,omitempty"`
	// Endpoint defines the API endpoint for remote logging requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`
	// Authorization defines the access entity that allows to authorize someone in remote logging service.
	Authorization string `json:"authorization,omitempty"`
	// SampledFraction defines the sampled fraction.
	SampledFraction string `json:"sampledFraction,omitempty"`
}

// ObservabilitySpec is the type to represent OSM's observability configurations.
type ObservabilitySpec struct {
	// Tracing defines OSM's tracing configuration.
	Tracing *TracingSpec `json:"tracing,omitempty"`

	// RemoteLogging defines OSM's remote logging configuration.
	RemoteLogging *RemoteLoggingSpec `json:"remoteLogging,omitempty"`
}

// MeshConfigSpec represents the spec of mesh config
type MeshConfigSpec struct {
	ServiceIdentity identity.ServiceIdentity
	SidecarLogLevel string
	SidecarTimeout  int
	Traffic         TrafficSpec
	FeatureFlags    FeatureFlags
	Probes          struct {
		ReadinessProbes []v1.Probe `json:"ReadinessProbes,omitempty"`
		LivenessProbes  []v1.Probe `json:"LivenessProbes,omitempty"`
		StartupProbes   []v1.Probe `json:"StartupProbes,omitempty"`
	}
	ClusterSet    map[string]string `json:"ClusterSet,omitempty"`
	Observability ObservabilitySpec `json:"Observability,omitempty"`

	sidecarCompressConfig bool
}

// Certificate represents an x509 certificate.
type Certificate struct {
	// If issued by fsm ca
	FsmIssued *bool `json:"FsmIssued,omitempty"`

	// The CommonName of the certificate
	CommonName *certificate.CommonName `json:"CommonName,omitempty"`

	// SubjectAltNames defines the Subject Alternative Names (domain names and IP addresses) secured by the certificate.
	SubjectAltNames []string `json:"SubjectAltNames,omitempty"`

	// When the cert expires
	Expiration string

	// PEM encoded Certificate and Key (byte arrays)
	CertChain  string
	PrivateKey string

	// Certificate authority signing this certificate
	IssuingCA string
}

// RetryPolicy is the type used to represent the retry policy specified in the Retry policy specification.
type RetryPolicy struct {
	// RetryOn defines the policies to retry on, delimited by comma.
	RetryOn string `json:"RetryOn"`

	// PerTryTimeout defines the time allowed for a retry before it's considered a failed attempt.
	// +optional
	PerTryTimeout *float64 `json:"PerTryTimeout"`

	// NumRetries defines the max number of retries to attempt.
	// +optional
	NumRetries *uint32 `json:"NumRetries"`

	// RetryBackoffBaseInterval defines the base interval for exponential retry backoff.
	// +optional
	RetryBackoffBaseInterval *float64 `json:"RetryBackoffBaseInterval"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	service.WeightedCluster
	RetryPolicy *v1alpha1.RetryPolicySpec
}

// InboundHTTPRouteRule http route rule
type InboundHTTPRouteRule struct {
	HTTPRouteRule
	RateLimit *HTTPPerRouteRateLimit `json:"RateLimit,omitempty"`
}

// InboundHTTPRouteRuleSlice http route rule array
type InboundHTTPRouteRuleSlice []*InboundHTTPRouteRule

// InboundHTTPRouteRules is a wrapper type
type InboundHTTPRouteRules struct {
	RouteRules InboundHTTPRouteRuleSlice `json:"RouteRules"`
	Pluggable
	HTTPRateLimit    *HTTPRateLimit   `json:"RateLimit,omitempty"`
	AllowedEndpoints AllowedEndpoints `json:"AllowedEndpoints,omitempty"`
}

// InboundHTTPServiceRouteRules is a wrapper type of map[HTTPRouteRuleName]*InboundHTTPRouteRules
type InboundHTTPServiceRouteRules map[HTTPRouteRuleName]*InboundHTTPRouteRules

// InboundTCPServiceRouteRules is a wrapper type
type InboundTCPServiceRouteRules struct {
	TargetClusters WeightedClusters `json:"TargetClusters"`
	Pluggable
}

// InboundTrafficMatch represents the match of InboundTraffic
type InboundTrafficMatch struct {
	Port                  Port                         `json:"Port"`
	Protocol              Protocol                     `json:"Protocol"`
	SourceIPRanges        SourceIPRanges               `json:"SourceIPRanges,omitempty"`
	HTTPHostPort2Service  HTTPHostPort2Service         `json:"HttpHostPort2Service,omitempty"`
	HTTPServiceRouteRules InboundHTTPServiceRouteRules `json:"HttpServiceRouteRules,omitempty"`
	TCPServiceRouteRules  *InboundTCPServiceRouteRules `json:"TcpServiceRouteRules,omitempty"`
	TCPRateLimit          *TCPRateLimit                `json:"RateLimit,omitempty"`
}

// InboundTrafficMatches is a wrapper type of map[Port]*InboundTrafficMatch
type InboundTrafficMatches map[Port]*InboundTrafficMatch

// OutboundHTTPRouteRule http route rule
type OutboundHTTPRouteRule struct {
	HTTPRouteRule
}

// OutboundHTTPRouteRuleSlice http route rule array
type OutboundHTTPRouteRuleSlice []*OutboundHTTPRouteRule

// OutboundHTTPRouteRules is a wrapper type
type OutboundHTTPRouteRules struct {
	RouteRules           OutboundHTTPRouteRuleSlice `json:"RouteRules"`
	EgressForwardGateway *string                    `json:"EgressForwardGateway,omitempty"`
	Pluggable
}

// OutboundHTTPServiceRouteRules is a wrapper type of map[HTTPRouteRuleName]*HTTPRouteRules
type OutboundHTTPServiceRouteRules map[HTTPRouteRuleName]*OutboundHTTPRouteRules

// OutboundTCPServiceRouteRules is a wrapper type
type OutboundTCPServiceRouteRules struct {
	TargetClusters       WeightedClusters `json:"TargetClusters"`
	AllowedEgressTraffic bool
	EgressForwardGateway *string `json:"EgressForwardGateway,omitempty"`
	Pluggable
}

// OutboundTrafficMatch represents the match of OutboundTraffic
type OutboundTrafficMatch struct {
	DestinationIPRanges   DestinationIPRanges
	Port                  Port                          `json:"Port"`
	Protocol              Protocol                      `json:"Protocol"`
	HTTPHostPort2Service  HTTPHostPort2Service          `json:"HttpHostPort2Service,omitempty"`
	HTTPServiceRouteRules OutboundHTTPServiceRouteRules `json:"HttpServiceRouteRules,omitempty"`
	TCPServiceRouteRules  *OutboundTCPServiceRouteRules `json:"TcpServiceRouteRules,omitempty"`
}

// OutboundTrafficMatchSlice is a wrapper type of []*OutboundTrafficMatch
type OutboundTrafficMatchSlice []*OutboundTrafficMatch

// OutboundTrafficMatches is a wrapper type of map[Port][]*OutboundTrafficMatch
type OutboundTrafficMatches map[Port]OutboundTrafficMatchSlice

// namedOutboundTrafficMatches is a wrapper type of map[string]*OutboundTrafficMatch
type namedOutboundTrafficMatches map[string]*OutboundTrafficMatch

// InboundTrafficPolicy represents the policy of InboundTraffic
type InboundTrafficPolicy struct {
	TrafficMatches  InboundTrafficMatches             `json:"TrafficMatches"`
	ClustersConfigs map[ClusterName]*WeightedEndpoint `json:"ClustersConfigs"`
}

// WeightedZoneEndpoint represents the endpoint with zone and weight
type WeightedZoneEndpoint struct {
	Weight      Weight `json:"Weight"`
	Cluster     string `json:"Key,omitempty"`
	LBType      string `json:"-"`
	ContextPath string `json:"Path,omitempty"`
	ViaGateway  string `json:"ViaGateway,omitempty"`
}

// WeightedEndpoints is a wrapper type of map[HTTPHostPort]WeightedZoneEndpoint
type WeightedEndpoints map[HTTPHostPort]*WeightedZoneEndpoint

// ClusterConfig represents the configs of Cluster
type ClusterConfig struct {
	Endpoints          *WeightedEndpoints  `json:"Endpoints"`
	ConnectionSettings *ConnectionSettings `json:"ConnectionSettings,omitempty"`
	RetryPolicy        *RetryPolicy        `json:"RetryPolicy,omitempty"`
	SourceCert         *Certificate        `json:"SourceCert,omitempty"`
	Hash               uint64              `json:"Hash,omitempty"`
}

// EgressGatewayClusterConfigs represents the configs of Egress Gateway Cluster
type EgressGatewayClusterConfigs struct {
	ClusterConfig
	Mode string `json:"Mode"`
}

// OutboundTrafficPolicy represents the policy of OutboundTraffic
type OutboundTrafficPolicy struct {
	namedTrafficMatches namedOutboundTrafficMatches
	TrafficMatches      OutboundTrafficMatches         `json:"TrafficMatches"`
	ClustersConfigs     map[ClusterName]*ClusterConfig `json:"ClustersConfigs"`
}

// ForwardTrafficMatches is a wrapper type of map[Port]WeightedClusters
type ForwardTrafficMatches map[string]WeightedClusters

// ForwardTrafficPolicy represents the policy of Egress Gateway
type ForwardTrafficPolicy struct {
	ForwardMatches ForwardTrafficMatches                        `json:"ForwardMatches"`
	EgressGateways map[ClusterName]*EgressGatewayClusterConfigs `json:"EgressGateways"`
}

// ConnectionSettings defines the connection settings for an
// upstream host.
type ConnectionSettings struct {
	// TCP specifies the TCP level connection settings.
	// Applies to both TCP and HTTP connections.
	// +optional
	TCP *TCPConnectionSettings `json:"tcp,omitempty"`

	// HTTP specifies the HTTP level connection settings.
	// +optional
	HTTP *HTTPConnectionSettings `json:"http,omitempty"`
}

// TCPConnectionSettings defines the TCP connection settings for an
// upstream host.
type TCPConnectionSettings struct {
	// MaxConnections specifies the maximum number of TCP connections
	// allowed to the upstream host.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxConnections *uint32 `json:"MaxConnections,omitempty"`

	// ConnectTimeout specifies the TCP connection timeout.
	// Defaults to 5s if not specified.
	// +optional
	ConnectTimeout *float64 `json:"ConnectTimeout,omitempty"`
}

// HTTPCircuitBreaking defines the HTTP Circuit Breaking settings for an
// upstream host.
type HTTPCircuitBreaking struct {
	// StatTimeWindow specifies statistical time period of circuit breaking
	StatTimeWindow *float64 `json:"StatTimeWindow"`

	// MinRequestAmount specifies minimum number of requests (in an active statistic time span) that can trigger circuit breaking.
	MinRequestAmount uint32 `json:"MinRequestAmount"`

	// DegradedTimeWindow specifies the duration of circuit breaking
	DegradedTimeWindow *float64 `json:"DegradedTimeWindow"`

	// SlowTimeThreshold specifies the time threshold of slow request
	SlowTimeThreshold *float64 `json:"SlowTimeThreshold,omitempty"`

	// SlowAmountThreshold specifies the amount threshold of slow request
	SlowAmountThreshold *uint32 `json:"SlowAmountThreshold,omitempty"`

	// SlowRatioThreshold specifies the ratio threshold of slow request
	SlowRatioThreshold *float32 `json:"SlowRatioThreshold,omitempty"`

	// ErrorAmountThreshold specifies the amount threshold of error request
	ErrorAmountThreshold *uint32 `json:"ErrorAmountThreshold,omitempty"`

	// ErrorRatioThreshold specifies the ratio threshold of error request
	ErrorRatioThreshold *float32 `json:"ErrorRatioThreshold,omitempty"`

	// DegradedStatusCode specifies the degraded http status code of circuit breaking
	DegradedStatusCode *int32 `json:"DegradedStatusCode,omitempty"`

	// DegradedResponseContent specifies the degraded http response content of circuit breaking
	DegradedResponseContent *string `json:"DegradedResponseContent,omitempty"`
}

// HTTPConnectionSettings defines the HTTP connection settings for an
// upstream host.
type HTTPConnectionSettings struct {
	// MaxRequests specifies the maximum number of parallel requests
	// allowed to the upstream host.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxRequests *uint32 `json:"MaxRequests,omitempty"`

	// MaxRequestsPerConnection specifies the maximum number of requests
	// per connection allowed to the upstream host.
	// Defaults to unlimited if not specified.
	// +optional
	MaxRequestsPerConnection *uint32 `json:"MaxRequestsPerConnection,omitempty"`

	// MaxPendingRequests specifies the maximum number of pending HTTP
	// requests allowed to the upstream host. For HTTP/2 connections,
	// if `maxRequestsPerConnection` is not configured, all requests will
	// be multiplexed over the same connection so this circuit breaker
	// will only be hit when no connection is already established.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxPendingRequests *uint32 `json:"MaxPendingRequests,omitempty"`

	// MaxRetries specifies the maximum number of parallel retries
	// allowed to the upstream host.
	// Defaults to 4294967295 (2^32 - 1) if not specified.
	// +optional
	MaxRetries *uint32 `json:"MaxRetries,omitempty"`

	// CircuitBreaking specifies the HTTP connection circuit breaking setting.
	CircuitBreaking *HTTPCircuitBreaking `json:"CircuitBreaking,omitempty"`
}

// PipyConf is a policy used by pipy sidecar
type PipyConf struct {
	Ts               *time.Time
	Version          *string
	Metrics          bool
	Spec             MeshConfigSpec
	Certificate      *Certificate
	Inbound          *InboundTrafficPolicy  `json:"Inbound"`
	Outbound         *OutboundTrafficPolicy `json:"Outbound"`
	Forward          *ForwardTrafficPolicy  `json:"Forward,omitempty"`
	AllowedEndpoints map[string]string      `json:"AllowedEndpoints"`
	Chains           map[string][]string    `json:"Chains,omitempty"`

	PluginSetV     string `json:"-"`
	pluginPolicies map[string]map[string]*map[string]*runtime.RawExtension
	hashNameSet    map[uint64]int
	dnsResolveDB   map[string][]string
}
