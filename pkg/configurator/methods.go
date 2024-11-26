package configurator

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

const (
	// defaultServiceCertValidityDuration is the default validity duration for service certificates
	defaultServiceCertValidityDuration = 24 * time.Hour

	// defaultIngressGatewayCertValidityDuration is the default validity duration for ingress gateway certificates
	defaultIngressGatewayCertValidityDuration = 24 * time.Hour

	// defaultCertKeyBitSize is the default certificate key bit size
	defaultCertKeyBitSize = 2048

	// minCertKeyBitSize is the minimum certificate key bit size
	minCertKeyBitSize = 2048

	// maxCertKeyBitSize is the maximum certificate key bit size
	maxCertKeyBitSize = 4096
)

// The functions in this file implement the configurator.Configurator interface

// GetMeshConfig returns the MeshConfig resource corresponding to the control plane
func (c *Client) GetMeshConfig() configv1alpha3.MeshConfig {
	return c.getMeshConfig()
}

// GetFSMNamespace returns the namespace in which the FSM controller pod resides.
func (c *Client) GetFSMNamespace() string {
	return c.fsmNamespace
}

func marshalConfigToJSON(config configv1alpha3.MeshConfigSpec) (string, error) {
	bytes, err := json.MarshalIndent(&config, "", "    ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetMeshConfigJSON returns the MeshConfig in pretty JSON.
func (c *Client) GetMeshConfigJSON() (string, error) {
	cm, err := marshalConfigToJSON(c.getMeshConfig().Spec)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigMarshaling)).Msgf("Error marshaling MeshConfig %s: %+v", c.getMeshConfigCacheKey(), c.getMeshConfig())
		return "", err
	}
	return cm, nil
}

// GetTrafficInterceptionMode returns the traffic interception mode
func (c *Client) GetTrafficInterceptionMode() string {
	return c.getMeshConfig().Spec.Traffic.InterceptionMode
}

// IsPermissiveTrafficPolicyMode tells us whether the FSM Control Plane is in permissive mode,
// where all existing traffic is allowed to flow as it is,
// or it is in SMI Spec mode, in which only traffic between source/destinations
// referenced in SMI policies is allowed.
func (c *Client) IsPermissiveTrafficPolicyMode() bool {
	return c.getMeshConfig().Spec.Traffic.EnablePermissiveTrafficPolicyMode
}

// GetServiceAccessMode tells us which service access mode,
func (c *Client) GetServiceAccessMode() configv1alpha3.ServiceAccessMode {
	return c.getMeshConfig().Spec.Traffic.ServiceAccessMode
}

// GetServiceAccessNames returns the service access names
func (c *Client) GetServiceAccessNames() *configv1alpha3.ServiceAccessNames {
	return c.getMeshConfig().Spec.Traffic.ServiceAccessNames
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *Client) IsEgressEnabled() bool {
	return c.getMeshConfig().Spec.Traffic.EnableEgress
}

// IsTracingEnabled returns whether tracing is enabled
func (c *Client) IsTracingEnabled() bool {
	return c.getMeshConfig().Spec.Observability.Tracing.Enable
}

// IsLocalDNSProxyEnabled returns whether local DNS proxy is enabled
func (c *Client) IsLocalDNSProxyEnabled() bool {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.Enable
}

// IsWildcardDNSProxyEnabled returns whether wildcard DNS proxy is enabled
func (c *Client) IsWildcardDNSProxyEnabled() bool {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.Wildcard.Enable
}

// GetLocalDNSProxyPrimaryUpstream returns the primary upstream DNS server for local DNS Proxy
func (c *Client) GetLocalDNSProxyPrimaryUpstream() string {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.PrimaryUpstreamDNSServerIPAddr
}

// GetLocalDNSProxySecondaryUpstream returns the secondary upstream DNS server for local DNS Proxy
func (c *Client) GetLocalDNSProxySecondaryUpstream() string {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.SecondaryUpstreamDNSServerIPAddr
}

// GenerateIPv6BasedOnIPv4 returns whether auto generate IPv6 based on IPv4
func (c *Client) GenerateIPv6BasedOnIPv4() bool {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.GenerateIPv6BasedOnIPv4
}

// GetTracingHost is the host to which we send tracing spans
func (c *Client) GetTracingHost() string {
	tracingAddress := c.getMeshConfig().Spec.Observability.Tracing.Address
	if tracingAddress != "" {
		return tracingAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultTracingHost, c.GetFSMNamespace())
}

// GetTracingPort returns the tracing listener port
func (c *Client) GetTracingPort() uint32 {
	tracingPort := c.getMeshConfig().Spec.Observability.Tracing.Port
	if tracingPort != 0 {
		return uint32(tracingPort)
	}
	return constants.DefaultTracingPort
}

// GetTracingEndpoint returns the listener's collector endpoint
func (c *Client) GetTracingEndpoint() string {
	tracingEndpoint := c.getMeshConfig().Spec.Observability.Tracing.Endpoint
	if tracingEndpoint != "" {
		return tracingEndpoint
	}
	return constants.DefaultTracingEndpoint
}

// GetTracingSampledFraction returns the sampled fraction
func (c *Client) GetTracingSampledFraction() float32 {
	sampledFraction := c.getMeshConfig().Spec.Observability.Tracing.SampledFraction
	if sampledFraction != nil && len(*sampledFraction) > 0 {
		if v, e := strconv.ParseFloat(*sampledFraction, 32); e == nil {
			return float32(v)
		}
	}
	return 1
}

// IsRemoteLoggingEnabled returns whether remote logging is enabled
func (c *Client) IsRemoteLoggingEnabled() bool {
	return c.getMeshConfig().Spec.Observability.RemoteLogging.Enable
}

// GetRemoteLoggingLevel returns the remote logging level
func (c *Client) GetRemoteLoggingLevel() uint16 {
	return c.getMeshConfig().Spec.Observability.RemoteLogging.Level
}

// GetRemoteLoggingHost is the host to which we send logging spans
func (c *Client) GetRemoteLoggingHost() string {
	remoteLoggingAddress := c.getMeshConfig().Spec.Observability.RemoteLogging.Address
	if remoteLoggingAddress != "" {
		return remoteLoggingAddress
	}
	return ""
}

// GetRemoteLoggingPort returns the remote logging listener port
func (c *Client) GetRemoteLoggingPort() uint32 {
	remoteLoggingPort := c.getMeshConfig().Spec.Observability.RemoteLogging.Port
	if remoteLoggingPort != 0 {
		return uint32(remoteLoggingPort)
	}
	return 0
}

// GetRemoteLoggingEndpoint returns the collector endpoint
func (c *Client) GetRemoteLoggingEndpoint() string {
	remoteLoggingEndpoint := c.getMeshConfig().Spec.Observability.RemoteLogging.Endpoint
	if remoteLoggingEndpoint != "" {
		return remoteLoggingEndpoint
	}
	return ""
}

// GetRemoteLoggingAuthorization returns the access entity that allows to authorize someone in remote logging service.
func (c *Client) GetRemoteLoggingAuthorization() string {
	remoteLoggingAuthorization := c.getMeshConfig().Spec.Observability.RemoteLogging.Authorization
	if remoteLoggingAuthorization != "" {
		return remoteLoggingAuthorization
	}
	return ""
}

// GetRemoteLoggingSampledFraction returns the sampled fraction
func (c *Client) GetRemoteLoggingSampledFraction() float32 {
	sampledFraction := c.getMeshConfig().Spec.Observability.RemoteLogging.SampledFraction
	if sampledFraction != nil && len(*sampledFraction) > 0 {
		if v, e := strconv.ParseFloat(*sampledFraction, 32); e == nil {
			return float32(v)
		}
	}
	return 1
}

// GetRemoteLoggingSecretName returns the name of the secret that contains the credentials to access the remote logging service.
func (c *Client) GetRemoteLoggingSecretName() string {
	return c.getMeshConfig().Spec.Observability.RemoteLogging.SecretName
}

// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
func (c *Client) GetMaxDataPlaneConnections() int {
	return c.getMeshConfig().Spec.Sidecar.MaxDataPlaneConnections
}

// GetSidecarTimeout returns connect/idle/read/write timeout
func (c *Client) GetSidecarTimeout() int {
	timeout := c.getMeshConfig().Spec.Sidecar.SidecarTimeout
	if timeout <= 0 {
		timeout = 60
	}
	return timeout
}

// GetSidecarLogLevel returns the sidecar log level
func (c *Client) GetSidecarLogLevel() string {
	logLevel := c.getMeshConfig().Spec.Sidecar.LogLevel
	if logLevel != "" {
		return logLevel
	}
	return constants.DefaultSidecarLogLevel
}

// GetSidecarClass returns the sidecar class
func (c *Client) GetSidecarClass() string {
	return constants.SidecarClassPipy
}

// GetSidecarImage returns the sidecar image
func (c *Client) GetSidecarImage() string {
	image := c.getMeshConfig().Spec.Sidecar.SidecarImage
	if len(image) == 0 {
		image = os.Getenv("FSM_DEFAULT_SIDECAR_IMAGE")
	}
	return image
}

// GetInitContainerImage returns the init container image
func (c *Client) GetInitContainerImage() string {
	return os.Getenv("FSM_DEFAULT_INIT_CONTAINER_IMAGE")
}

// GetProxyServerPort returns the port on which the Discovery Service listens for new connections from Sidecars
func (c *Client) GetProxyServerPort() uint32 {
	port := c.getMeshConfig().Spec.RepoServer.Port
	if port > 0 {
		return uint32(port)
	}
	return constants.ProxyServerPort
}

// GetSidecarDisabledMTLS returns the status of mTLS
func (c *Client) GetSidecarDisabledMTLS() bool {
	return c.getMeshConfig().Spec.Sidecar.SidecarDisabledMTLS
}

// GetRepoServerIPAddr returns the ip address of RepoServer
func (c *Client) GetRepoServerIPAddr() string {
	ipAddr := os.Getenv("FSM_REPO_SERVER_IPADDR")
	if len(ipAddr) == 0 {
		ipAddr = c.getMeshConfig().Spec.RepoServer.IPAddr
	}
	if len(ipAddr) == 0 {
		ipAddr = "127.0.0.1"
	}
	return ipAddr
}

// GetRepoServerCodebase returns the codebase of RepoServer
func (c *Client) GetRepoServerCodebase() string {
	codebase := os.Getenv("FSM_REPO_SERVER_CODEBASE")
	if len(codebase) == 0 {
		codebase = c.getMeshConfig().Spec.RepoServer.Codebase
	}
	if len(codebase) > 0 && strings.HasSuffix(codebase, "/") {
		codebase = strings.TrimSuffix(codebase, "/")
	}
	if len(codebase) > 0 && strings.HasPrefix(codebase, "/") {
		codebase = strings.TrimPrefix(codebase, "/")
	}
	return codebase
}

// GetServiceCertValidityPeriod returns the validity duration for service certificates, and a default in case of invalid duration
func (c *Client) GetServiceCertValidityPeriod() time.Duration {
	durationStr := c.getMeshConfig().Spec.Certificate.ServiceCertValidityDuration
	validityDuration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing service certificate validity duration %s", durationStr)
		return defaultServiceCertValidityDuration
	}

	return validityDuration
}

// GetIngressGatewayCertValidityPeriod returns the validity duration for ingress gateway certificates, and a default in case of unspecified or invalid duration
func (c *Client) GetIngressGatewayCertValidityPeriod() time.Duration {
	ingressGatewayCertSpec := c.getMeshConfig().Spec.Certificate.IngressGateway
	if ingressGatewayCertSpec == nil {
		log.Warn().Msgf("Attempting to get the ingress gateway certificate validity duration even though a cert has not been specified in the mesh config")
		return defaultIngressGatewayCertValidityDuration
	}
	validityDuration, err := time.ParseDuration(ingressGatewayCertSpec.ValidityDuration)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing ingress gateway certificate validity duration %s", ingressGatewayCertSpec.ValidityDuration)
		return defaultServiceCertValidityDuration
	}

	return validityDuration
}

// GetCertKeyBitSize returns the certificate key bit size to be used
func (c *Client) GetCertKeyBitSize() int {
	bitSize := c.getMeshConfig().Spec.Certificate.CertKeyBitSize
	if bitSize < minCertKeyBitSize || bitSize > maxCertKeyBitSize {
		log.Error().Msgf("Invalid key bit size: %d", bitSize)
		return defaultCertKeyBitSize
	}

	return bitSize
}

// IsPrivilegedInitContainer returns whether init containers should be privileged
func (c *Client) IsPrivilegedInitContainer() bool {
	return c.getMeshConfig().Spec.Sidecar.EnablePrivilegedInitContainer
}

// GetConfigResyncInterval returns the duration for resync interval.
// If error or non-parsable value, returns 0 duration
func (c *Client) GetConfigResyncInterval() time.Duration {
	resyncDuration := c.getMeshConfig().Spec.Sidecar.ConfigResyncInterval
	duration, err := time.ParseDuration(resyncDuration)
	if err != nil {
		log.Warn().Msgf("Error parsing config resync interval: %s", duration)
		return time.Duration(0)
	}
	return duration
}

// GetProxyResources returns the `Resources` configured for proxies, if any
func (c *Client) GetProxyResources() corev1.ResourceRequirements {
	return c.getMeshConfig().Spec.Sidecar.Resources
}

// GetInjectedInitResources returns the `Resources` configured for proxies, if any
func (c *Client) GetInjectedInitResources() corev1.ResourceRequirements {
	return c.getMeshConfig().Spec.Sidecar.InitResources
}

// GetInjectedHealthcheckResources returns the `Resources` configured for proxies, if any
func (c *Client) GetInjectedHealthcheckResources() corev1.ResourceRequirements {
	return c.getMeshConfig().Spec.Sidecar.HealthcheckResources
}

// GetFeatureFlags returns FSM's feature flags
func (c *Client) GetFeatureFlags() configv1alpha3.FeatureFlags {
	return c.getMeshConfig().Spec.FeatureFlags
}

// GetFSMLogLevel returns the configured FSM log level
func (c *Client) GetFSMLogLevel() string {
	return c.getMeshConfig().Spec.Observability.FSMLogLevel
}

// GetGlobalPluginChains returns plugin chains
func (c *Client) GetGlobalPluginChains() map[string][]trafficpolicy.Plugin {
	pluginChainMap := make(map[string][]trafficpolicy.Plugin)
	pluginChainSpec := c.getMeshConfig().Spec.PluginChains

	inboundTCPChains := make([]trafficpolicy.Plugin, 0)
	for _, plugin := range pluginChainSpec.InboundTCPChains {
		if plugin.Disable {
			continue
		}
		inboundTCPChains = append(inboundTCPChains, trafficpolicy.Plugin{
			Name:     plugin.Plugin,
			Priority: plugin.Priority,
			BuildIn:  true,
		})
	}

	inboundHTTPChains := make([]trafficpolicy.Plugin, 0)
	for _, plugin := range pluginChainSpec.InboundHTTPChains {
		if plugin.Disable {
			continue
		}
		inboundHTTPChains = append(inboundHTTPChains, trafficpolicy.Plugin{
			Name:     plugin.Plugin,
			Priority: plugin.Priority,
			BuildIn:  true,
		})
	}

	outboundTCPChains := make([]trafficpolicy.Plugin, 0)
	for _, plugin := range pluginChainSpec.OutboundTCPChains {
		if plugin.Disable {
			continue
		}
		outboundTCPChains = append(outboundTCPChains, trafficpolicy.Plugin{
			Name:     plugin.Plugin,
			Priority: plugin.Priority,
			BuildIn:  true,
		})
	}

	outboundHTTPChains := make([]trafficpolicy.Plugin, 0)
	for _, plugin := range pluginChainSpec.OutboundHTTPChains {
		if plugin.Disable {
			continue
		}
		outboundHTTPChains = append(outboundHTTPChains, trafficpolicy.Plugin{
			Name:     plugin.Plugin,
			Priority: plugin.Priority,
			BuildIn:  true,
		})
	}

	pluginChainMap["inbound-tcp"] = inboundTCPChains
	pluginChainMap["inbound-http"] = inboundHTTPChains
	pluginChainMap["outbound-tcp"] = outboundTCPChains
	pluginChainMap["outbound-http"] = outboundHTTPChains
	return pluginChainMap
}

// IsGatewayAPIEnabled returns whether GatewayAPI is enabled
func (c *Client) IsGatewayAPIEnabled() bool {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.GatewayAPI.Enabled && !mcSpec.Ingress.Enabled
}

// GetFSMGatewayLogLevel returns log level of FSM Gateway
func (c *Client) GetFSMGatewayLogLevel() string {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.GatewayAPI.LogLevel
}

// IsIngressEnabled returns whether Ingress is enabled
func (c *Client) IsIngressEnabled() bool {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.Ingress.Enabled && !mcSpec.GatewayAPI.Enabled
}

// IsNamespacedIngressEnabled returns whether Namespaced Ingress is enabled
func (c *Client) IsNamespacedIngressEnabled() bool {
	mcSpec := c.getMeshConfig().Spec
	return c.IsIngressEnabled() && mcSpec.Ingress.Namespaced
}

// IsServiceLBEnabled returns whether ServiceLB is enabled
func (c *Client) IsServiceLBEnabled() bool {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.ServiceLB.Enabled
}

// IsFLBEnabled returns whether FLB is enabled
func (c *Client) IsFLBEnabled() bool {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.FLB.Enabled
}

func (c *Client) GetFLBUpstreamMode() configv1alpha3.FLBUpstreamMode {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.FLB.UpstreamMode
}

// IsMultiClusterControlPlane returns whether current cluster is the control plane of a multi cluster set
func (c *Client) IsMultiClusterControlPlane() bool {
	clusterSet := c.getMeshConfig().Spec.ClusterSet

	return clusterSet.ControlPlaneUID == "" ||
		clusterSet.UID == clusterSet.ControlPlaneUID
}

// GetImageRegistry returns the image registry
func (c *Client) GetImageRegistry() string {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.Image.Registry
}

// GetImageTag returns the image tag
func (c *Client) GetImageTag() string {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.Image.Tag
}

func (c *Client) GetImagePullPolicy() corev1.PullPolicy {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.Image.PullPolicy
}

// ServiceLBImage returns the image for service load balancer
func (c *Client) ServiceLBImage() string {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.ServiceLB.Image
}

// GetFLBSecretName returns the secret name for FLB
func (c *Client) GetFLBSecretName() string {
	return c.getMeshConfig().Spec.FLB.SecretName
}

// IsFLBStrictModeEnabled returns whether FLB is in strict mode
func (c *Client) IsFLBStrictModeEnabled() bool {
	return c.getMeshConfig().Spec.FLB.StrictMode
}

// IsManaged returns whether the cluster is managed
func (c *Client) IsManaged() bool {
	return c.getMeshConfig().Spec.ClusterSet.IsManaged
}

// GetClusterUID returns the UID of the cluster
func (c *Client) GetClusterUID() string {
	return c.getMeshConfig().Spec.ClusterSet.UID
}

// GetMultiClusterControlPlaneUID returns the UID of the control plane of the multi cluster set
func (c *Client) GetMultiClusterControlPlaneUID() string {
	return c.getMeshConfig().Spec.ClusterSet.ControlPlaneUID
}

// IsIngressTLSEnabled returns whether TLS is enabled for ingress
func (c *Client) IsIngressTLSEnabled() bool {
	tls := c.getMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		return tls.Enabled
	}

	return false
}

// GetIngressTLSListenPort returns the port that ingress listens on for TLS
func (c *Client) GetIngressTLSListenPort() int32 {
	tls := c.getMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		return tls.Listen
	}

	return 443
}

// IsIngressMTLSEnabled returns whether mTLS is enabled for ingress
func (c *Client) IsIngressMTLSEnabled() bool {
	tls := c.getMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		return tls.MTLS
	}

	return false
}

// IsIngressSSLPassthroughEnabled returns whether SSL Passthrough is enabled for ingress
func (c *Client) IsIngressSSLPassthroughEnabled() bool {
	tls := c.getMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		if passthrough := tls.SSLPassthrough; passthrough != nil {
			return passthrough.Enabled
		}

		return false
	}

	return false
}

// GetIngressSSLPassthroughUpstreamPort returns the port that ingress listens on for SSL Passthrough
func (c *Client) GetIngressSSLPassthroughUpstreamPort() int32 {
	tls := c.getMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		if passthrough := tls.SSLPassthrough; passthrough != nil {
			return passthrough.UpstreamPort
		}

		return 443
	}

	return 443
}

// IsIngressHTTPEnabled returns whether HTTP is enabled for ingress
func (c *Client) IsIngressHTTPEnabled() bool {
	http := c.getMeshConfig().Spec.Ingress.HTTP
	if http != nil {
		return http.Enabled
	}

	return false
}

// GetIngressHTTPListenPort returns the port that ingress listens on for HTTP
func (c *Client) GetIngressHTTPListenPort() int32 {
	http := c.getMeshConfig().Spec.Ingress.HTTP
	if http != nil {
		return http.Listen
	}

	return 80
}

// GetFSMIngressLogLevel returns the log level of ingress
func (c *Client) GetFSMIngressLogLevel() string {
	mcSpec := c.getMeshConfig().Spec
	return mcSpec.Ingress.LogLevel
}
