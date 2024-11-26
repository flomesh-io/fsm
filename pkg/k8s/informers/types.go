package informers

import (
	"errors"
	"time"

	"k8s.io/client-go/tools/cache"
)

// InformerKey stores the different Informers we keep for K8s resources
type InformerKey string

const (
	// InformerKeyNamespace is the InformerKey for a Namespace informer
	InformerKeyNamespace InformerKey = "Namespace"
	// InformerKeyNamespaceAll is the InformerKey for all Namespaces informer
	InformerKeyNamespaceAll InformerKey = "NamespaceAll"
	// InformerKeyService is the InformerKey for a Service informer
	InformerKeyService InformerKey = "Service"
	// InformerKeyPod is the InformerKey for a Pod informer
	InformerKeyPod InformerKey = "Pod"
	// InformerKeyEndpoints is the InformerKey for a Endpoints informer
	InformerKeyEndpoints InformerKey = "Endpoints"
	// InformerKeyEndpointSlices is the InformerKey for a EndpointSlices informer
	InformerKeyEndpointSlices InformerKey = "EndpointSlices"
	// InformerKeyServiceAccount is the InformerKey for a ServiceAccount informer
	InformerKeyServiceAccount InformerKey = "ServiceAccount"
	// InformerKeySecret is the InformerKey for a Secret informer
	InformerKeySecret InformerKey = "Secret"
	// InformerKeyConfigMap is the InformerKey for a ConfigMap informer
	InformerKeyConfigMap InformerKey = "ConfigMap"

	// InformerKeyTrafficSplit is the InformerKey for a TrafficSplit informer
	InformerKeyTrafficSplit InformerKey = "TrafficSplit"
	// InformerKeyTrafficTarget is the InformerKey for a TrafficTarget informer
	InformerKeyTrafficTarget InformerKey = "TrafficTarget"
	// InformerKeyHTTPRouteGroup is the InformerKey for a HTTPRouteGroup informer
	InformerKeyHTTPRouteGroup InformerKey = "HTTPRouteGroup"
	// InformerKeyTCPRoute is the InformerKey for a TCPRoute informer
	InformerKeyTCPRoute InformerKey = "TCPRoute"

	// InformerKeyMeshConfig is the InformerKey for a MeshConfig informer
	InformerKeyMeshConfig InformerKey = "MeshConfig"
	// InformerKeyMeshRootCertificate is the InformerKey for a MeshRootCertificate informer
	InformerKeyMeshRootCertificate InformerKey = "MeshRootCertificate"

	// InformerKeyIsolation is the InformerKey for a Isolation informer
	InformerKeyIsolation InformerKey = "Isolation"
	// InformerKeyEgress is the InformerKey for an Egress informer
	InformerKeyEgress InformerKey = "Egress"
	// InformerKeyEgressGateway is the InformerKey for an EgressGateway informer
	InformerKeyEgressGateway InformerKey = "EgressGateway"
	// InformerKeyIngressBackend is the InformerKey for a IngressBackend informer
	InformerKeyIngressBackend InformerKey = "IngressBackend"
	// InformerKeyUpstreamTrafficSetting is the InformerKey for a UpstreamTrafficSetting informer
	InformerKeyUpstreamTrafficSetting InformerKey = "UpstreamTrafficSetting"
	// InformerKeyRetry is the InformerKey for a Retry informer
	InformerKeyRetry InformerKey = "Retry"
	// InformerKeyAccessControl is the InformerKey for a AccessControl informer
	InformerKeyAccessControl InformerKey = "AccessControl"
	// InformerKeyAccessCert is the InformerKey for a AccessCert informer
	InformerKeyAccessCert InformerKey = "AccessCert"
	// InformerKeyServiceImport is the InformerKey for a ServiceImport informer
	InformerKeyServiceImport InformerKey = "ServiceImport"
	// InformerKeyServiceExport is the InformerKey for a ServiceExport informer
	InformerKeyServiceExport InformerKey = "ServiceExport"
	// InformerKeyGlobalTrafficPolicy is the InformerKey for a GlobalTrafficPolicy informer
	InformerKeyGlobalTrafficPolicy InformerKey = "GlobalTrafficPolicy"
	// InformerKeyIngressClass is the InformerKey for a IngressClass informer
	InformerKeyIngressClass InformerKey = "IngressClass"

	// InformerKeyPlugin is the InformerKey for a Plugin informer
	InformerKeyPlugin InformerKey = "Plugin"
	// InformerKeyPluginChain is the InformerKey for a PluginChain informer
	InformerKeyPluginChain InformerKey = "PluginChain"
	// InformerKeyPluginConfig is the InformerKey for a PluginConfig informer
	InformerKeyPluginConfig InformerKey = "PluginConfig"

	// InformerKeyVirtualMachine is the InformerKey for a VirtualMachine informer
	InformerKeyVirtualMachine InformerKey = "VirtualMachine"

	// InformerKeyConsulConnector is the InformerKey for a ConsulConnector informer
	InformerKeyConsulConnector InformerKey = "ConsulConnector"

	// InformerKeyEurekaConnector is the InformerKey for a EurekaConnector informer
	InformerKeyEurekaConnector InformerKey = "EurekaConnector"

	// InformerKeyNacosConnector is the InformerKey for a NacosConnector informer
	InformerKeyNacosConnector InformerKey = "NacosConnector"

	// InformerKeyMachineConnector is the InformerKey for a MachineConnector informer
	InformerKeyMachineConnector InformerKey = "MachineConnector"

	// InformerKeyGatewayConnector is the InformerKey for a GatewayConnector informer
	InformerKeyGatewayConnector InformerKey = "GatewayConnector"

	// InformerKeyK8sIngressClass is the InformerKey for a k8s IngressClass informer
	InformerKeyK8sIngressClass InformerKey = "IngressClass-k8s"
	// InformerKeyK8sIngress is the InformerKey for a k8s Ingress informer
	InformerKeyK8sIngress InformerKey = "Ingress-k8s"

	// InformerKeyNamespacedIngress is the InformerKey for a NamespacedIngress informer
	InformerKeyNamespacedIngress InformerKey = "NamespacedIngress"

	// InformerKeyGatewayAPIGatewayClass is the InformerKey for a GatewayClass informer
	InformerKeyGatewayAPIGatewayClass InformerKey = "GatewayClass-gwapi"
	// InformerKeyGatewayAPIGateway is the InformerKey for a Gateway informer
	InformerKeyGatewayAPIGateway InformerKey = "Gateway-gwapi"
	// InformerKeyGatewayAPIHTTPRoute is the InformerKey for a HTTPRoute informer
	InformerKeyGatewayAPIHTTPRoute InformerKey = "HTTPRoute-gwapi"
	// InformerKeyGatewayAPIGRPCRoute is the InformerKey for a GRPCRoute informer
	InformerKeyGatewayAPIGRPCRoute InformerKey = "GRPCRoute-gwapi"
	// InformerKeyGatewayAPITLSRoute is the InformerKey for a TLSRoute informer
	InformerKeyGatewayAPITLSRoute InformerKey = "TLSRoute-gwapi"
	// InformerKeyGatewayAPITCPRoute is the InformerKey for a TCPRoute informer
	InformerKeyGatewayAPITCPRoute InformerKey = "TCPRoute-gwapi"
	// InformerKeyGatewayAPIUDPRoute is the InformerKey for a UDPRoute informer
	InformerKeyGatewayAPIUDPRoute InformerKey = "UDPRoute-gwapi"
	// InformerKeyGatewayAPIReferenceGrant is the InformerKey for a ReferenceGrant informer
	InformerKeyGatewayAPIReferenceGrant InformerKey = "ReferenceGrant-gwapi"
	// InformerKeyRateLimit is the InformerKey for a RateLimit extension informer
	InformerKeyRateLimit InformerKey = "RateLimit"
	// InformerKeyCircuitBreaker is the InformerKey for a CircuitBreaker informer
	InformerKeyCircuitBreaker InformerKey = "CircuitBreaker"
	// InformerKeyFaultInjection is the InformerKey for a FaultInjection informer
	InformerKeyFaultInjection InformerKey = "FaultInjection"
	// InformerKeyBackendTLSPolicy is the InformerKey for a BackendTLSPolicy informer
	InformerKeyBackendTLSPolicy InformerKey = "BackendTLSPolicy"
	// InformerKeyBackendLBPolicy is the InformerKey for a BackendLBPolicy informer
	InformerKeyBackendLBPolicy InformerKey = "BackendLBPolicy"
	// InformerKeyHealthCheckPolicyV1alpha2 is the InformerKey for a HealthCheckPolicy informer
	InformerKeyHealthCheckPolicyV1alpha2 InformerKey = "HealthCheckPolicy-v1alpha2"
	// InformerKeyFilter is the InformerKey for a Filter informer
	InformerKeyFilter InformerKey = "Filter"
	// InformerKeyListenerFilter is the InformerKey for a ListenerFilter informer
	InformerKeyListenerFilter InformerKey = "ListenerFilter"
	// InformerKeyFilterDefinition is the InformerKey for a FilterDefinition informer
	InformerKeyFilterDefinition InformerKey = "FilterDefinition"
	// InformerKeyFilterConfig is the InformerKey for a FilterConfig informer
	InformerKeyFilterConfig InformerKey = "FilterConfig"
	// InformerKeyGatewayHTTPLog is the InformerKey for a HTTPLog informer
	InformerKeyGatewayHTTPLog InformerKey = "Gateway-HTTPLog"
	// InformerKeyGatewayMetrics is the InformerKey for a Metrics informer
	InformerKeyGatewayMetrics InformerKey = "Gateway-Metrics"
	// InformerKeyGatewayZipkin is the InformerKey for a Zipkin informer
	InformerKeyGatewayZipkin InformerKey = "Gateway-Zipkin"
	// InformerKeyGatewayProxyTag is the InformerKey for a ProxyTag informer
	InformerKeyGatewayProxyTag InformerKey = "Gateway-ProxyTag"

	// InformerKeyXNetworkAccessControl is the InformerKey for a XNetwork AccessControl informer
	InformerKeyXNetworkAccessControl InformerKey = "XNetwork-AccessControl"
)

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client, and have our
	// own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
)

var (
	errInitInformers = errors.New("informer not initialized")
	errSyncingCaches = errors.New("failed initial cache sync for informers")
)

// InformerCollection is an abstraction around a set of informers
// initialized with the clients stored in its fields. This data
// type should only be passed around as a pointer
type InformerCollection struct {
	informers map[InformerKey]cache.SharedIndexInformer
	//listers   *Lister
	meshName string
}

// ResourceType is the type used to represent the type of resource
type ResourceType string

const (
	// ServicesResourceType is the type used to represent the services resource
	ServicesResourceType ResourceType = "services"

	// EndpointSlicesResourceType is the type used to represent the endpoint slices resource
	EndpointSlicesResourceType ResourceType = "endpointslices"

	// EndpointsResourceType is the type used to represent the endpoints resource
	EndpointsResourceType ResourceType = "endpoints"

	// ServiceImportsResourceType is the type used to represent the service imports resource
	ServiceImportsResourceType ResourceType = "serviceimports"

	// SecretsResourceType is the type used to represent the secrets resource
	SecretsResourceType ResourceType = "secrets"

	// ConfigMapsResourceType is the type used to represent the config maps resource
	ConfigMapsResourceType ResourceType = "configmaps"

	// GatewayClassesResourceType is the type used to represent the gateway classes resource
	GatewayClassesResourceType ResourceType = "gatewayclasses"

	// GatewaysResourceType is the type used to represent the gateways resource
	GatewaysResourceType ResourceType = "gateways"

	// HTTPRoutesResourceType is the type used to represent the HTTP routes resource
	HTTPRoutesResourceType ResourceType = "httproutes"

	// GRPCRoutesResourceType is the type used to represent the gRPC routes resource
	GRPCRoutesResourceType ResourceType = "grpcroutes"

	// TCPRoutesResourceType is the type used to represent the TCP routes resource
	TCPRoutesResourceType ResourceType = "tcproutes"

	// TLSRoutesResourceType is the type used to represent the TLS routes resource
	TLSRoutesResourceType ResourceType = "tlsroutes"

	// UDPRoutesResourceType is the type used to represent the UDP routes resource
	UDPRoutesResourceType ResourceType = "udproutes"

	// ReferenceGrantResourceType is the type used to represent the reference grants resource
	ReferenceGrantResourceType ResourceType = "referencegrants"

	// RateLimitsResourceType is the type used to represent the rate limit resource
	RateLimitsResourceType ResourceType = "ratelimits"

	// CircuitBreakersResourceType is the type used to represent the circuit breakers  resource
	CircuitBreakersResourceType ResourceType = "circuitbreakers"

	// AccessControlPoliciesResourceType is the type used to represent the access control policies resource
	AccessControlPoliciesResourceType ResourceType = "accesscontrols"

	// HealthCheckPoliciesResourceType is the type used to represent the health check policies resource
	HealthCheckPoliciesResourceType ResourceType = "healthchecks"

	// FaultInjectionsResourceType is the type used to represent the fault injections resource
	FaultInjectionsResourceType ResourceType = "faultinjections"

	// BackendTLSPoliciesResourceType is the type used to represent the backend tls policies resource
	BackendTLSPoliciesResourceType ResourceType = "backendtls"

	// BackendLBPoliciesResourceType is the type used to represent the backend lb policies resource
	BackendLBPoliciesResourceType ResourceType = "backendlbs"

	// FiltersResourceType is the type used to represent the filters resource
	FiltersResourceType ResourceType = "filters"

	// ListenerFiltersResourceType is the type used to represent the listener filters resource
	ListenerFiltersResourceType ResourceType = "listenerfilters"

	// FilterDefinitionsResourceType is the type used to represent the filter definitions resource
	FilterDefinitionsResourceType ResourceType = "filterdefinitions"

	// FilterConfigsResourceType is the type used to represent the filter configs resource
	FilterConfigsResourceType ResourceType = "filterconfigs"

	// HTTPLogsResourceType is the type used to represent the http logs resource
	HTTPLogsResourceType ResourceType = "httplogs"

	// MetricsResourceType is the type used to represent the metrics resource
	MetricsResourceType ResourceType = "metrics"

	// ZipkinResourceType is the type used to represent the zipkin tracing resource
	ZipkinResourceType ResourceType = "zipkins"

	// ProxyTagResourceType is the type used to represent the proxy tag resource
	ProxyTagResourceType ResourceType = "proxytags"
)
