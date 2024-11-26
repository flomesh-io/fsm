package listeners

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

type client struct {
	mc *configv1alpha3.MeshConfig
}

func (c *client) GetFLBUpstreamMode() configv1alpha3.FLBUpstreamMode {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRemoteLoggingLevel() uint16 {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetSidecarTimeout() int {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetFSMIngressLogLevel() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetImageTag() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetImagePullPolicy() corev1.PullPolicy {
	//TODO implement me
	panic("implement me")
}

func meshConfigToConfigurator(meshConfig *configv1alpha3.MeshConfig) configurator.Configurator {
	return &client{mc: meshConfig}
}

func (c *client) GetMeshConfig() configv1alpha3.MeshConfig {
	return *c.mc
}

func (c *client) GetFSMNamespace() string {
	return c.GetMeshConfig().Namespace
}

func (c *client) GetMeshConfigJSON() (string, error) {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetTrafficInterceptionMode() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsPermissiveTrafficPolicyMode() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetServiceAccessMode() configv1alpha3.ServiceAccessMode {
	//TODO implement me
	panic("implement me")
}

// GetServiceAccessNames returns the service access names
func (c *client) GetServiceAccessNames() *configv1alpha3.ServiceAccessNames {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsEgressEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsTracingEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsLocalDNSProxyEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsWildcardDNSProxyEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetLocalDNSProxyPrimaryUpstream() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetLocalDNSProxySecondaryUpstream() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GenerateIPv6BasedOnIPv4() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetTracingHost() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetTracingPort() uint32 {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetTracingEndpoint() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetTracingSampledFraction() float32 {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsRemoteLoggingEnabled() bool {
	return c.mc.Spec.Observability.RemoteLogging.Enable
}

func (c *client) GetRemoteLoggingHost() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRemoteLoggingPort() uint32 {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRemoteLoggingEndpoint() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRemoteLoggingAuthorization() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRemoteLoggingSampledFraction() float32 {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRemoteLoggingSecretName() string {
	return c.mc.Spec.Observability.RemoteLogging.SecretName
}

func (c *client) GetMaxDataPlaneConnections() int {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetFSMLogLevel() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetSidecarLogLevel() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetSidecarClass() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetSidecarImage() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetInitContainerImage() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetProxyServerPort() uint32 {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetSidecarDisabledMTLS() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRepoServerIPAddr() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetRepoServerCodebase() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetServiceCertValidityPeriod() time.Duration {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetIngressGatewayCertValidityPeriod() time.Duration {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetCertKeyBitSize() int {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsPrivilegedInitContainer() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetConfigResyncInterval() time.Duration {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetProxyResources() corev1.ResourceRequirements {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetInjectedInitResources() corev1.ResourceRequirements {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetInjectedHealthcheckResources() corev1.ResourceRequirements {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetFeatureFlags() configv1alpha3.FeatureFlags {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetGlobalPluginChains() map[string][]trafficpolicy.Plugin {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsGatewayAPIEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetFSMGatewayLogLevel() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsIngressEnabled() bool {
	return c.GetMeshConfig().Spec.Ingress.Enabled
}

func (c *client) IsIngressTLSEnabled() bool {
	tls := c.GetMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		return tls.Enabled
	}

	return false
}

func (c *client) GetIngressTLSListenPort() int32 {
	tls := c.GetMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		return tls.Listen
	}

	return 443
}

func (c *client) IsIngressMTLSEnabled() bool {
	tls := c.GetMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		return tls.MTLS
	}

	return false
}

func (c *client) IsIngressSSLPassthroughEnabled() bool {
	tls := c.GetMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		if passthrough := tls.SSLPassthrough; passthrough != nil {
			return passthrough.Enabled
		}

		return false
	}

	return false
}

func (c *client) GetIngressSSLPassthroughUpstreamPort() int32 {
	tls := c.GetMeshConfig().Spec.Ingress.TLS
	if tls != nil {
		if passthrough := tls.SSLPassthrough; passthrough != nil {
			return passthrough.UpstreamPort
		}

		return 443
	}

	return 443
}

func (c *client) IsIngressHTTPEnabled() bool {
	http := c.GetMeshConfig().Spec.Ingress.HTTP
	if http != nil {
		return http.Enabled
	}

	return false
}

func (c *client) GetIngressHTTPListenPort() int32 {
	http := c.GetMeshConfig().Spec.Ingress.HTTP
	if http != nil {
		return http.Listen
	}

	return 80
}

func (c *client) IsNamespacedIngressEnabled() bool {
	mcSpec := c.GetMeshConfig().Spec
	return c.IsIngressEnabled() && mcSpec.Ingress.Namespaced
}

func (c *client) IsServiceLBEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsFLBEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetFLBSecretName() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsFLBStrictModeEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsMultiClusterControlPlane() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) IsManaged() bool {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetClusterUID() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetMultiClusterControlPlaneUID() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) GetImageRegistry() string {
	//TODO implement me
	panic("implement me")
}

func (c *client) ServiceLBImage() string {
	//TODO implement me
	panic("implement me")
}
