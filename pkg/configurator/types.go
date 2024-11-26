// Package configurator implements the Configurator interface that provides APIs to retrieve FSM control plane configurations.
package configurator

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

var (
	log = logger.New("configurator")
)

// Client is the type used to represent the Kubernetes Client for the config.flomesh.io API group
type Client struct {
	fsmNamespace   string
	informers      *informers.InformerCollection
	meshConfigName string
}

// Configurator is the controller interface for K8s namespaces
type Configurator interface {
	// GetMeshConfig returns the MeshConfig resource corresponding to the control plane
	GetMeshConfig() configv1alpha3.MeshConfig

	// GetFSMNamespace returns the namespace in which FSM controller pod resides
	GetFSMNamespace() string

	// GetMeshConfigJSON returns the MeshConfig in pretty JSON (human readable)
	GetMeshConfigJSON() (string, error)

	// GetTrafficInterceptionMode returns the traffic interception mode
	GetTrafficInterceptionMode() string

	// IsPermissiveTrafficPolicyMode determines whether we are in "allow-all" mode or SMI policy (block by default) mode
	IsPermissiveTrafficPolicyMode() bool

	// GetServiceAccessMode returns the service access mode
	GetServiceAccessMode() configv1alpha3.ServiceAccessMode

	// GetServiceAccessNames returns the service access names
	GetServiceAccessNames() *configv1alpha3.ServiceAccessNames

	// IsEgressEnabled determines whether egress is globally enabled in the mesh or not
	IsEgressEnabled() bool

	// IsTracingEnabled returns whether tracing is enabled
	IsTracingEnabled() bool

	// IsLocalDNSProxyEnabled returns whether local DNS proxy is enabled
	IsLocalDNSProxyEnabled() bool

	// IsWildcardDNSProxyEnabled returns whether wildcard DNS proxy is enabled
	IsWildcardDNSProxyEnabled() bool

	// GetLocalDNSProxyPrimaryUpstream returns the primary upstream DNS server for local DNS Proxy
	GetLocalDNSProxyPrimaryUpstream() string

	// GetLocalDNSProxySecondaryUpstream returns the secondary upstream DNS server for local DNS Proxy
	GetLocalDNSProxySecondaryUpstream() string

	// GenerateIPv6BasedOnIPv4 returns whether auto generate IPv6 based on IPv4
	GenerateIPv6BasedOnIPv4() bool

	// GetTracingHost is the host to which we send tracing spans
	GetTracingHost() string

	// GetTracingPort returns the tracing listener port
	GetTracingPort() uint32

	// GetTracingEndpoint returns the collector endpoint
	GetTracingEndpoint() string

	// GetTracingSampledFraction returns the sampled fraction
	GetTracingSampledFraction() float32

	// IsRemoteLoggingEnabled returns whether remote logging is enabled
	IsRemoteLoggingEnabled() bool

	// GetRemoteLoggingLevel returns the remote logging level
	GetRemoteLoggingLevel() uint16

	// GetRemoteLoggingHost is the host to which we send logging spans
	GetRemoteLoggingHost() string

	// GetRemoteLoggingPort returns the remote logging listener port
	GetRemoteLoggingPort() uint32

	// GetRemoteLoggingEndpoint returns the collector endpoint
	GetRemoteLoggingEndpoint() string

	// GetRemoteLoggingAuthorization returns the access entity that allows to authorize someone in remote logging service.
	GetRemoteLoggingAuthorization() string

	// GetRemoteLoggingSampledFraction returns the sampled fraction
	GetRemoteLoggingSampledFraction() float32

	// GetRemoteLoggingSecretName returns the name of secret that contains the access entity that allows to authorize someone in remote logging service.
	GetRemoteLoggingSecretName() string

	// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
	GetMaxDataPlaneConnections() int

	// GetSidecarTimeout returns connect/idle/read/write timeout
	GetSidecarTimeout() int

	// GetFSMLogLevel returns the configured FSM log level
	GetFSMLogLevel() string

	// GetSidecarLogLevel returns the sidecar log level
	GetSidecarLogLevel() string

	// GetSidecarClass returns the sidecar class
	GetSidecarClass() string

	// GetSidecarImage returns the sidecar image
	GetSidecarImage() string

	// GetInitContainerImage returns the init container image
	GetInitContainerImage() string

	// GetProxyServerPort returns the port on which the Discovery Service listens for new connections from Sidecars
	GetProxyServerPort() uint32

	// GetSidecarDisabledMTLS returns the status of mTLS
	GetSidecarDisabledMTLS() bool

	// GetRepoServerIPAddr returns the ip address of RepoServer
	GetRepoServerIPAddr() string

	// GetRepoServerCodebase returns the codebase of RepoServer
	GetRepoServerCodebase() string

	// GetServiceCertValidityPeriod returns the validity duration for service certificates
	GetServiceCertValidityPeriod() time.Duration

	// GetIngressGatewayCertValidityPeriod returns the validity duration for the Ingress
	// Gateway certificate, default value if not specified
	GetIngressGatewayCertValidityPeriod() time.Duration

	// GetCertKeyBitSize returns the certificate key bit size
	GetCertKeyBitSize() int

	// IsPrivilegedInitContainer determines whether init containers should be privileged
	IsPrivilegedInitContainer() bool

	// GetConfigResyncInterval returns the duration for resync interval.
	// If error or non-parsable value, returns 0 duration
	GetConfigResyncInterval() time.Duration

	// GetProxyResources returns the `Resources` configured for proxies, if any
	GetProxyResources() corev1.ResourceRequirements

	// GetInjectedInitResources returns the `Resources` configured for proxies, if any
	GetInjectedInitResources() corev1.ResourceRequirements

	// GetInjectedHealthcheckResources returns the `Resources` configured for proxies, if any
	GetInjectedHealthcheckResources() corev1.ResourceRequirements

	// GetFeatureFlags returns FSM's feature flags
	GetFeatureFlags() configv1alpha3.FeatureFlags

	// GetGlobalPluginChains returns plugin chains
	GetGlobalPluginChains() map[string][]trafficpolicy.Plugin

	// IsGatewayAPIEnabled returns whether GatewayAPI is enabled
	IsGatewayAPIEnabled() bool

	// GetFSMGatewayLogLevel returns log level of FSM Gateway
	GetFSMGatewayLogLevel() string

	// IsIngressEnabled returns whether Ingress is enabled
	IsIngressEnabled() bool

	// IsIngressTLSEnabled returns whether Ingress is enabled
	IsIngressTLSEnabled() bool

	// GetIngressTLSListenPort returns the listen port of Ingress TLS
	GetIngressTLSListenPort() int32

	// IsIngressMTLSEnabled returns whether Ingress mTLS is enabled
	IsIngressMTLSEnabled() bool

	// IsIngressSSLPassthroughEnabled returns whether Ingress is enabled
	IsIngressSSLPassthroughEnabled() bool

	// GetIngressSSLPassthroughUpstreamPort returns the upstream port for SSL passthrough
	GetIngressSSLPassthroughUpstreamPort() int32

	// IsNamespacedIngressEnabled returns whether Namespaced Ingress is enabled
	IsNamespacedIngressEnabled() bool

	// IsIngressHTTPEnabled returns whether http port of Ingress is enabled
	IsIngressHTTPEnabled() bool

	// GetIngressHTTPListenPort returns the listen port of Ingress HTTP
	GetIngressHTTPListenPort() int32

	// GetFSMIngressLogLevel returns log level of FSM Gateway
	GetFSMIngressLogLevel() string

	// IsServiceLBEnabled returns whether ServiceLB is enabled
	IsServiceLBEnabled() bool

	// IsFLBEnabled returns whether FLB is enabled
	IsFLBEnabled() bool

	// GetFLBSecretName returns the secret name of FLB
	GetFLBSecretName() string

	// IsFLBStrictModeEnabled returns whether FLB Strict Mode is enabled
	IsFLBStrictModeEnabled() bool

	// IsMultiClusterControlPlane returns whether current cluster is the control plane of a multi cluster set
	IsMultiClusterControlPlane() bool

	// IsManaged returns whether the current cluster is managed by multicluster control plane
	IsManaged() bool

	// GetClusterUID returns the UID of the cluster
	GetClusterUID() string

	// GetMultiClusterControlPlaneUID returns the UID of the control plane if it's managed, or empty if not managed
	GetMultiClusterControlPlaneUID() string

	// GetImageRegistry returns the image registry
	GetImageRegistry() string

	// GetImageTag returns the image tag
	GetImageTag() string

	// GetImagePullPolicy returns the image pull policy
	GetImagePullPolicy() corev1.PullPolicy

	// ServiceLBImage string returns the service-lb image
	ServiceLBImage() string

	// GetFLBUpstreamMode returns the upstream mode of FLB
	GetFLBUpstreamMode() configv1alpha3.FLBUpstreamMode
}
