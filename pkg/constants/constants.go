// Package constants defines the constants that are used by multiple other packages within FSM.
package constants

import "time"

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

// Gateway API constants
const (
	GatewayController = "flomesh.io/gateway-controller"

	GatewayPrefix                       = "gateway.flomesh.io"
	GatewayMTLSAnnotation               = GatewayPrefix + "/mtls"
	GatewayUpstreamSSLNameAnnotation    = GatewayPrefix + "/upstream-ssl-name"
	GatewayUpstreamSSLSecretAnnotation  = GatewayPrefix + "/upstream-ssl-secret"
	GatewayUpstreamSSLVerifyAnnotation  = GatewayPrefix + "/upstream-ssl-verify"
	GatewayTLSVerifyClientAnnotation    = GatewayPrefix + "/tls-verify-client"
	GatewayTLSVerifyDepthAnnotation     = GatewayPrefix + "/tls-verify-depth"
	GatewayTLSTrustedCASecretAnnotation = GatewayPrefix + "/tls-trusted-ca-secret"
	GatewayBackendProtocolAnnotation    = GatewayPrefix + "/upstream-protocol"

	GatewayMutatingWebhookPath        = "/mutate-gateway-networking-k8s-io-v1beta1-gateway"
	GatewayValidatingWebhookPath      = "/validate-gateway-networking-k8s-io-v1beta1-gateway"
	GatewayClassMutatingWebhookPath   = "/mutate-gateway-networking-k8s-io-v1beta1-gatewayclass"
	GatewayClassValidatingWebhookPath = "/validate-gateway-networking-k8s-io-v1beta1-gatewayclass"
	HTTPRouteMutatingWebhookPath      = "/mutate-gateway-networking-k8s-io-v1beta1-httproute"
	HTTPRouteValidatingWebhookPath    = "/validate-gateway-networking-k8s-io-v1beta1-httproute"
	GRPCRouteMutatingWebhookPath      = "/mutate-gateway-networking-k8s-io-v1alpha2-grpcroute"
	GRPCRouteValidatingWebhookPath    = "/validate-gateway-networking-k8s-io-v1alpha2-grpcroute"
	TCPRouteMutatingWebhookPath       = "/mutate-gateway-networking-k8s-io-v1alpha2-tcproute"
	TCPRouteValidatingWebhookPath     = "/validate-gateway-networking-k8s-io-v1alpha2-tcproute"
	TLSRouteMutatingWebhookPath       = "/mutate-gateway-networking-k8s-io-v1alpha2-tlsroute"
	TLSRouteValidatingWebhookPath     = "/validate-gateway-networking-k8s-io-v1alpha2-tlsroute"
)

// PIPY Repo constants
const (
	DefaultPipyRepoApiPath = "/api/v1/repo"
	DefaultPipyFileApiPath = "/api/v1/repo-files"
)
