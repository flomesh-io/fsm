package v1alpha3

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MeshConfig is the type used to represent the mesh configuration.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=meshconfig,scope=Namespaced
type MeshConfig struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the MeshConfig specification.
	// +optional
	Spec MeshConfigSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// MeshConfigSpec is the spec for FSM's configuration.
type MeshConfigSpec struct {
	// ClusterSetSpec defines the configurations of cluster.
	ClusterSet ClusterSetSpec `json:"clusterSet,omitempty"`

	// Sidecar defines the configurations of the proxy sidecar in a mesh.
	Sidecar SidecarSpec `json:"sidecar,omitempty"`

	// RepoServer defines the configurations of pipy repo server.
	RepoServer RepoServerSpec `json:"repoServer,omitempty"`

	// Traffic defines the traffic management configurations for a mesh instance.
	Traffic TrafficSpec `json:"traffic,omitempty"`

	// Observalility defines the observability configurations for a mesh instance.
	Observability ObservabilitySpec `json:"observability,omitempty"`

	// Certificate defines the certificate management configurations for a mesh instance.
	Certificate CertificateSpec `json:"certificate,omitempty"`

	// FeatureFlags defines the feature flags for a mesh instance.
	FeatureFlags FeatureFlags `json:"featureFlags,omitempty"`

	// PluginChains defines the default plugin chains.
	PluginChains PluginChainsSpec `json:"pluginChains,omitempty"`

	// Ingress defines the configurations of Ingress features.
	Ingress IngressSpec `json:"ingress,omitempty"`

	// GatewayAPI defines the configurations of GatewayAPI features.
	GatewayAPI GatewayAPISpec `json:"gatewayAPI,omitempty"`

	// ServiceLB defines the configurations of ServiceLBServiceLB features.
	ServiceLB ServiceLBSpec `json:"serviceLB,omitempty"`

	// FLB defines the configurations of FLB features.
	FLB FLBSpec `json:"flb,omitempty"`

	// EgressGateway defines the configurations of EgressGateway features.
	EgressGateway EgressGatewaySpec `json:"egressGateway,omitempty"`

	// Image defines the configurations of Image info
	Image ImageSpec `json:"image"`

	// Misc defines the configurations of misc info
	Misc MiscSpec `json:"misc"`

	// Connector defines the configurations of connector info
	Connector ConnectorSpec `json:"connector"`
}

// LocalProxyMode is a type alias representing the way the sidecar proxies to the main application
// +kubebuilder:validation:Enum=Localhost;PodIP
type LocalProxyMode string

const (
	// LocalProxyModeLocalhost indicates the the sidecar should communicate with the main application over localhost
	LocalProxyModeLocalhost LocalProxyMode = "Localhost"
	// LocalProxyModePodIP indicates that the sidecar should communicate with the main application via the pod ip
	LocalProxyModePodIP LocalProxyMode = "PodIP"
)

// ResolveAddr is the type to represent FSM's Resolve Addr configuration.
type ResolveAddr struct {
	// IPv4 defines a ipv4 address for resolve DN.
	IPv4 string `json:"ipv4"`

	// IPv6 defines a ipv6 address for resolve DN.
	IPv6 string `json:"ipv6,omitempty"`
}

// WildcardDN is the type to represent FSM's Wildcard DN configuration.
type WildcardDN struct {
	// Enable defines a boolean indicating if wildcard are enabled for local DNS Proxy.
	Enable bool `json:"enable"`

	// LOs defines loopback addresses for resolve DN.
	LOs []*ResolveAddr `json:"los"`

	// IPs defines ip addresses for resolve DN.
	IPs []*ResolveAddr `json:"ips"`
}

// ResolveDN is the type to represent FSM's Resolve DN configuration.
type ResolveDN struct {
	// DN defines resolve DN.
	DN string `json:"dn"`

	// IPs defines ip addresses for resolve DN.
	IPs []*ResolveAddr `json:"ips"`
}

// LocalDNSProxy is the type to represent FSM's local DNS proxy configuration.
type LocalDNSProxy struct {
	// Enable defines a boolean indicating if the sidecars are enabled for local DNS Proxy.
	Enable bool `json:"enable"`

	// PrimaryUpstreamDNSServerIPAddr defines a primary upstream DNS server for local DNS Proxy.
	// +optional
	PrimaryUpstreamDNSServerIPAddr string `json:"primaryUpstreamDNSServerIPAddr,omitempty"`

	// SecondaryUpstreamDNSServerIPAddr defines a secondary upstream DNS server for local DNS Proxy.
	// +optional
	SecondaryUpstreamDNSServerIPAddr string `json:"secondaryUpstreamDNSServerIPAddr,omitempty"`

	// +kubebuilder:default=false
	// +optional
	GenerateIPv6BasedOnIPv4 bool `json:"generateIPv6BasedOnIPv4,omitempty"`

	// Wildcard defines Wildcard DN.
	Wildcard WildcardDN `json:"wildcard"`

	// DB defines Resolve DB.
	DB []ResolveDN `json:"db,omitempty"`
}

// SidecarSpec is the type used to represent the specifications for the proxy sidecar.
type SidecarSpec struct {
	// EnablePrivilegedInitContainer defines a boolean indicating whether the init container for a meshed pod should run as privileged.
	EnablePrivilegedInitContainer bool `json:"enablePrivilegedInitContainer"`

	// +kubebuilder:default=true
	// +optional
	CompressConfig bool `json:"compressConfig"`

	// LogLevel defines the logging level for the sidecar's logs. Non developers should generally never set this value. In production environments the LogLevel should be set to error.
	LogLevel string `json:"logLevel,omitempty"`

	// SidecarImage defines the container image used for the proxy sidecar.
	SidecarImage string `json:"sidecarImage,omitempty"`

	// SidecarDisabledMTLS defines whether mTLS is disabled.
	SidecarDisabledMTLS bool `json:"sidecarDisabledMTLS"`

	// MaxDataPlaneConnections defines the maximum allowed data plane connections from a proxy sidecar to the FSM controller.
	MaxDataPlaneConnections int `json:"maxDataPlaneConnections,omitempty"`

	// ConfigResyncInterval defines the resync interval for regular proxy broadcast updates.
	ConfigResyncInterval string `json:"configResyncInterval,omitempty"`

	// SidecarTimeout defines the connect/idle/read/write timeout.
	SidecarTimeout int `json:"sidecarTimeout,omitempty"`

	// Resources defines the compute resources for the sidecar.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// InitResources defines the compute resources for init container.
	InitResources corev1.ResourceRequirements `json:"initResources,omitempty"`

	// HealthcheckResources defines the compute resources for init container.
	HealthcheckResources corev1.ResourceRequirements `json:"healthcheckResources,omitempty"`

	// TLSMinProtocolVersion defines the minimum TLS protocol version that the sidecar supports. Valid TLS protocol versions are TLS_AUTO, TLSv1_0, TLSv1_1, TLSv1_2 and TLSv1_3.
	TLSMinProtocolVersion string `json:"tlsMinProtocolVersion,omitempty"`

	// TLSMaxProtocolVersion defines the maximum TLS protocol version that the sidecar supports. Valid TLS protocol versions are TLS_AUTO, TLSv1_0, TLSv1_1, TLSv1_2 and TLSv1_3.
	TLSMaxProtocolVersion string `json:"tlsMaxProtocolVersion,omitempty"`

	// CipherSuites defines a list of ciphers that listener supports when negotiating TLS 1.0-1.2. This setting has no effect when negotiating TLS 1.3. For valid cipher names, see the latest OpenSSL ciphers manual page. E.g. https://www.openssl.org/docs/man1.1.1/apps/ciphers.html.
	CipherSuites []string `json:"cipherSuites,omitempty"`

	// ECDHCurves defines a list of ECDH curves that TLS connection supports. If not specified, the curves are [X25519, P-256] for non-FIPS build and P-256 for builds using BoringSSL FIPS.
	ECDHCurves []string `json:"ecdhCurves,omitempty"`

	// LocalProxyMode defines the network interface the proxy will use to send traffic to the backend service application. Acceptable values are [`Localhost`, `PodIP`]. The default is `Localhost`
	LocalProxyMode LocalProxyMode `json:"localProxyMode,omitempty"`

	// LocalDNSProxy improves the performance of your computer by caching the responses coming from your DNS servers
	LocalDNSProxy LocalDNSProxy `json:"localDNSProxy,omitempty"`
}

// TrafficSpec is the type used to represent FSM's traffic management configuration.
type TrafficSpec struct {
	// InterceptionMode defines a string indicating which traffic interception mode is used.
	// +kubebuilder:validation:Enum=NodeLevel;PodLevel
	InterceptionMode string `json:"interceptionMode"`

	// EnableEgress defines a boolean indicating if mesh-wide Egress is enabled.
	EnableEgress bool `json:"enableEgress"`

	// OutboundIPRangeExclusionList defines a global list of IP address ranges to exclude from outbound traffic interception by the sidecar proxy.
	OutboundIPRangeExclusionList []string `json:"outboundIPRangeExclusionList"`

	// OutboundIPRangeInclusionList defines a global list of IP address ranges to include for outbound traffic interception by the sidecar proxy.
	// IP addresses outside this range will be excluded from outbound traffic interception by the sidecar proxy.
	OutboundIPRangeInclusionList []string `json:"outboundIPRangeInclusionList"`

	// OutboundPortExclusionList defines a global list of ports to exclude from outbound traffic interception by the sidecar proxy.
	OutboundPortExclusionList []int `json:"outboundPortExclusionList"`

	// InboundPortExclusionList defines a global list of ports to exclude from inbound traffic interception by the sidecar proxy.
	InboundPortExclusionList []int `json:"inboundPortExclusionList"`

	// EnablePermissiveTrafficPolicyMode defines a boolean indicating if permissive traffic policy mode is enabled mesh-wide.
	EnablePermissiveTrafficPolicyMode bool `json:"enablePermissiveTrafficPolicyMode"`

	// InboundExternalAuthorization defines a ruleset that, if enabled, will configure a remote external authorization endpoint
	// for all inbound and ingress traffic in the mesh.
	InboundExternalAuthorization ExternalAuthzSpec `json:"inboundExternalAuthorization,omitempty"`

	// NetworkInterfaceExclusionList defines a global list of network interface
	// names to exclude from inbound and outbound traffic interception by the
	// sidecar proxy.
	NetworkInterfaceExclusionList []string `json:"networkInterfaceExclusionList"`

	// HTTP1PerRequestLoadBalancing defines a boolean indicating if load balancing based on request is enabled for http1.
	HTTP1PerRequestLoadBalancing bool `json:"http1PerRequestLoadBalancing"`

	// HTTP1PerRequestLoadBalancing defines a boolean indicating if load balancing based on request is enabled for http2.
	HTTP2PerRequestLoadBalancing bool `json:"http2PerRequestLoadBalancing"`

	// ServiceAccessMode defines a string indicating service access mode.
	// +kubebuilder:default=mixed
	ServiceAccessMode ServiceAccessMode `json:"serviceAccessMode"`

	// +kubebuilder:default={mustWithServicePort: false, withTrustDomain: true}
	// +optional
	ServiceAccessNames *ServiceAccessNames `json:"serviceAccessNames,omitempty"`
}

// ServiceAccessMode is a type alias representing the mode service accessed.
// +kubebuilder:validation:Enum=ip;domain;mixed
type ServiceAccessMode string

const (
	//ServiceAccessModeIP defines the ip service access mode
	ServiceAccessModeIP ServiceAccessMode = "ip"

	//ServiceAccessModeDomain defines the domain service access mode
	ServiceAccessModeDomain ServiceAccessMode = "domain"

	//ServiceAccessModeMixed defines the mixed service access mode
	ServiceAccessModeMixed ServiceAccessMode = "mixed"
)

type CloudServiceAccessNames struct {
	// +kubebuilder:default=true
	// +optional
	WithNamespace bool `json:"withNamespace,omitempty"`
}

type ServiceAccessNames struct {
	// +kubebuilder:default=false
	// +optional
	MustWithServicePort bool `json:"mustWithServicePort,omitempty"`

	// +kubebuilder:default=true
	// +optional
	WithTrustDomain bool `json:"withTrustDomain,omitempty"`

	// +kubebuilder:default=true
	// +optional
	MustWithNamespace bool `json:"mustWithNamespace,omitempty"`

	// +kubebuilder:default={withNamespace: true}
	// +optional
	CloudServiceAccessNames *CloudServiceAccessNames `json:"cloud,omitempty"`
}

// ObservabilitySpec is the type to represent FSM's observability configurations.
type ObservabilitySpec struct {
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic;disabled
	// FSMLogLevel defines the log level for FSM control plane logs.
	FSMLogLevel string `json:"fsmLogLevel,omitempty"`

	// Tracing defines FSM's tracing configuration.
	Tracing TracingSpec `json:"tracing,omitempty"`

	// RemoteLogging defines FSM's remote logging configuration.
	RemoteLogging RemoteLoggingSpec `json:"remoteLogging,omitempty"`
}

// TracingSpec is the type to represent FSM's tracing configuration.
type TracingSpec struct {
	// Enable defines a boolean indicating if the sidecars are enabled for tracing.
	Enable bool `json:"enable"`

	// Port defines the tracing collector's port.
	Port int16 `json:"port,omitempty"`

	// Address defines the tracing collectio's hostname.
	Address string `json:"address,omitempty"`

	// Endpoint defines the API endpoint for tracing requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`

	// SampledFraction defines the sampled fraction.
	SampledFraction *string `json:"sampledFraction,omitempty"`
}

// RemoteLoggingSpec is the type to represent FSM's remote logging configuration.
type RemoteLoggingSpec struct {
	// Enable defines a boolean indicating if the sidecars are enabled for remote logging.
	Enable bool `json:"enable"`

	// Level defines the remote logging's level.
	Level uint16 `json:"level,omitempty"`

	// Port defines the remote logging's port.
	Port int16 `json:"port,omitempty"`

	// Address defines the remote logging's hostname.
	Address string `json:"address,omitempty"`

	// Endpoint defines the API endpoint for remote logging requests sent to the collector.
	Endpoint string `json:"endpoint,omitempty"`

	// Authorization defines the access entity that allows to authorize someone in remote logging service.
	Authorization string `json:"authorization,omitempty"`

	// SampledFraction defines the sampled fraction.
	SampledFraction *string `json:"sampledFraction,omitempty"`

	// SecretName defines the name of the secret that contains the configuration for remote logging.
	SecretName string `json:"secretName,omitempty"`
}

// ExternalAuthzSpec is a type to represent external authorization configuration.
type ExternalAuthzSpec struct {
	// Enable defines a boolean indicating if the external authorization policy is to be enabled.
	Enable bool `json:"enable"`

	// Address defines the remote address of the external authorization endpoint.
	Address string `json:"address,omitempty"`

	// Port defines the destination port of the remote external authorization endpoint.
	Port uint16 `json:"port,omitempty"`

	// StatPrefix defines a prefix for the stats sink for this external authorization policy.
	StatPrefix string `json:"statPrefix,omitempty"`

	// Timeout defines the timeout in which a response from the external authorization endpoint.
	// is expected to execute.
	Timeout string `json:"timeout,omitempty"`

	// FailureModeAllow defines a boolean indicating if traffic should be allowed on a failure to get a
	// response against the external authorization endpoint.
	FailureModeAllow bool `json:"failureModeAllow"`
}

// CertificateSpec is the type to reperesent FSM's certificate management configuration.
type CertificateSpec struct {
	// ServiceCertValidityDuration defines the service certificate validity duration.
	ServiceCertValidityDuration string `json:"serviceCertValidityDuration,omitempty"`

	// CertKeyBitSize defines the certicate key bit size.
	CertKeyBitSize int `json:"certKeyBitSize,omitempty"`

	// IngressGateway defines the certificate specification for an ingress gateway.
	// +optional
	IngressGateway *IngressGatewayCertSpec `json:"ingressGateway,omitempty"`
}

// IngressGatewayCertSpec is the type to represent the certificate specification for an ingress gateway.
type IngressGatewayCertSpec struct {
	// SubjectAltNames defines the Subject Alternative Names (domain names and IP addresses) secured by the certificate.
	SubjectAltNames []string `json:"subjectAltNames"`

	// ValidityDuration defines the validity duration of the certificate.
	ValidityDuration string `json:"validityDuration"`

	// Secret defines the secret in which the certificate is stored.
	Secret corev1.SecretReference `json:"secret"`
}

// MeshConfigList lists the MeshConfig objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MeshConfig `json:"items"`
}

// FeatureFlags is a type to represent FSM's feature flags.
type FeatureFlags struct {
	// EnableEgressPolicy defines if FSM's Egress policy is enabled.
	EnableEgressPolicy bool `json:"enableEgressPolicy"`

	// EnableSnapshotCacheMode defines if XDS server starts with snapshot cache.
	EnableSnapshotCacheMode bool `json:"enableSnapshotCacheMode"`

	//EnableAsyncProxyServiceMapping defines if FSM will map proxies to services asynchronously.
	EnableAsyncProxyServiceMapping bool `json:"enableAsyncProxyServiceMapping"`

	// EnableIngressBackendPolicy defines if FSM will use the IngressBackend API to allow ingress traffic to
	// service mesh backends.
	EnableIngressBackendPolicy bool `json:"enableIngressBackendPolicy"`

	// EnableAccessControlPolicy defines if FSM will use the AccessControl API to allow access control traffic to
	// service mesh backends.
	EnableAccessControlPolicy bool `json:"enableAccessControlPolicy"`

	// EnableAccessCertPolicy defines if FSM can issue certificates for external services..
	EnableAccessCertPolicy bool `json:"enableAccessCertPolicy"`

	// EnableSidecarPrettyConfig defines if pretty sidecar config is enabled.
	EnableSidecarPrettyConfig bool `json:"enableSidecarPrettyConfig"`

	// EnableSidecarActiveHealthChecks defines if FSM will Sidecar active health
	// checks between services allowed to communicate.
	EnableSidecarActiveHealthChecks bool `json:"enableSidecarActiveHealthChecks"`

	// EnableRetryPolicy defines if retry policy is enabled.
	EnableRetryPolicy bool `json:"enableRetryPolicy"`

	// EnablePluginPolicy defines if plugin policy is enabled.
	EnablePluginPolicy bool `json:"enablePluginPolicy"`

	// EnableAutoDefaultRoute defines if auto default route is enabled.
	EnableAutoDefaultRoute bool `json:"enableAutoDefaultRoute"`

	// EnableValidateGatewayListenerHostname defines if validate gateway listener hostname is enabled.
	EnableValidateGatewayListenerHostname bool `json:"enableValidateGatewayListenerHostname"`

	// EnableValidateHTTPRouteHostnames defines if validate http route hostnames is enabled.
	EnableValidateHTTPRouteHostnames bool `json:"enableValidateHTTPRouteHostnames"`

	// EnableValidateGRPCRouteHostnames defines if validate grpc route hostnames is enabled.
	EnableValidateGRPCRouteHostnames bool `json:"enableValidateGRPCRouteHostnames"`

	// EnableValidateTCPRouteHostnames defines if validate tcp route hostnames is enabled.
	EnableValidateTLSRouteHostnames bool `json:"enableValidateTLSRouteHostnames"`

	// UseEndpointSlicesForGateway defines if endpoint slices are enabled for calculating gateway routes.
	UseEndpointSlicesForGateway bool `json:"useEndpointSlicesForGateway"`

	// DropRouteRuleIfNoAvailableBackends defines if route rule should be dropped when no available endpoints.
	DropRouteRuleIfNoAvailableBackends bool `json:"dropRouteRuleIfNoAvailableBackends"`
}

// RepoServerSpec is the type to represent repo server.
type RepoServerSpec struct {
	// IPAddr of the pipy repo server
	IPAddr string `json:"ipaddr"`

	// Port defines the pipy repo server's port.
	Port int16 `json:"port,omitempty"`

	// Codebase is the folder used by fsmController
	Codebase string `json:"codebase"`
}

// ClusterPropertySpec is the type to represent cluster property.
type ClusterPropertySpec struct {
	// Name defines the name of cluster property.
	Name string `json:"name"`

	// Value defines the name of cluster property.
	Value string `json:"value"`
}

// ClusterSetSpec is the type to represent cluster set.
type ClusterSetSpec struct {
	// +kubebuilder:default=false
	// IsManaged defines if the cluster is managed.
	IsManaged bool `json:"isManaged"`

	// UID defines Unique ID of cluster.
	UID string `json:"uid"`

	// +kubebuilder:default=default
	// +optional
	// Region defines Region of cluster.
	Region string `json:"region"`

	// +kubebuilder:default=default
	// +optional
	// Zone defines Zone of cluster.
	Zone string `json:"zone"`

	// +kubebuilder:default=default
	// +optional
	// Group defines Group of cluster.
	Group string `json:"group"`

	// Name defines Name of cluster.
	Name string `json:"name"`

	// ControlPlaneUID defines the unique ID of the control plane cluster,
	//   in case it's managed
	ControlPlaneUID string `json:"controlPlaneUID"`

	// Properties defines properties for cluster.
	Properties []ClusterPropertySpec `json:"properties"`
}

// PluginChainsSpec is the type to represent plugin chains.
type PluginChainsSpec struct {
	// InboundTCPChains defines inbound tcp chains
	InboundTCPChains []*PluginChainSpec `json:"inbound-tcp"`

	// InboundHTTPChains defines inbound http chains
	InboundHTTPChains []*PluginChainSpec `json:"inbound-http"`

	// OutboundTCPChains defines outbound tcp chains
	OutboundTCPChains []*PluginChainSpec `json:"outbound-tcp"`

	// OutboundHTTPChains defines outbound http chains
	OutboundHTTPChains []*PluginChainSpec `json:"outbound-http"`
}

// PluginChainSpec is the type to represent plugin chain.
type PluginChainSpec struct {
	// Plugin defines the name of plugin
	Plugin string `json:"plugin"`

	// Priority defines the priority of plugin
	Priority float32 `json:"priority"`

	// Disable defines the visibility of plugin
	Disable bool `json:"disable"`
}

// IngressSpec is the type to represent ingress.
type IngressSpec struct {
	// +kubebuilder:default=true
	// Enabled defines if ingress is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=false
	// Namespaced defines if ingress is namespaced.
	Namespaced bool `json:"namespaced"`

	// +kubebuilder:default=LoadBalancer
	// +kubebuilder:validation:Enum=LoadBalancer;NodePort
	// Type defines the type of ingress service.
	Type corev1.ServiceType `json:"type"`

	// +kubebuilder:default=info
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic;disabled
	// LogLevel defines the log level of ingress.
	LogLevel string `json:"logLevel"`

	// +kubebuilder:default={enabled: true, bind: 80, listen: 8000, nodePort: 30508}
	// +optional
	// HTTP defines the http configuration of ingress.
	HTTP *HTTP `json:"http"`

	// +kubebuilder:default={enabled: true, bind: 443, listen: 8443, nodePort: 30607, mTLS: false}
	// +optional
	// TLS defines the tls configuration of ingress.
	TLS *TLS `json:"tls"`
}

// HTTP is the type to represent http.
type HTTP struct {
	// +kubebuilder:default=true
	// Enabled defines if http is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=80
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Bind defines the bind port of http.
	Bind int32 `json:"bind"`

	// +kubebuilder:default=8000
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Listen defines the listen port of http.
	Listen int32 `json:"listen"`

	// +kubebuilder:default=30508
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// NodePort defines the node port of http.
	NodePort int32 `json:"nodePort"`
}

// TLS is the type to represent tls.
type TLS struct {
	// +kubebuilder:default=false
	// Enabled defines if tls is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=443
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Bind defines the bind port of tls.
	Bind int32 `json:"bind" validate:"gte=1,lte=65535"`

	// +kubebuilder:default=8443
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Listen defines the listen port of tls.
	Listen int32 `json:"listen" validate:"gte=1,lte=65535"`

	// +kubebuilder:default=30607
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// NodePort defines the node port of tls.
	NodePort int32 `json:"nodePort" validate:"gte=0,lte=65535"`

	// +kubebuilder:default=false
	// MTLS defines if mTLS is enabled.
	MTLS bool `json:"mTLS"`

	// +kubebuilder:default={enabled: false, upstreamPort: 443}
	// +optional
	// SSLPassthrough defines the ssl passthrough configuration of tls.
	SSLPassthrough *SSLPassthrough `json:"sslPassthrough"`
}

// SSLPassthrough is the type to represent ssl passthrough.
type SSLPassthrough struct {
	// +kubebuilder:default=false
	// Enabled defines if ssl passthrough is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=443
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// UpstreamPort defines the upstream port of ssl passthrough.
	UpstreamPort int32 `json:"upstreamPort"`
}

// GatewayAPISpec is the type to represent gateway api.
type GatewayAPISpec struct {
	// +kubebuilder:default=false
	// Enabled defines if gateway api is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=info
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic;disabled
	// LogLevel defines the log level of gateway api.
	LogLevel string `json:"logLevel"`
}

// ServiceLBSpec is the type to represent service lb.
type ServiceLBSpec struct {
	// +kubebuilder:default=false
	// Enabled defines if service lb is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default="flomesh/mirrored-klipper-lb:v0.4.7"
	// Image defines the service lb image.
	Image string `json:"image"`
}

type FLBUpstreamMode string

const (
	FLBUpstreamModeNodePort FLBUpstreamMode = "NodePort"
	FLBUpstreamModeEndpoint FLBUpstreamMode = "Endpoint"
)

// FLBSpec is the type to represent flb.
type FLBSpec struct {
	// +kubebuilder:default=false
	// Enabled defines if flb is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=false
	// StrictMode defines if flb is in strict mode.
	StrictMode bool `json:"strictMode"`

	// +kubebuilder:default=Endpoint
	// +kubebuilder:validation:Enum=NodePort;Endpoint
	// UpstreamMode defines the upstream mode of flb.
	UpstreamMode FLBUpstreamMode `json:"upstreamMode"`

	// +kubebuilder:default=fsm-flb-secret
	// SecretName defines the secret name of flb.
	SecretName string `json:"secretName"`
}

// EgressGatewaySpec is the type to represent egress gateway.
type EgressGatewaySpec struct {
	// +kubebuilder:default=false
	// Enabled defines if flb is enabled.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=info
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic;disabled
	// LogLevel defines the log level of gateway api.
	LogLevel string `json:"logLevel"`

	// +kubebuilder:default=http2tunnel
	// +kubebuilder:validation:Enum=http2tunnel;sock5
	// Mode defines the mode of egress gateway.
	Mode string `json:"mode"`

	// +kubebuilder:default=1080
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Port defines the port of egress gateway.
	Port *int32 `json:"port,omitempty"`

	// +kubebuilder:default=6060
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// AdminPort defines the admin port of egress gateway.
	AdminPort *int32 `json:"adminPort,omitempty"`

	// +kubebuilder:default=1
	// Replicas defines the replicas of egress gateway.
	Replicas *int32 `json:"replicas,omitempty"`
}

// ImageSpec is the type to represent image.
type ImageSpec struct {
	// +kubebuilder:default=flomesh
	// Registry defines the registry of docker image.
	Registry string `json:"registry"`

	// +kubebuilder:default=latest
	// Tag defines the tag of docker image.
	Tag string `json:"tag"`

	// +kubebuilder:default=IfNotPresent
	// PullPolicy defines the pull policy of docker image.
	PullPolicy corev1.PullPolicy `json:"pullPolicy"`
}

// MiscSpec is the type to represent misc configs.
type MiscSpec struct {
	// +kubebuilder:default="flomesh/pipy-repo:1.5.5"
	// RepoServerImage defines the image of repo server.
	RepoServerImage string `json:"repoServerImage"`
}

// ConnectorGatewaySpec is the type to represent connector gateway configs.
type ConnectorGatewaySpec struct {
	ClusterIP  string `json:"clusterIP"`
	ExternalIP string `json:"externalIP"`

	IngressAddr     string `json:"ingressAddr"`
	IngressHTTPPort uint   `json:"ingressHTTPPort"`
	IngressGRPCPort uint   `json:"ingressGRPCPort"`

	EgressAddr     string `json:"egressAddr"`
	EgressHTTPPort uint   `json:"egressHTTPPort"`
	EgressGRPCPort uint   `json:"egressGRPCPort"`
}

// LoadBalancerType defines the type of load balancer
type LoadBalancerType string

const (
	// ActiveActiveLbType is the type of load balancer that distributes traffic to all targets
	ActiveActiveLbType LoadBalancerType = "ActiveActive"

	// FailOverLbType is the type of load balancer that distributes traffic to the first available target
	FailOverLbType LoadBalancerType = "FailOver"
)

type LoadBalancer struct {
	// +kubebuilder:default=ActiveActive
	// +kubebuilder:validation:Enum=ActiveActive;FailOver
	// Type of global load distribution
	Type LoadBalancerType `json:"type,omitempty"`

	MasterNamespace string   `json:"masterNamespace,omitempty"`
	SlaveNamespaces []string `json:"slaveNamespaces,omitempty"`
}

func (lb *LoadBalancer) IsMasterNamespace(namespace string) bool {
	if FailOverLbType == lb.Type && len(lb.SlaveNamespaces) > 0 {
		return strings.EqualFold(lb.MasterNamespace, namespace)
	}
	return false
}

func (lb *LoadBalancer) IsSlaveNamespace(namespace string) bool {
	if FailOverLbType == lb.Type && len(lb.SlaveNamespaces) > 0 {
		for _, ns := range lb.SlaveNamespaces {
			if strings.EqualFold(ns, namespace) {
				return true
			}
		}
	}
	return false
}

// ConnectorSpec is the type to represent connector configs.
type ConnectorSpec struct {
	Lb LoadBalancer `json:"lb"`

	// +kubebuilder:default="viaGateway Managed by fsm-connector-gateway."
	Notice string `json:"DO_NOT_EDIT_viaGateway"`

	// ViaGateway defines gateway settings
	ViaGateway ConnectorGatewaySpec `json:"viaGateway"`
}
