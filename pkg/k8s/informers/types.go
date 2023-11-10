package informers

import (
	"errors"
	"time"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/listers/policyattachment/v1alpha1"

	v1 "k8s.io/client-go/listers/core/v1"
	discoveryv1 "k8s.io/client-go/listers/discovery/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/listers/multicluster/v1alpha1"
	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/listers/namespacedingress/v1alpha1"

	"k8s.io/client-go/tools/cache"
)

// InformerKey stores the different Informers we keep for K8s resources
type InformerKey string

const (
	// InformerKeyNamespace is the InformerKey for a Namespace informer
	InformerKeyNamespace InformerKey = "Namespace"
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
	// InformerKeyRateLimitPolicy is the InformerKey for a RateLimitPolicy informer
	InformerKeyRateLimitPolicy InformerKey = "RateLimitPolicy"
	// InformerKeySessionStickyPolicy is the InformerKey for a SessionStickyPolicy informer
	InformerKeySessionStickyPolicy InformerKey = "SessionStickyPolicy"
	// InformerKeyLoadBalancerPolicy is the InformerKey for a LoadBalancerPolicy informer
	InformerKeyLoadBalancerPolicy InformerKey = "LoadBalancerPolicy"
	// InformerKeyCircuitBreakingPolicy is the InformerKey for a CircuitBreakingPolicy informer
	InformerKeyCircuitBreakingPolicy InformerKey = "CircuitBreakingPolicy"
	// InformerKeyAccessControlPolicy is the InformerKey for a AccessControlPolicy informer
	InformerKeyAccessControlPolicy InformerKey = "AccessControlPolicy"
	// InformerKeyHealthCheckPolicy is the InformerKey for a HealthCheckPolicy informer
	InformerKeyHealthCheckPolicy InformerKey = "HealthCheckPolicy"
	// InformerKeyFaultInjectionPolicy is the InformerKey for a FaultInjectionPolicy informer
	InformerKeyFaultInjectionPolicy InformerKey = "FaultInjectionPolicy"
	// InformerKeyUpstreamTLSPolicy is the InformerKey for a UpstreamTLSPolicy informer
	InformerKeyUpstreamTLSPolicy InformerKey = "UpstreamTLSPolicy"
	// InformerKeyRetryPolicy is the InformerKey for a RetryPolicy informer
	InformerKeyRetryPolicy InformerKey = "RetryPolicy"
	// InformerKeyGatewayTLSPolicy is the InformerKey for a GatewayTLSPolicy informer
	InformerKeyGatewayTLSPolicy InformerKey = "GatewayTLSPolicy"
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
	listers   *Lister
	meshName  string
}

// Lister is the listers for the informers in the collection
type Lister struct {
	Service               v1.ServiceLister
	ServiceImport         mcsv1alpha1.ServiceImportLister
	Endpoints             v1.EndpointsLister
	EndpointSlice         discoveryv1.EndpointSliceLister
	Secret                v1.SecretLister
	GatewayClass          gwv1beta1.GatewayClassLister
	Gateway               gwv1beta1.GatewayLister
	HTTPRoute             gwv1beta1.HTTPRouteLister
	GRPCRoute             gwv1alpha2.GRPCRouteLister
	TLSRoute              gwv1alpha2.TLSRouteLister
	TCPRoute              gwv1alpha2.TCPRouteLister
	K8sIngressClass       networkingv1.IngressClassLister
	K8sIngress            networkingv1.IngressLister
	NamespacedIngress     nsigv1alpha1.NamespacedIngressLister
	RateLimitPolicy       gwpav1alpha1.RateLimitPolicyLister
	SessionStickyPolicy   gwpav1alpha1.SessionStickyPolicyLister
	LoadBalancerPolicy    gwpav1alpha1.LoadBalancerPolicyLister
	CircuitBreakingPolicy gwpav1alpha1.CircuitBreakingPolicyLister
	AccessControlPolicy   gwpav1alpha1.AccessControlPolicyLister
	HealthCheckPolicy     gwpav1alpha1.HealthCheckPolicyLister
	FaultInjectionPolicy  gwpav1alpha1.FaultInjectionPolicyLister
	UpstreamTLSPolicy     gwpav1alpha1.UpstreamTLSPolicyLister
	RetryPolicy           gwpav1alpha1.RetryPolicyLister
	GatewayTLSPolicy      gwpav1alpha1.GatewayTLSPolicyLister
}
