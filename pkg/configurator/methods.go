package configurator

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	configv1alpha2 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/auth"
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
func (c *Client) GetMeshConfig() configv1alpha2.MeshConfig {
	return c.getMeshConfig()
}

// GetFSMNamespace returns the namespace in which the FSM controller pod resides.
func (c *Client) GetFSMNamespace() string {
	return c.fsmNamespace
}

func marshalConfigToJSON(config configv1alpha2.MeshConfigSpec) (string, error) {
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
func (c *Client) GetServiceAccessMode() string {
	return c.getMeshConfig().Spec.Traffic.ServiceAccessMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *Client) IsEgressEnabled() bool {
	return c.getMeshConfig().Spec.Traffic.EnableEgress
}

// IsDebugServerEnabled determines whether fsm debug HTTP server is enabled
func (c *Client) IsDebugServerEnabled() bool {
	return c.getMeshConfig().Spec.Observability.EnableDebugServer
}

// IsTracingEnabled returns whether tracing is enabled
func (c *Client) IsTracingEnabled() bool {
	return c.getMeshConfig().Spec.Observability.Tracing.Enable
}

// IsLocalDNSProxyEnabled returns whether local DNS proxy is enabled
func (c *Client) IsLocalDNSProxyEnabled() bool {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.Enable
}

// GetLocalDNSProxyPrimaryUpstream returns the primary upstream DNS server for local DNS Proxy
func (c *Client) GetLocalDNSProxyPrimaryUpstream() string {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.PrimaryUpstreamDNSServerIPAddr
}

// GetLocalDNSProxySecondaryUpstream returns the secondary upstream DNS server for local DNS Proxy
func (c *Client) GetLocalDNSProxySecondaryUpstream() string {
	return c.getMeshConfig().Spec.Sidecar.LocalDNSProxy.SecondaryUpstreamDNSServerIPAddr
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

// GetMaxDataPlaneConnections returns the max data plane connections allowed, 0 if disabled
func (c *Client) GetMaxDataPlaneConnections() int {
	return c.getMeshConfig().Spec.Sidecar.MaxDataPlaneConnections
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
	class := c.getMeshConfig().Spec.Sidecar.SidecarClass
	if class == "" {
		class = os.Getenv("FSM_DEFAULT_SIDECAR_CLASS")
	}
	if class == "" {
		class = constants.SidecarClassPipy
	}
	return class
}

// GetSidecarImage returns the sidecar image
func (c *Client) GetSidecarImage() string {
	image := c.getMeshConfig().Spec.Sidecar.SidecarImage
	if len(image) == 0 {
		sidecarClass := c.getMeshConfig().Spec.Sidecar.SidecarClass
		sidecarDrivers := c.getMeshConfig().Spec.Sidecar.SidecarDrivers
		for _, sidecarDriver := range sidecarDrivers {
			if strings.EqualFold(strings.ToLower(sidecarClass), strings.ToLower(sidecarDriver.SidecarName)) {
				image = sidecarDriver.SidecarImage
				break
			}
		}
	}
	if len(image) == 0 {
		image = os.Getenv("FSM_DEFAULT_SIDECAR_IMAGE")
	}
	return image
}

// GetInitContainerImage returns the init container image
func (c *Client) GetInitContainerImage() string {
	image := c.getMeshConfig().Spec.Sidecar.InitContainerImage
	if len(image) == 0 {
		sidecarClass := c.getMeshConfig().Spec.Sidecar.SidecarClass
		sidecarDrivers := c.getMeshConfig().Spec.Sidecar.SidecarDrivers
		for _, sidecarDriver := range sidecarDrivers {
			if strings.EqualFold(strings.ToLower(sidecarClass), strings.ToLower(sidecarDriver.SidecarName)) {
				image = sidecarDriver.InitContainerImage
				break
			}
		}
	}
	if len(image) == 0 {
		image = os.Getenv("FSM_DEFAULT_INIT_CONTAINER_IMAGE")
	}
	return image
}

// GetProxyServerPort returns the port on which the Discovery Service listens for new connections from Sidecars
func (c *Client) GetProxyServerPort() uint32 {
	sidecarClass := c.getMeshConfig().Spec.Sidecar.SidecarClass
	sidecarDrivers := c.getMeshConfig().Spec.Sidecar.SidecarDrivers
	for _, sidecarDriver := range sidecarDrivers {
		if strings.EqualFold(strings.ToLower(sidecarClass), strings.ToLower(sidecarDriver.SidecarName)) {
			return sidecarDriver.ProxyServerPort
		}
	}
	return constants.ProxyServerPort
}

// GetSidecarDisabledMTLS returns the status of mTLS
func (c *Client) GetSidecarDisabledMTLS() bool {
	disabledMTLS := false
	sidecarClass := c.getMeshConfig().Spec.Sidecar.SidecarClass
	sidecarDrivers := c.getMeshConfig().Spec.Sidecar.SidecarDrivers
	for _, sidecarDriver := range sidecarDrivers {
		if strings.EqualFold(strings.ToLower(sidecarClass), strings.ToLower(sidecarDriver.SidecarName)) {
			disabledMTLS = sidecarDriver.SidecarDisabledMTLS
			break
		}
	}
	return disabledMTLS
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

// GetInboundExternalAuthConfig returns the External Authentication configuration for incoming traffic, if any
func (c *Client) GetInboundExternalAuthConfig() auth.ExtAuthConfig {
	extAuthConfig := auth.ExtAuthConfig{}
	inboundExtAuthzMeshConfig := c.getMeshConfig().Spec.Traffic.InboundExternalAuthorization

	extAuthConfig.Enable = inboundExtAuthzMeshConfig.Enable
	extAuthConfig.Address = inboundExtAuthzMeshConfig.Address
	extAuthConfig.Port = uint16(inboundExtAuthzMeshConfig.Port)
	extAuthConfig.StatPrefix = inboundExtAuthzMeshConfig.StatPrefix
	extAuthConfig.FailureModeAllow = inboundExtAuthzMeshConfig.FailureModeAllow

	duration, err := time.ParseDuration(inboundExtAuthzMeshConfig.Timeout)
	if err != nil {
		log.Debug().Err(err).Msgf("ExternAuthzTimeout: Not a valid duration %s. defaulting to 1s.", duration)
		duration = 1 * time.Second
	}
	extAuthConfig.AuthzTimeout = duration

	return extAuthConfig
}

// GetFeatureFlags returns FSM's feature flags
func (c *Client) GetFeatureFlags() configv1alpha2.FeatureFlags {
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
