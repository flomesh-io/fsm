// Package constants defines the constants that are used by multiple other packages within FSM.
package constants

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"helm.sh/helm/v3/pkg/chartutil"
)

const (
	// WildcardIPAddr is a string constant.
	WildcardIPAddr = "0.0.0.0"

	// SidecarAdminPort is Sidecar's admin port
	SidecarAdminPort = 15000

	// SidecarAdminPortName is Sidecar's admin port name
	SidecarAdminPortName = "proxy-admin"

	// SidecarInboundListenerPort is Sidecar's inbound listener port number.
	SidecarInboundListenerPort = 15003

	// SidecarInboundListenerPortName is Sidecar's inbound listener port name.
	SidecarInboundListenerPortName = "proxy-inbound"

	// SidecarInboundPrometheusListenerPortName is Sidecar's inbound listener port name for prometheus.
	SidecarInboundPrometheusListenerPortName = "proxy-metrics"

	// SidecarOutboundListenerPort is Sidecar's outbound listener port number.
	SidecarOutboundListenerPort = 15001

	// SidecarOutboundListenerPortName is Sidecar's outbound listener port name.
	SidecarOutboundListenerPortName = "proxy-outbound"

	// SidecarUID is the Sidecar's User ID
	SidecarUID int64 = 1500

	// LocalhostIPAddress is the local host address.
	LocalhostIPAddress = "127.0.0.1"

	// SidecarMetricsCluster is the cluster name of the Prometheus metrics cluster
	SidecarMetricsCluster = "sidecar-metrics-cluster"

	// SidecarTracingCluster is the default name to refer to the tracing cluster.
	SidecarTracingCluster = "sidecar-tracing-cluster"

	// DefaultTracingEndpoint is the default endpoint route.
	DefaultTracingEndpoint = "/api/v2/spans"

	// DefaultTracingHost is the default tracing server name.
	DefaultTracingHost = "jaeger"

	// DefaultTracingPort is the tracing listener port.
	DefaultTracingPort = uint32(9411)

	// DefaultSidecarLogLevel is the default sidecar log level if not defined in the fsm MeshConfig
	DefaultSidecarLogLevel = "error"

	// DefaultFSMLogLevel is the default FSM log level if none is specified
	DefaultFSMLogLevel = "info"

	// SidecarPrometheusInboundListenerPort is Sidecar's inbound listener port number for prometheus
	SidecarPrometheusInboundListenerPort = 15010

	// InjectorWebhookPort is the port on which the sidecar injection webhook listens
	InjectorWebhookPort = 9090

	// FSMHTTPServerPort is the port on which fsm-controller and fsm-injector serve HTTP requests for metrics, health probes etc.
	FSMHTTPServerPort = 9091

	// DebugPort is the port on which FSM exposes its debug server
	DebugPort = 9092

	// ValidatorWebhookPort is the port on which the resource validator webhook listens
	ValidatorWebhookPort = 9093

	// FSMWebhookPort is the port mutating and validating webhook listens
	FSMWebhookPort = 9443

	// FSMControllerLeaderElectionID is the name of the resource that leader election
	// 	will use for holding the leader lock.
	FSMControllerLeaderElectionID = "fsm-controller.flomesh.io"

	// FSMControllerName is the name of the FSM Controller (formerly ADS service).
	FSMControllerName = "fsm-controller"

	// FSMInjectorName is the name of the FSM Injector.
	FSMInjectorName = "fsm-injector"

	// FSMBootstrapName is the name of the FSM Bootstrap.
	FSMBootstrapName = "fsm-bootstrap"

	// FSMIngressName is the name of the FSM Ingress.
	FSMIngressName = "fsm-ingress"

	// FSMGatewayName is the name of the FSM Gateway.
	FSMGatewayName = "fsm-gateway"

	// FSMEgressGatewayName is the name of the FSM Egress Gateway.
	FSMEgressGatewayName = "fsm-egress-gateway"

	// FSMServiceLBName is the name of the FSM ServiceLB.
	FSMServiceLBName = "fsm-servicelb"

	// ProxyServerPort is the port on which the Pipy Repo Service (ADS) listens for new connections from sidecar proxies
	ProxyServerPort = 6060

	// PrometheusScrapePath is the path for prometheus to scrap sidecar metrics from
	PrometheusScrapePath = "/stats/prometheus"

	// CertificationAuthorityCommonName is the CN used for the root certificate for FSM.
	CertificationAuthorityCommonName = "fsm-ca.flomesh.io"

	// CertificationAuthorityRootValidityPeriod is when the root certificate expires
	CertificationAuthorityRootValidityPeriod = 87600 * time.Hour // a decade

	// FSMCertificateValidityPeriod is the TTL of the certificates.
	FSMCertificateValidityPeriod = 87600 * time.Hour // a decade

	// DefaultCABundleSecretName is the default name of the secret for the FSM CA bundle
	DefaultCABundleSecretName = "fsm-ca-bundle" // #nosec G101: Potential hardcoded credentials

	// RegexMatchAll is a regex pattern match for all
	RegexMatchAll = ".*"

	// WildcardHTTPMethod is a wildcard for all HTTP methods
	WildcardHTTPMethod = "*"

	// FSMKubeResourceMonitorAnnotation is the key of the annotation used to monitor a K8s resource
	FSMKubeResourceMonitorAnnotation = "flomesh.io/monitored-by"

	// KubernetesOpaqueSecretCAKey is the key which holds the CA bundle in a Kubernetes secret.
	KubernetesOpaqueSecretCAKey = "ca.crt"

	// KubernetesOpaqueSecretRootPrivateKeyKey is the key which holds the CA's private key in a Kubernetes secret.
	KubernetesOpaqueSecretRootPrivateKeyKey = "private.key"

	// SidecarUniqueIDLabelName is the label applied to pods with the unique ID of the sidecar.
	SidecarUniqueIDLabelName = "fsm-proxy-uuid"

	// ----- Environment Variables

	// EnvVarLogKubernetesEvents is the name of the env var instructing the event handlers whether to log at all (true/false)
	EnvVarLogKubernetesEvents = "FSM_LOG_KUBERNETES_EVENTS"

	// EnvVarHumanReadableLogMessages is an environment variable, which when set to "true" enables colorful human-readable log messages.
	EnvVarHumanReadableLogMessages = "FSM_HUMAN_DEBUG_LOG"

	// ClusterWeightAcceptAll is the weight for a cluster that accepts 100 percent of traffic sent to it
	ClusterWeightAcceptAll = 100

	// ClusterWeightFailOver is the weight for a cluster that accepts 0 percent of traffic sent to it
	ClusterWeightFailOver = 0

	// PrometheusDefaultRetentionTime is the default days for which data is retained in prometheus
	PrometheusDefaultRetentionTime = "15d"

	// DomainDelimiter is a delimiter used in representing domains
	DomainDelimiter = "."

	// SidecarContainerName is the name used to identify the sidecar container added on mesh-enabled deployments
	SidecarContainerName = "sidecar"

	// InitContainerName is the name of the init container
	InitContainerName = "fsm-init"
)

// HealthProbe constants
const (
	// LivenessProbePort is the port to use for liveness probe
	LivenessProbePort = int32(15901)

	// ReadinessProbePort is the port to use for readiness probe
	ReadinessProbePort = int32(15902)

	// StartupProbePort is the port to use for startup probe
	StartupProbePort = int32(15903)

	// HealthcheckPort is the port to use for healthcheck probe
	HealthcheckPort = int32(15904)

	// LivenessProbePath is the path to use for liveness probe
	LivenessProbePath = "/fsm-liveness-probe"

	// ReadinessProbePath is the path to use for readiness probe
	ReadinessProbePath = "/fsm-readiness-probe"

	// StartupProbePath is the path to use for startup probe
	StartupProbePath = "/fsm-startup-probe"

	// HealthcheckPath is the path to use for healthcheck probe
	HealthcheckPath = "/fsm-healthcheck"
)

// Annotations used by the control plane
const (
	// SidecarInjectionAnnotation is the annotation used for sidecar injection
	SidecarInjectionAnnotation = "flomesh.io/sidecar-injection"

	// MetricsAnnotation is the annotation used for enabling/disabling metrics
	MetricsAnnotation = "flomesh.io/metrics"
)

// Annotations and labels used by the MeshRootCertificate
const (
	// MRCStateValidatingRollout is the validating rollout status option for the State of the MeshRootCertificate
	MRCStateValidatingRollout = "validatingRollout"

	// MRCStateIssuingRollout is the issuing rollout status option for the State of the MeshRootCertificate
	MRCStateIssuingRollout = "issuingRollout"

	// MRCStateActive is the active status option for the State of the MeshRootCertificate
	MRCStateActive = "active"

	// MRCStateIssuingRollback is the issuing rollback status option for the State of the MeshRootCertificate
	MRCStateIssuingRollback = "issuingRollback"

	// MRCStateValidatingRollback is the validating rollback status option for the State of the MeshRootCertificate
	MRCStateValidatingRollback = "validatingRollback"

	// MRCStateInactive is the inactive status option for the State of the MeshRootCertificate
	MRCStateInactive = "inactive"

	// MRCStateError is the error status option for the State of the MeshRootCertificate
	MRCStateError = "error"
)

// Labels used by the control plane
const (
	// IgnoreLabel is the label used to ignore a resource
	IgnoreLabel = "flomesh.io/ignore"

	// ReconcileLabel is the label used to reconcile a resource
	ReconcileLabel = "flomesh.io/reconcile"

	// AppLabel is the label used to identify the app
	AppLabel = "app"
)

// Annotations used for Metrics
const (
	// PrometheusScrapeAnnotation is the annotation used to configure prometheus scraping
	PrometheusScrapeAnnotation = "prometheus.io/scrape"

	// PrometheusPortAnnotation is the annotation used to configure the port to scrape on
	PrometheusPortAnnotation = "prometheus.io/port"

	// PrometheusPathAnnotation is the annotation used to configure the path to scrape on
	PrometheusPathAnnotation = "prometheus.io/path"
)

// Annotations used for Egress Gateway
const (
	// EgressGatewayModeAnnotation is the key of the annotation used to indicate the mode of egress gateway
	EgressGatewayModeAnnotation = "flomesh.io/egress-gateway-mode"
)

// Egress Gateway Mode
const (
	// http2tunnel
	EgressGatewayModeHTTP2Tunnel = "http2tunnel"

	// sock5
	EgressGatewayModeSock5 = "sock5"
)

// Annotations used for sidecar
const (
	// SidecarResourceLimitsAnnotationPrefix is the key of the annotation used to indicate sidecar resource limits annotation prefix
	SidecarResourceLimitsAnnotationPrefix = "flomesh.io/sidecar-resource-limits"

	// SidecarResourceRequestsAnnotationPrefix is the key of the annotation used to indicate sidecar resource requests annotation prefix
	SidecarResourceRequestsAnnotationPrefix = "flomesh.io/sidecar-resource-requests"
)

// App labels as defined in the "fsm.labels" template in _helpers.tpl of the Helm chart.
const (
	FSMAppNameLabelKey     = "app.kubernetes.io/name"
	FSMAppNameLabelValue   = "flomesh.io"
	FSMAppInstanceLabelKey = "app.kubernetes.io/instance"
	FSMAppVersionLabelKey  = "app.kubernetes.io/version"
)

// Application protocols
const (
	// HTTP protocol
	ProtocolHTTP = "http"

	// HTTPS protocol
	ProtocolHTTPS = "https"

	// TCP protocol
	ProtocolTCP = "tcp"

	// gRPC protocol
	ProtocolGRPC = "grpc"

	// ProtocolTCPServerFirst implies TCP based server first protocols
	// Ex. MySQL, SMTP, PostgreSQL etc. where the server initiates the first
	// byte in a TCP connection.
	ProtocolTCPServerFirst = "tcp-server-first"
)

// Operating systems.
const (
	// OSLinux is the name for Linux operating system.
	OSLinux string = "linux"
)

// Logging contexts
const (
	// LogFieldContext is the key used to specify the logging context
	LogFieldContext = "context"
)

// Control plane HTTP server paths
const (
	// FSMControllerReadinessPath is the path at which FSM controller serves readiness probes
	FSMControllerReadinessPath = "/health/ready"

	// FSMControllerLivenessPath is the path at which FSM controller serves liveness probes
	FSMControllerLivenessPath = "/health/alive"

	// FSMControllerSMIVersionPath is the path at which FSM controller servers SMI version info
	FSMControllerSMIVersionPath = "/smi/version"

	// MetricsPath is the path at which FSM controller serves metrics
	MetricsPath = "/metrics"

	// VersionPath is the path at which FSM controller serves version info
	VersionPath = "/version"

	// WebhookHealthPath is the path at which the webooks serve health probes
	WebhookHealthPath = "/healthz"
)

// FSM HTTP Server Responses
const (
	// ServiceReadyResponse is the response returned by the server to indicate it is ready
	ServiceReadyResponse = "Service is ready"

	// ServiceAliveResponse is the response returned by the server to indicate it is alive
	ServiceAliveResponse = "Service is alive"
)

var (
	// SupportedProtocolsInMesh is a list of the protocols FSM supports for in-mesh traffic
	SupportedProtocolsInMesh = []string{ProtocolTCPServerFirst, ProtocolHTTP, ProtocolTCP, ProtocolGRPC}
)

const (
	// SidecarClassPipy is the SidecarClass field value for context field.
	SidecarClassPipy = "pipy"
)

const (
	//TrafficInterceptionModeIptables defines the iptables traffic interception mode
	TrafficInterceptionModeIptables = "iptables"

	//TrafficInterceptionModeEBPF defines the ebpf traffic interception mode
	TrafficInterceptionModeEBPF = "ebpf"

	//TrafficInterceptionModeNone defines the none traffic interception mode
	TrafficInterceptionModeNone = "none"
)

const (
	//ServiceAccessModeIP defines the ip service access mode
	ServiceAccessModeIP = "ip"

	//ServiceAccessModeDomain defines the domain service access mode
	ServiceAccessModeDomain = "domain"

	//ServiceAccessModeMixed defines the mixed service access mode
	ServiceAccessModeMixed = "mixed"
)

// GatewayAPI Group and Kinds
const (
	// GatewayAPIGroup is the group name used in Gateway API
	GatewayAPIGroup = "gateway.networking.k8s.io"

	// GatewayKind is the kind name of Gateway used in Gateway API
	GatewayKind = "Gateway"

	// HTTPRouteKind is the kind name of HTTPRoute used in Gateway API
	HTTPRouteKind = "HTTPRoute"

	// GRPCRouteKind is the kind name of GRPCRoute used in Gateway API
	GRPCRouteKind = "GRPCRoute"

	// TLSRouteKind is the kind name of TLSRoute used in Gateway API
	TLSRouteKind = "TLSRoute"

	// TCPRouteKind is the kind name of TCPRoute used in Gateway API
	TCPRouteKind = "TCPRoute"

	// UDPRouteKind is the kind name of UDPRoute used in Gateway API
	UDPRouteKind = "UDPRoute"
)

// Gateway API constants
const (
	// FSMGatewayClassName is the name of FSM GatewayClass
	FSMGatewayClassName = "fsm-gateway-cls"

	// GatewayController is the name of the FSM gateway controller
	GatewayController = "flomesh.io/gateway-controller"

	// GatewayPrefix is the prefix for all gateway annotations
	GatewayPrefix = "gateway.flomesh.io"

	// GatewayMTLSAnnotation is the annotation used to indicate whether the gateway should be configured for mTLS
	GatewayMTLSAnnotation = GatewayPrefix + "/mtls"

	// GatewayMutatingWebhookPath is the path at which the gateway mutating webhook is served
	GatewayMutatingWebhookPath = "/mutate-gateway-networking-k8s-io-v1beta1-gateway"

	// GatewayValidatingWebhookPath is the path at which the gateway validating webhook is served
	GatewayValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1beta1-gateway"

	// GatewayClassMutatingWebhookPath is the path at which the gateway class mutating webhook is served
	GatewayClassMutatingWebhookPath = "/mutate-gateway-networking-k8s-io-v1beta1-gatewayclass"

	// GatewayClassValidatingWebhookPath is the path at which the gateway class validating webhook is served
	GatewayClassValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1beta1-gatewayclass"

	// HTTPRouteMutatingWebhookPath is the path at which the HTTP route mutating webhook is served
	HTTPRouteMutatingWebhookPath = "/mutate-gateway-networking-k8s-io-v1beta1-httproute"

	// HTTPRouteValidatingWebhookPath is the path at which the HTTP route validating webhook is served
	HTTPRouteValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1beta1-httproute"

	// GRPCRouteMutatingWebhookPath is the path at which the gRPC route mutating webhook is served
	GRPCRouteMutatingWebhookPath = "/mutate-gateway-networking-k8s-io-v1alpha2-grpcroute"

	// GRPCRouteValidatingWebhookPath is the path at which the gRPC route validating webhook is served
	GRPCRouteValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1alpha2-grpcroute"

	// TCPRouteMutatingWebhookPath is the path at which the TCP route mutating webhook is served
	TCPRouteMutatingWebhookPath = "/mutate-gateway-networking-k8s-io-v1alpha2-tcproute"

	// TCPRouteValidatingWebhookPath is the path at which the TCP route validating webhook is served
	TCPRouteValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1alpha2-tcproute"

	// TLSRouteMutatingWebhookPath is the path at which the TLS route mutating webhook is served
	TLSRouteMutatingWebhookPath = "/mutate-gateway-networking-k8s-io-v1alpha2-tlsroute"

	// TLSRouteValidatingWebhookPath is the path at which the TLS route validating webhook is served
	TLSRouteValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1alpha2-tlsroute"
)

// PolicyAttachment constants
const (
	// RateLimitPolicyMutatingWebhookPath is the path at which the RateLimitPolicy mutating webhook is served
	RateLimitPolicyMutatingWebhookPath = "/mutate-gateway-flomesh-io-v1alpha1-ratelimitpolicy"

	// RateLimitPolicyValidatingWebhookPath is the path at which the RateLimitPolicy validating webhook is served
	RateLimitPolicyValidatingWebhookPath = "/validate-gateway-flomesh-io-v1alpha1-ratelimitpolicy"
)

// PIPY Repo constants
const (
	// DefaultPipyRepoPath is the default path for the PIPY repo
	DefaultPipyRepoPath = "/repo"

	// DefaultPipyRepoAPIPath is the default path for the PIPY repo API
	DefaultPipyRepoAPIPath = "/api/v1/repo"

	// DefaultPipyFileAPIPath is the default path for the PIPY file API
	DefaultPipyFileAPIPath = "/api/v1/repo-files"

	// DefaultServiceBasePath is the default path for the service codebase
	DefaultServiceBasePath = "/base/services"

	// DefaultIngressBasePath is the default path for the ingress codebase
	DefaultIngressBasePath = "/base/ingress"

	// DefaultGatewayBasePath is the default path for the gateway codebase
	DefaultGatewayBasePath = "/base/gateways"
)

// MultiCluster constants
const (
	// MultiClustersPrefix is the prefix for all multi-cluster annotations
	MultiClustersPrefix = "multicluster.flomesh.io"

	// MultiClustersServiceExportHash is the annotation used to indicate the hash of the exported service
	MultiClustersServiceExportHash = MultiClustersPrefix + "/export-hash"

	// MultiClusterLabelServiceName is used to indicate the name of multi-cluster service
	// that an EndpointSlice belongs to.
	MultiClusterLabelServiceName = MultiClustersPrefix + "/service-name"

	// MultiClusterLabelSourceCluster is used to indicate the name of the cluster in which an exported resource exists.
	MultiClusterLabelSourceCluster = MultiClustersPrefix + "/source-cluster"

	// MultiClusterDerivedServiceAnnotation is set on a ServiceImport to reference the
	// derived Service that represents the imported service for kube-proxy.
	MultiClusterDerivedServiceAnnotation = MultiClustersPrefix + "/derived-service"

	// ClusterTpl is the template for cluster name
	ClusterTpl = "{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}"

	// ClusterMutatingWebhookPath is the path at which the cluster mutating webhook is served
	ClusterMutatingWebhookPath = "/mutate-flomesh-io-v1alpha1-cluster"

	// ClusterValidatingWebhookPath is the path at which the cluster validating webhook is served
	ClusterValidatingWebhookPath = "/validate-flomesh-io-v1alpha1-cluster"

	// ServiceImportMutatingWebhookPath is the path at which the service import mutating webhook is served
	ServiceImportMutatingWebhookPath = "/mutate-flomesh-io-v1alpha1-serviceimport"

	// ServiceImportValidatingWebhookPath is the path at which the service import validating webhook is served
	ServiceImportValidatingWebhookPath = "/validate-flomesh-io-v1alpha1-serviceimport"

	// ServiceExportMutatingWebhookPath is the path at which the service export mutating webhook is served
	ServiceExportMutatingWebhookPath = "/mutate-flomesh-io-v1alpha1-serviceexport"

	// ServiceExportValidatingWebhookPath is the path at which the service export validating webhook is served
	ServiceExportValidatingWebhookPath = "/validate-flomesh-io-v1alpha1-serviceexport"

	// GlobalTrafficPolicyMutatingWebhookPath is the path at which the global traffic policy mutating webhook is served
	GlobalTrafficPolicyMutatingWebhookPath = "/mutate-flomesh-io-v1alpha1-globaltrafficpolicy"

	// GlobalTrafficPolicyValidatingWebhookPath is the path at which the global traffic policy validating webhook is served
	GlobalTrafficPolicyValidatingWebhookPath = "/validate-flomesh-io-v1alpha1-globaltrafficpolicy"
)

// FLB constants
const (
	// FLBPrefix is the prefix for all flb annotations
	FLBPrefix = "flb.flomesh.io"

	// FLBEnabledAnnotation is the annotation used to indicate if the flb is enabled
	FLBEnabledAnnotation = FLBPrefix + "/enabled"

	// FLBAddressPoolAnnotation is the annotation used to indicate the address pool
	FLBAddressPoolAnnotation = FLBPrefix + "/address-pool"

	// FLBDesiredIPAnnotation is the annotation used to indicate the desired ip
	FLBDesiredIPAnnotation = FLBPrefix + "/desired-ip"

	// FLBMaxConnectionsAnnotation is the annotation used to indicate the max connections
	FLBMaxConnectionsAnnotation = FLBPrefix + "/max-connections"

	// FLBReadTimeoutAnnotation is the annotation used to indicate the read timeout
	FLBReadTimeoutAnnotation = FLBPrefix + "/read-timeout"

	// FLBWriteTimeoutAnnotation is the annotation used to indicate the write timeout
	FLBWriteTimeoutAnnotation = FLBPrefix + "/write-timeout"

	// FLBIdleTimeoutAnnotation is the annotation used to indicate the idle timeout
	FLBIdleTimeoutAnnotation = FLBPrefix + "/idle-timeout"

	// FLBAlgoAnnotation is the annotation used to indicate the algo
	FLBAlgoAnnotation = FLBPrefix + "/algo"

	// FLBTagsAnnotation is the annotation used to indicate the tags
	FLBTagsAnnotation = FLBPrefix + "/tags"

	// FLBSecretLabel is the label used to indicate the secret
	FLBSecretLabel = FLBPrefix + "/config"

	// FLBSecretKeyBaseURL is the key for the base url
	FLBSecretKeyBaseURL = "baseUrl"

	// FLBSecretKeyUsername is the key for the username
	FLBSecretKeyUsername = "username"

	// FLBSecretKeyPassword is the key for the password
	FLBSecretKeyPassword = "password"

	// FLBSecretKeyK8sCluster is the key for the k8s cluster which FLB controller is installed
	FLBSecretKeyK8sCluster = "k8sCluster"

	// FLBSecretKeyDefaultAddressPool is the key for the default address pool
	FLBSecretKeyDefaultAddressPool = "defaultAddressPool"

	// FLBSecretKeyDefaultAlgo is the key for the default algo
	FLBSecretKeyDefaultAlgo = "defaultAlgo"

	// FLBServiceMutatingWebhookPath is the path at which the flb service mutating webhook is served
	FLBServiceMutatingWebhookPath = "/mutate-flb-core-v1-service"

	// FLBServiceValidatingWebhookPath is the path at which the flb service validating webhook is served
	FLBServiceValidatingWebhookPath = "/validate-flb-core-v1-service"

	// FLBSecretMutatingWebhookPath is the path at which the flb secret mutating webhook is served
	FLBSecretMutatingWebhookPath = "/mutate-flb-core-v1-secret"

	// FLBSecretValidatingWebhookPath is the path at which the flb secret validating webhook is served
	FLBSecretValidatingWebhookPath = "/validate-flb-core-v1-secret"
)

// MultiCluster variables
var (
	// ClusterIDTemplate is a template for cluster ID
	ClusterIDTemplate = template.Must(template.New("ClusterIDTemplate").Parse(ClusterTpl))
)

// Webhook constants
const (
	// KubernetesEndpointSliceServiceNameLabel is the label used to indicate the name of the service
	KubernetesEndpointSliceServiceNameLabel = "kubernetes.io/service-name"

	// RootCACertName is the name of the root CA cert
	RootCACertName = "ca.crt"

	// TLSCertName is the name of the TLS cert
	TLSCertName = "tls.crt"

	// TLSPrivateKeyName is the name of the TLS private key
	TLSPrivateKeyName = "tls.key"

	// WebhookServerServingCertsPathTpl is the template for webhook server serving certs path
	WebhookServerServingCertsPathTpl = "%s/k8s-webhook-server/serving-certs"

	// DefaultMutatingWebhookConfigurationName is the name of the default mutating webhook configuration
	DefaultMutatingWebhookConfigurationName = "flomesh-mutating-webhook-configuration"

	// DefaultValidatingWebhookConfigurationName is the name of the default validating webhook configuration
	DefaultValidatingWebhookConfigurationName = "flomesh-validating-webhook-configuration"
)

// Webhook variables
var (
	// WebhookServerServingCertsPath is the path at which the webhook server serving certs are stored
	WebhookServerServingCertsPath = fmt.Sprintf(WebhookServerServingCertsPathTpl, os.TempDir())
)

// NamespacedIngress constants
const (
	// NamespacedIngressMutatingWebhookPath is the path at which the namespaced ingress mutating webhook is served
	NamespacedIngressMutatingWebhookPath = "/mutate-flomesh-io-v1alpha1-namespacedingress"

	// NamespacedIngressValidatingWebhookPath is the path at which the namespaced ingress validating webhook is served
	NamespacedIngressValidatingWebhookPath = "/validate-flomesh-io-v1alpha1-namespacedingress"

	// IngressMutatingWebhookPath is the path at which the ingress mutating webhook is served
	IngressMutatingWebhookPath = "/mutate-networking-v1-ingress"

	// IngressValidatingWebhookPath is the path at which the ingress validating webhook is served
	IngressValidatingWebhookPath = "/validate-networking-v1-ingress"
)

// Ingress constants
const (
	// IngressPipyController is the name of the ingress controller
	IngressPipyController = "flomesh.io/ingress-pipy"

	// IngressPipyClass is the name of the ingress class
	IngressPipyClass = "pipy"

	// NoDefaultIngressClass is the name of the undefined ingress class
	NoDefaultIngressClass = ""

	// IngressAnnotationKey is the key of the ingress class annotation
	IngressAnnotationKey = "kubernetes.io/ingress.class"

	// IngressClassAnnotationKey is the key of the default ingress class annotation
	IngressClassAnnotationKey = "ingressclass.kubernetes.io/is-default-class"

	// PipyIngressAnnotationPrefix is the prefix of the pipy ingress annotations
	PipyIngressAnnotationPrefix = "pipy.ingress.kubernetes.io"

	// PipyIngressAnnotationRewriteFrom is the annotation used to indicate the rewrite target from
	PipyIngressAnnotationRewriteFrom = PipyIngressAnnotationPrefix + "/rewrite-target-from"

	// PipyIngressAnnotationRewriteTo is the annotation used to indicate the rewrite target to
	PipyIngressAnnotationRewriteTo = PipyIngressAnnotationPrefix + "/rewrite-target-to"

	// PipyIngressAnnotationSessionSticky is the annotation used to indicate the session sticky
	PipyIngressAnnotationSessionSticky = PipyIngressAnnotationPrefix + "/session-sticky"

	// PipyIngressAnnotationLoadBalancer is the annotation used to indicate the load balancer type
	PipyIngressAnnotationLoadBalancer = PipyIngressAnnotationPrefix + "/lb-type"

	// PipyIngressAnnotationUpstreamSSLName is the annotation used to indicate the upstream ssl name
	PipyIngressAnnotationUpstreamSSLName = PipyIngressAnnotationPrefix + "/upstream-ssl-name"

	// PipyIngressAnnotationUpstreamSSLSecret is the annotation used to indicate the upstream ssl secret
	PipyIngressAnnotationUpstreamSSLSecret = PipyIngressAnnotationPrefix + "/upstream-ssl-secret"

	// PipyIngressAnnotationUpstreamSSLVerify is the annotation used to indicate the upstream ssl verify
	PipyIngressAnnotationUpstreamSSLVerify = PipyIngressAnnotationPrefix + "/upstream-ssl-verify"

	// PipyIngressAnnotationTLSVerifyClient is the annotation used to indicate the tls verify client
	PipyIngressAnnotationTLSVerifyClient = PipyIngressAnnotationPrefix + "/tls-verify-client"

	// PipyIngressAnnotationTLSVerifyDepth is the annotation used to indicate the tls verify depth
	PipyIngressAnnotationTLSVerifyDepth = PipyIngressAnnotationPrefix + "/tls-verify-depth"

	// PipyIngressAnnotationTLSTrustedCASecret is the annotation used to indicate the tls trusted ca secret
	PipyIngressAnnotationTLSTrustedCASecret = PipyIngressAnnotationPrefix + "/tls-trusted-ca-secret"

	// PipyIngressAnnotationBackendProtocol is the annotation used to indicate the backend protocol
	PipyIngressAnnotationBackendProtocol = PipyIngressAnnotationPrefix + "/upstream-protocol"
)

// Ingress variables
var (
	// DefaultIngressClass is the default ingress class
	DefaultIngressClass = ""

	//// MinK8sVersionForIngressV1 is the minimum k8s version for ingress v1
	//MinK8sVersionForIngressV1 = semver.Version{Major: 1, Minor: 19, Patch: 0}
	//
	//// MinK8sVersionForIngressV1beta1 is the minimum k8s version for ingress v1beta1
	//MinK8sVersionForIngressV1beta1 = semver.Version{Major: 1, Minor: 16, Patch: 0}
	//
	//// MinK8sVersionForIngressClassV1beta1 is the minimum k8s version for ingress class v1beta1
	//MinK8sVersionForIngressClassV1beta1 = semver.Version{Major: 1, Minor: 18, Patch: 0}
)

// Helm Chart variables
var (
	// KubeVersion119 is the Kubernetes version 1.19 for helm chart rendering
	KubeVersion119 = &chartutil.KubeVersion{
		Version: fmt.Sprintf("v%s.%s.0", "1", "19"),
		Major:   "1",
		Minor:   "19",
	}

	// KubeVersion121 is the Kubernetes version 1.21 for helm chart rendering
	KubeVersion121 = &chartutil.KubeVersion{
		Version: fmt.Sprintf("v%s.%s.0", "1", "21"),
		Major:   "1",
		Minor:   "21",
	}
)
