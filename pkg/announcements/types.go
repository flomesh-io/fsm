// Package announcements provides the types and constants required to contextualize events received from the
// Kubernetes API server that are propagated internally within the control plane to trigger configuration changes.
package announcements

// Kind is used to record the kind of announcement
type Kind string

func (at Kind) String() string {
	return string(at)
}

const (
	// ProxyUpdate is the event kind used to trigger an update to subscribed proxies
	ProxyUpdate Kind = "proxy-update"

	// IngressUpdate is the event kind used to trigger an update to subscribed ingresses
	IngressUpdate Kind = "ingress-update"

	// GatewayUpdate is the event kind used to trigger an update to subscribed gateways
	GatewayUpdate Kind = "gateway-update"

	// MCSUpdate is the event kind used to trigger an update to subscribed gateways
	MCSUpdate Kind = "mcs-update"

	// PodAdded is the type of announcement emitted when we observe an addition of a Kubernetes Pod
	PodAdded Kind = "pod-added"

	// PodDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Pod
	PodDeleted Kind = "pod-deleted"

	// PodUpdated is the type of announcement emitted when we observe an update to a Kubernetes Pod
	PodUpdated Kind = "pod-updated"

	// ---

	// EndpointAdded is the type of announcement emitted when we observe an addition of a Kubernetes Endpoint
	EndpointAdded Kind = "endpoint-added"

	// EndpointDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Endpoint
	EndpointDeleted Kind = "endpoint-deleted"

	// EndpointUpdated is the type of announcement emitted when we observe an update to a Kubernetes Endpoint
	EndpointUpdated Kind = "endpoint-updated"

	// ---

	// NamespaceAdded is the type of announcement emitted when we observe an addition of a Kubernetes Namespace
	NamespaceAdded Kind = "namespace-added"

	// NamespaceDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Namespace
	NamespaceDeleted Kind = "namespace-deleted"

	// NamespaceUpdated is the type of announcement emitted when we observe an update to a Kubernetes Namespace
	NamespaceUpdated Kind = "namespace-updated"

	// ---

	// ServiceAdded is the type of announcement emitted when we observe an addition of a Kubernetes Service
	ServiceAdded Kind = "service-added"

	// ServiceDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Service
	ServiceDeleted Kind = "service-deleted"

	// ServiceUpdated is the type of announcement emitted when we observe an update to a Kubernetes Service
	ServiceUpdated Kind = "service-updated"

	// ---

	// ServiceAccountAdded is the type of announcement emitted when we observe an addition of a Kubernetes Service Account
	ServiceAccountAdded Kind = "serviceaccount-added"

	// ServiceAccountDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Service Account
	ServiceAccountDeleted Kind = "serviceaccount-deleted"

	// ServiceAccountUpdated is the type of announcement emitted when we observe an update to a Kubernetes Service
	ServiceAccountUpdated Kind = "serviceaccount-updated"

	// ---

	// TrafficSplitAdded is the type of announcement emitted when we observe an addition of a Kubernetes TrafficSplit
	TrafficSplitAdded Kind = "trafficsplit-added"

	// TrafficSplitDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TrafficSplit
	TrafficSplitDeleted Kind = "trafficsplit-deleted"

	// TrafficSplitUpdated is the type of announcement emitted when we observe an update to a Kubernetes TrafficSplit
	TrafficSplitUpdated Kind = "trafficsplit-updated"

	// ---

	// RouteGroupAdded is the type of announcement emitted when we observe an addition of a Kubernetes RouteGroup
	RouteGroupAdded Kind = "routegroup-added"

	// RouteGroupDeleted the type of announcement emitted when we observe the deletion of a Kubernetes RouteGroup
	RouteGroupDeleted Kind = "routegroup-deleted"

	// RouteGroupUpdated is the type of announcement emitted when we observe an update to a Kubernetes RouteGroup
	RouteGroupUpdated Kind = "routegroup-updated"

	// ---

	// TCPRouteAdded is the type of announcement emitted when we observe an addition of a Kubernetes TCPRoute
	TCPRouteAdded Kind = "tcproute-added"

	// TCPRouteDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TCPRoute
	TCPRouteDeleted Kind = "tcproute-deleted"

	// TCPRouteUpdated is the type of announcement emitted when we observe an update to a Kubernetes TCPRoute
	TCPRouteUpdated Kind = "tcproute-updated"

	// ---

	// TrafficTargetAdded is the type of announcement emitted when we observe an addition of a Kubernetes TrafficTarget
	TrafficTargetAdded Kind = "traffictarget-added"

	// TrafficTargetDeleted the type of announcement emitted when we observe the deletion of a Kubernetes TrafficTarget
	TrafficTargetDeleted Kind = "traffictarget-deleted"

	// TrafficTargetUpdated is the type of announcement emitted when we observe an update to a Kubernetes TrafficTarget
	TrafficTargetUpdated Kind = "traffictarget-updated"

	// ---

	// IngressAdded is the type of announcement emitted when we observe an addition of a Kubernetes Ingress
	IngressAdded Kind = "ingress-added"

	// IngressDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Ingress
	IngressDeleted Kind = "ingress-deleted"

	// IngressUpdated is the type of announcement emitted when we observe an update to a Kubernetes Ingress
	IngressUpdated Kind = "ingress-updated"

	// ---

	// IngressClassAdded is the type of announcement emitted when we observe an addition of a Kubernetes IngressClass
	IngressClassAdded Kind = "ingressclass-added"

	// IngressClassDeleted the type of announcement emitted when we observe the deletion of a Kubernetes IngressClass
	IngressClassDeleted Kind = "ingressclass-deleted"

	// IngressClassUpdated is the type of announcement emitted when we observe an update to a Kubernetes IngressClass
	IngressClassUpdated Kind = "ingressclass-updated"

	// ---

	// CertificateRotated is the type of announcement emitted when a certificate is rotated by the certificate provider
	CertificateRotated Kind = "certificate-rotated"

	// --- config.flomesh.io API events

	// MeshConfigAdded is the type of announcement emitted when we observe an addition of a Kubernetes MeshConfig
	MeshConfigAdded Kind = "meshconfig-added"

	// MeshConfigDeleted the type of announcement emitted when we observe the deletion of a Kubernetes MeshConfig
	MeshConfigDeleted Kind = "meshconfig-deleted"

	// MeshConfigUpdated is the type of announcement emitted when we observe an update to a Kubernetes MeshConfig
	MeshConfigUpdated Kind = "meshconfig-updated"

	// MeshRootCertificateAdded is the type of announcement emitted when we observe an addition of a Kubernetes MeshRootCertificate
	MeshRootCertificateAdded Kind = "meshrootcertificate-added"

	// MeshRootCertificateDeleted is the type of announcement emitted when we observe the deletion of a Kubernetes MeshRootCertificate
	MeshRootCertificateDeleted Kind = "meshrootcertificate-deleted"

	// MeshRootCertificateUpdated is the type of announcement emitted when we observe an update to a Kubernetes MeshRootCertificate
	MeshRootCertificateUpdated Kind = "meshrootcertificate-updated"

	// --- policy.flomesh.io API events

	// EgressAdded is the type of announcement emitted when we observe an addition of egresses.policy.flomesh.io
	EgressAdded Kind = "egress-added"

	// EgressDeleted the type of announcement emitted when we observe a deletion of egresses.policy.flomesh.io
	EgressDeleted Kind = "egress-deleted"

	// EgressUpdated is the type of announcement emitted when we observe an update to egresses.policy.flomesh.io
	EgressUpdated Kind = "egress-updated"

	// EgressGatewayAdded is the type of announcement emitted when we observe an addition of egressgateways.policy.flomesh.io
	EgressGatewayAdded Kind = "egressgateway-added"

	// EgressGatewayDeleted the type of announcement emitted when we observe a deletion of egressgateways.policy.flomesh.io
	EgressGatewayDeleted Kind = "egressgateway-deleted"

	// EgressGatewayUpdated is the type of announcement emitted when we observe an update to egressgateways.policy.flomesh.io
	EgressGatewayUpdated Kind = "egressgateway-updated"

	// IngressBackendAdded is the type of announcement emitted when we observe an addition of ingressbackends.policy.flomesh.io
	IngressBackendAdded Kind = "ingressbackend-added"

	// IngressBackendDeleted the type of announcement emitted when we observe a deletion of ingressbackends.policy.flomesh.io
	IngressBackendDeleted Kind = "ingressbackend-deleted"

	// IngressBackendUpdated is the type of announcement emitted when we observe an update to ingressbackends.policy.flomesh.io
	IngressBackendUpdated Kind = "ingressbackend-updated"

	// AccessControlAdded is the type of announcement emitted when we observe an addition of accesscontrols.policy.flomesh.io
	AccessControlAdded Kind = "accesscontrol-added"

	// AccessControlDeleted the type of announcement emitted when we observe a deletion of accesscontrols.policy.flomesh.io
	AccessControlDeleted Kind = "accesscontrol-deleted"

	// AccessControlUpdated is the type of announcement emitted when we observe an update to accesscontrols.policy.flomesh.io
	AccessControlUpdated Kind = "accesscontrol-updated"

	// AccessCertAdded is the type of announcement emitted when we observe an addition of accesscerts.policy.flomesh.io
	AccessCertAdded Kind = "accesscert-added"

	// AccessCertDeleted the type of announcement emitted when we observe a deletion of accesscerts.policy.flomesh.io
	AccessCertDeleted Kind = "accesscert-deleted"

	// AccessCertUpdated is the type of announcement emitted when we observe an update to accesscerts.policy.flomesh.io
	AccessCertUpdated Kind = "accesscert-updated"

	// ServiceExportAdded is the type of announcement emitted when we observe an addition of serviceexports.flomesh.io
	ServiceExportAdded Kind = "serviceexport-added"

	// ServiceExportDeleted the type of announcement emitted when we observe a deletion of serviceexports.flomesh.io
	ServiceExportDeleted Kind = "serviceexport-deleted"

	// ServiceExportUpdated is the type of announcement emitted when we observe an update to serviceexports.flomesh.io
	ServiceExportUpdated Kind = "serviceexport-updated"

	// MultiClusterServiceExportCreated is the type of announcement emitted when we observe a creation to serviceexports.flomesh.io
	MultiClusterServiceExportCreated Kind = "mcs-serviceexport-created"

	// MultiClusterServiceExportDeleted the type of announcement emitted when we observe a deletion of serviceexports.flomesh.io
	MultiClusterServiceExportDeleted Kind = "mcs-serviceexport-deleted"

	// MultiClusterServiceExportAccepted is the type of announcement emitted when we observe an accept to serviceexports.flomesh.io
	MultiClusterServiceExportAccepted Kind = "mcs-serviceexport-accepted"

	// MultiClusterServiceExportRejected is the type of announcement emitted when we observe a rejection to serviceexports.flomesh.io
	MultiClusterServiceExportRejected Kind = "mcs-serviceexport-rejected"

	// ServiceImportAdded is the type of announcement emitted when we observe an addition of serviceimports.flomesh.io
	ServiceImportAdded Kind = "serviceimport-added"

	// ServiceImportDeleted the type of announcement emitted when we observe a deletion of serviceimports.flomesh.io
	ServiceImportDeleted Kind = "serviceimport-deleted"

	// ServiceImportUpdated is the type of announcement emitted when we observe an update to serviceimports.flomesh.io
	ServiceImportUpdated Kind = "serviceimport-updated"

	// GlobalTrafficPolicyAdded is the type of announcement emitted when we observe an addition of serviceimports.flomesh.io
	GlobalTrafficPolicyAdded Kind = "globaltrafficpolicy-added"

	// GlobalTrafficPolicyDeleted the type of announcement emitted when we observe a deletion of serviceimports.flomesh.io
	GlobalTrafficPolicyDeleted Kind = "globaltrafficpolicy-deleted"

	// GlobalTrafficPolicyUpdated is the type of announcement emitted when we observe an update to serviceimports.flomesh.io
	GlobalTrafficPolicyUpdated Kind = "globaltrafficpolicy-updated"

	// RetryPolicyAdded is the type of announcement emitted when we observe an addition of retries.policy.flomesh.io
	RetryPolicyAdded Kind = "retry-added"

	// RetryPolicyDeleted the type of announcement emitted when we observe a deletion of retries.policy.flomesh.io
	RetryPolicyDeleted Kind = "retry-deleted"

	// RetryPolicyUpdated is the type of announcement emitted when we observe an update to retries.policy.flomesh.io
	RetryPolicyUpdated Kind = "retry-updated"

	// UpstreamTrafficSettingAdded is the type of announcement emitted when we observe an addition of upstreamtrafficsettings.policy.flomesh.io
	UpstreamTrafficSettingAdded Kind = "upstreamtrafficsetting-added"

	// UpstreamTrafficSettingDeleted is the type of announcement emitted when we observe a deletion of upstreamtrafficsettings.policy.flomesh.io
	UpstreamTrafficSettingDeleted Kind = "upstreamtrafficsetting-deleted"

	// UpstreamTrafficSettingUpdated is the type of announcement emitted when we observe an update of upstreamtrafficsettings.policy.flomesh.io
	UpstreamTrafficSettingUpdated Kind = "upstreamtrafficsetting-updated"

	// ---

	// PluginAdded is the type of announcement emitted when we observe an addition of plugins.plugin.flomesh.io
	PluginAdded Kind = "plugin-added"

	// PluginDeleted the type of announcement emitted when we observe a deletion of plugins.plugin.flomesh.io
	PluginDeleted Kind = "plugin-deleted"

	// PluginUpdated is the type of announcement emitted when we observe an update to plugins.plugin.flomesh.io
	PluginUpdated Kind = "plugin-updated"

	// PluginChainAdded is the type of announcement emitted when we observe an addition of pluginchains.plugin.flomesh.io
	PluginChainAdded Kind = "pluginchain-added"

	// PluginChainDeleted the type of announcement emitted when we observe a deletion of pluginchains.plugin.flomesh.io
	PluginChainDeleted Kind = "pluginchain-deleted"

	// PluginChainUpdated is the type of announcement emitted when we observe an update to pluginchains.plugin.flomesh.io
	PluginChainUpdated Kind = "pluginchain-updated"

	// PluginConfigAdded is the type of announcement emitted when we observe an addition of pluginconfigs.flomesh.io
	PluginConfigAdded Kind = "pluginconfig-added"

	// PluginConfigDeleted the type of announcement emitted when we observe a deletion of pluginconfigs.plugin.flomesh.io
	PluginConfigDeleted Kind = "pluginconfig-deleted"

	// PluginConfigUpdated is the type of announcement emitted when we observe an update to pluginconfigs.plugin.flomesh.io
	PluginConfigUpdated Kind = "pluginconfig-updated"

	// ---

	// EndpointSlicesAdded is the type of announcement emitted when we observe an addition of a Kubernetes EndpointSlices
	EndpointSlicesAdded Kind = "endpointslices-added"

	// EndpointSlicesDeleted the type of announcement emitted when we observe the deletion of a Kubernetes EndpointSlices
	EndpointSlicesDeleted Kind = "endpointslices-deleted"

	// EndpointSlicesUpdated is the type of announcement emitted when we observe an update to a Kubernetes EndpointSlices
	EndpointSlicesUpdated Kind = "endpointslices-updated"

	// ---

	// SecretAdded is the type of announcement emitted when we observe an addition of a Kubernetes Secret
	SecretAdded Kind = "secret-added"

	// SecretDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Secret
	SecretDeleted Kind = "secret-deleted"

	// SecretUpdated is the type of announcement emitted when we observe an update to a Kubernetes Secret
	SecretUpdated Kind = "secret-updated"

	// ---

	// GatewayAPIGatewayClassAdded is the type of announcement emitted when we observe an addition of gatewayclasses.gateway.networking.k8s.io
	GatewayAPIGatewayClassAdded Kind = "gwapi-gatewayclass-added"

	// GatewayAPIGatewayClassDeleted the type of announcement emitted when we observe a deletion of gatewayclasses.gateway.networking.k8s.io
	GatewayAPIGatewayClassDeleted Kind = "gwapi-gatewayclass-deleted"

	// GatewayAPIGatewayClassUpdated is the type of announcement emitted when we observe an update to gatewayclasses.gateway.networking.k8s.io
	GatewayAPIGatewayClassUpdated Kind = "gwapi-gatewayclass-updated"

	// ---

	// GatewayAPIGatewayAdded is the type of announcement emitted when we observe an addition of gateways.gateway.networking.k8s.io
	GatewayAPIGatewayAdded Kind = "gwapi-gateway-added"

	// GatewayAPIGatewayDeleted the type of announcement emitted when we observe a deletion of gateways.gateway.networking.k8s.io
	GatewayAPIGatewayDeleted Kind = "gwapi-gateway-deleted"

	// GatewayAPIGatewayUpdated is the type of announcement emitted when we observe an update to gateways.gateway.networking.k8s.io
	GatewayAPIGatewayUpdated Kind = "gwapi-gateway-updated"

	// ---

	// GatewayAPIHTTPRouteAdded is the type of announcement emitted when we observe an addition of httproutes.gateway.networking.k8s.io
	GatewayAPIHTTPRouteAdded Kind = "gwapi-httproute-added"

	// GatewayAPIHTTPRouteDeleted the type of announcement emitted when we observe a deletion of httproutes.gateway.networking.k8s.io
	GatewayAPIHTTPRouteDeleted Kind = "gwapi-httproute-deleted"

	// GatewayAPIHTTPRouteUpdated is the type of announcement emitted when we observe an update to httproutes.gateway.networking.k8s.io
	GatewayAPIHTTPRouteUpdated Kind = "gwapi-httproute-updated"

	// ---

	// GatewayAPIGRPCRouteAdded is the type of announcement emitted when we observe an addition of grpcroutes.gateway.networking.k8s.io
	GatewayAPIGRPCRouteAdded Kind = "gwapi-grpcroute-added"

	// GatewayAPIGRPCRouteDeleted the type of announcement emitted when we observe a deletion of grpcroutes.gateway.networking.k8s.io
	GatewayAPIGRPCRouteDeleted Kind = "gwapi-grpcroute-deleted"

	// GatewayAPIGRPCRouteUpdated is the type of announcement emitted when we observe an update to grpcroutes.gateway.networking.k8s.io
	GatewayAPIGRPCRouteUpdated Kind = "gwapi-grpcroute-updated"

	// ---

	// GatewayAPITLSRouteAdded is the type of announcement emitted when we observe an addition of tlsroutes.gateway.networking.k8s.io
	GatewayAPITLSRouteAdded Kind = "gwapi-tlsroute-added"

	// GatewayAPITLSRouteDeleted the type of announcement emitted when we observe a deletion of tlsroutes.gateway.networking.k8s.io
	GatewayAPITLSRouteDeleted Kind = "gwapi-tlsroute-deleted"

	// GatewayAPITLSRouteUpdated is the type of announcement emitted when we observe an update to tlsroutes.gateway.networking.k8s.io
	GatewayAPITLSRouteUpdated Kind = "gwapi-tlsroute-updated"

	// ---

	// GatewayAPITCPRouteAdded is the type of announcement emitted when we observe an addition of tcproutes.gateway.networking.k8s.io
	GatewayAPITCPRouteAdded Kind = "gwapi-tcproute-added"

	// GatewayAPITCPRouteDeleted the type of announcement emitted when we observe a deletion of tcproutes.gateway.networking.k8s.io
	GatewayAPITCPRouteDeleted Kind = "gwapi-tcproute-deleted"

	// GatewayAPITCPRouteUpdated is the type of announcement emitted when we observe an update to tcproutes.gateway.networking.k8s.io
	GatewayAPITCPRouteUpdated Kind = "gwapi-tcproute-updated"

	// ---

	// RateLimitPolicyAdded is the type of announcement emitted when we observe an addition of ratelimitpolicies.gateway.flomesh.io
	RateLimitPolicyAdded Kind = "ratelimitpolicy-added"

	// RateLimitPolicyDeleted the type of announcement emitted when we observe a deletion of ratelimitpolicies.gateway.flomesh.io
	RateLimitPolicyDeleted Kind = "ratelimitpolicy-deleted"

	// RateLimitPolicyUpdated is the type of announcement emitted when we observe an update to ratelimitpolicies.gateway.flomesh.io
	RateLimitPolicyUpdated Kind = "ratelimitpolicy-updated"

	// ---

	// SessionStickyPolicyAdded is the type of announcement emitted when we observe an addition of sessionstickypolicies.gateway.flomesh.io
	SessionStickyPolicyAdded Kind = "sessionstickypolicy-added"

	// SessionStickyPolicyDeleted the type of announcement emitted when we observe a deletion of sessionstickypolicies.gateway.flomesh.io
	SessionStickyPolicyDeleted Kind = "sessionstickypolicy-deleted"

	// SessionStickyPolicyUpdated is the type of announcement emitted when we observe an update to sessionstickypolicies.gateway.flomesh.io
	SessionStickyPolicyUpdated Kind = "sessionstickypolicy-updated"

	// ---

	// LoadBalancerPolicyAdded is the type of announcement emitted when we observe an addition of loadbalancerpolicies.gateway.flomesh.io
	LoadBalancerPolicyAdded Kind = "loadbalancerpolicy-added"

	// LoadBalancerPolicyDeleted the type of announcement emitted when we observe a deletion of loadbalancerpolicies.gateway.flomesh.io
	LoadBalancerPolicyDeleted Kind = "loadbalancerpolicy-deleted"

	// LoadBalancerPolicyUpdated is the type of announcement emitted when we observe an update to loadbalancerpolicies.gateway.flomesh.io
	LoadBalancerPolicyUpdated Kind = "loadbalancerpolicy-updated"

	// ---

	// CircuitBreakingPolicyAdded is the type of announcement emitted when we observe an addition of circuitbreakingpolicies.gateway.flomesh.io
	CircuitBreakingPolicyAdded Kind = "circuitbreaking-added"

	// CircuitBreakingPolicyDeleted the type of announcement emitted when we observe a deletion of circuitbreakingpolicies.gateway.flomesh.io
	CircuitBreakingPolicyDeleted Kind = "circuitbreaking-deleted"

	// CircuitBreakingPolicyUpdated is the type of announcement emitted when we observe an update to circuitbreakingpolicies.gateway.flomesh.io
	CircuitBreakingPolicyUpdated Kind = "circuitbreaking-updated"

	// ---

	// AccessControlPolicyAdded is the type of announcement emitted when we observe an addition of accesscontrolpolicies.gateway.flomesh.io
	AccessControlPolicyAdded Kind = "accesscontrolpolicy-added"

	// AccessControlPolicyDeleted the type of announcement emitted when we observe a deletion of accesscontrolpolicies.gateway.flomesh.io
	AccessControlPolicyDeleted Kind = "accesscontrolpolicy-deleted"

	// AccessControlPolicyUpdated is the type of announcement emitted when we observe an update to accesscontrolpolicies.gateway.flomesh.io
	AccessControlPolicyUpdated Kind = "accesscontrolpolicy-updated"

	// ---

	// HealthCheckPolicyAdded is the type of announcement emitted when we observe an addition of healthcheckpolicies.gateway.flomesh.io
	HealthCheckPolicyAdded Kind = "healthcheckpolicy-added"

	// HealthCheckPolicyDeleted the type of announcement emitted when we observe a deletion of healthcheckpolicies.gateway.flomesh.io
	HealthCheckPolicyDeleted Kind = "healthcheckpolicy-deleted"

	// HealthCheckPolicyUpdated is the type of announcement emitted when we observe an update to healthcheckpolicies.gateway.flomesh.io
	HealthCheckPolicyUpdated Kind = "healthcheckpolicy-updated"

	// ---

	// FaultInjectionPolicyAdded is the type of announcement emitted when we observe an addition of faultinjectionpolicies.gateway.flomesh.io
	FaultInjectionPolicyAdded Kind = "faultinjectionpolicy-added"

	// FaultInjectionPolicyDeleted the type of announcement emitted when we observe a deletion of faultinjectionpolicies.gateway.flomesh.io
	FaultInjectionPolicyDeleted Kind = "faultinjectionpolicy-deleted"

	// FaultInjectionPolicyUpdated is the type of announcement emitted when we observe an update to faultinjectionpolicies.gateway.flomesh.io
	FaultInjectionPolicyUpdated Kind = "faultinjectionpolicy-updated"

	// ---

	// UpstreamTLSPolicyAdded is the type of announcement emitted when we observe an addition of upstreamtlspolicies.gateway.flomesh.io
	UpstreamTLSPolicyAdded Kind = "upstreamtlspolicy-added"

	// UpstreamTLSPolicyDeleted the type of announcement emitted when we observe a deletion of upstreamtlspolicies.gateway.flomesh.io
	UpstreamTLSPolicyDeleted Kind = "upstreamtlspolicy-deleted"

	// UpstreamTLSPolicyUpdated is the type of announcement emitted when we observe an update to upstreamtlspolicies.gateway.flomesh.io
	UpstreamTLSPolicyUpdated Kind = "upstreamtlspolicy-updated"
)

// Announcement is a struct for messages between various components of FSM signaling a need for a change in Sidecar proxy configuration
type Announcement struct {
	Type               Kind
	ReferencedObjectID interface{}
}
