package gateway

import (
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/k8s"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
)

//gocyclo:ignore
func getEventTypesByObjectType(obj interface{}) *k8s.EventTypes {
	switch obj.(type) {
	case *corev1.Service:
		return getEventTypesByInformerKey(fsminformers.InformerKeyService)
	case *mcsv1alpha1.ServiceImport:
		return getEventTypesByInformerKey(fsminformers.InformerKeyServiceImport)
	case *corev1.Endpoints:
		return getEventTypesByInformerKey(fsminformers.InformerKeyEndpoints)
	case *discoveryv1.EndpointSlice:
		return getEventTypesByInformerKey(fsminformers.InformerKeyEndpointSlices)
	case *corev1.Secret:
		return getEventTypesByInformerKey(fsminformers.InformerKeySecret)
	case *corev1.ConfigMap:
		return getEventTypesByInformerKey(fsminformers.InformerKeyConfigMap)
	case *gwv1.GatewayClass:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIGatewayClass)
	case *gwv1.Gateway:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIGateway)
	case *gwv1.HTTPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIHTTPRoute)
	case *gwv1.GRPCRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIGRPCRoute)
	case *gwv1alpha2.TLSRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPITLSRoute)
	case *gwv1alpha2.TCPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPITCPRoute)
	case *gwv1alpha2.UDPRoute:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIUDPRoute)
	case *gwv1beta1.ReferenceGrant:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayAPIReferenceGrant)
	case *gwpav1alpha2.BackendLBPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyBackendLBPolicy)
	case *gwv1alpha3.BackendTLSPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyBackendTLSPolicy)
	case *gwpav1alpha2.HealthCheckPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyHealthCheckPolicyV1alpha2)
	case *gwpav1alpha2.RouteRuleFilterPolicy:
		return getEventTypesByInformerKey(fsminformers.InformerKeyRouteRuleFilterPolicy)
	case *extv1alpha1.Filter:
		return getEventTypesByInformerKey(fsminformers.InformerKeyFilter)
	case *extv1alpha1.FilterDefinition:
		return getEventTypesByInformerKey(fsminformers.InformerKeyFilterDefinition)
	case *extv1alpha1.ListenerFilter:
		return getEventTypesByInformerKey(fsminformers.InformerKeyListenerFilter)
	case *extv1alpha1.CircuitBreaker:
		return getEventTypesByInformerKey(fsminformers.InformerKeyCircuitBreaker)
	case *extv1alpha1.FaultInjection:
		return getEventTypesByInformerKey(fsminformers.InformerKeyFaultInjection)
	case *extv1alpha1.RateLimit:
		return getEventTypesByInformerKey(fsminformers.InformerKeyRateLimit)
	case *extv1alpha1.HTTPLog:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayHTTPLog)
	case *extv1alpha1.Metrics:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayMetrics)
	case *extv1alpha1.Zipkin:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayZipkin)
	case *extv1alpha1.FilterConfig:
		return getEventTypesByInformerKey(fsminformers.InformerKeyFilterConfig)
	case *extv1alpha1.ProxyTag:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayProxyTag)
	case *extv1alpha1.ExternalRateLimit:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayExternalRateLimit)
	case *extv1alpha1.IPRestriction:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayIPRestriction)
	case *extv1alpha1.RequestTermination:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayRequestTermination)
	case *extv1alpha1.ConcurrencyLimit:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayConcurrencyLimit)
	case *extv1alpha1.DNSModifier:
		return getEventTypesByInformerKey(fsminformers.InformerKeyGatewayDNSModifier)
	}

	return nil
}

//gocyclo:ignore
func getEventTypesByInformerKey(informerKey fsminformers.InformerKey) *k8s.EventTypes {
	switch informerKey {
	case fsminformers.InformerKeyService:
		return &k8s.EventTypes{
			Add:    announcements.ServiceAdded,
			Update: announcements.ServiceUpdated,
			Delete: announcements.ServiceDeleted,
		}
	case fsminformers.InformerKeyServiceImport:
		return &k8s.EventTypes{
			Add:    announcements.ServiceImportAdded,
			Update: announcements.ServiceImportUpdated,
			Delete: announcements.ServiceImportDeleted,
		}
	case fsminformers.InformerKeyEndpointSlices:
		return &k8s.EventTypes{
			Add:    announcements.EndpointSlicesAdded,
			Update: announcements.EndpointSlicesUpdated,
			Delete: announcements.EndpointSlicesDeleted,
		}
	case fsminformers.InformerKeyEndpoints:
		return &k8s.EventTypes{
			Add:    announcements.EndpointAdded,
			Update: announcements.EndpointUpdated,
			Delete: announcements.EndpointDeleted,
		}
	case fsminformers.InformerKeySecret:
		return &k8s.EventTypes{
			Add:    announcements.SecretAdded,
			Update: announcements.SecretUpdated,
			Delete: announcements.SecretDeleted,
		}
	case fsminformers.InformerKeyConfigMap:
		return &k8s.EventTypes{
			Add:    announcements.ConfigMapAdded,
			Update: announcements.ConfigMapUpdated,
			Delete: announcements.ConfigMapDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIGatewayClass:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIGatewayClassAdded,
			Update: announcements.GatewayAPIGatewayClassUpdated,
			Delete: announcements.GatewayAPIGatewayClassDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIGateway:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIGatewayAdded,
			Update: announcements.GatewayAPIGatewayUpdated,
			Delete: announcements.GatewayAPIGatewayDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIHTTPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIHTTPRouteAdded,
			Update: announcements.GatewayAPIHTTPRouteUpdated,
			Delete: announcements.GatewayAPIHTTPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIGRPCRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIGRPCRouteAdded,
			Update: announcements.GatewayAPIGRPCRouteUpdated,
			Delete: announcements.GatewayAPIGRPCRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPITLSRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPITLSRouteAdded,
			Update: announcements.GatewayAPITLSRouteUpdated,
			Delete: announcements.GatewayAPITLSRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPITCPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPITCPRouteAdded,
			Update: announcements.GatewayAPITCPRouteUpdated,
			Delete: announcements.GatewayAPITCPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIUDPRoute:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIUDPRouteAdded,
			Update: announcements.GatewayAPIUDPRouteUpdated,
			Delete: announcements.GatewayAPIUDPRouteDeleted,
		}
	case fsminformers.InformerKeyGatewayAPIReferenceGrant:
		return &k8s.EventTypes{
			Add:    announcements.GatewayAPIReferenceGrantAdded,
			Update: announcements.GatewayAPIReferenceGrantUpdated,
			Delete: announcements.GatewayAPIReferenceGrantDeleted,
		}

	case fsminformers.InformerKeyBackendLBPolicy:
		return &k8s.EventTypes{
			Add:    announcements.BackendLBPolicyAdded,
			Update: announcements.BackendLBPolicyUpdated,
			Delete: announcements.BackendLBPolicyDeleted,
		}
	case fsminformers.InformerKeyBackendTLSPolicy:
		return &k8s.EventTypes{
			Add:    announcements.BackendTLSPolicyAdded,
			Update: announcements.BackendTLSPolicyUpdated,
			Delete: announcements.BackendTLSPolicyDeleted,
		}
	case fsminformers.InformerKeyHealthCheckPolicyV1alpha2:
		return &k8s.EventTypes{
			Add:    announcements.HealthCheckPolicyAdded,
			Update: announcements.HealthCheckPolicyUpdated,
			Delete: announcements.HealthCheckPolicyDeleted,
		}
	case fsminformers.InformerKeyRouteRuleFilterPolicy:
		return &k8s.EventTypes{
			Add:    announcements.RouteRuleFilterPolicyAdded,
			Update: announcements.RouteRuleFilterPolicyUpdated,
			Delete: announcements.RouteRuleFilterPolicyDeleted,
		}
	case fsminformers.InformerKeyFilter:
		return &k8s.EventTypes{
			Add:    announcements.FilterAdded,
			Update: announcements.FilterUpdated,
			Delete: announcements.FilterDeleted,
		}
	case fsminformers.InformerKeyFilterDefinition:
		return &k8s.EventTypes{
			Add:    announcements.FilterDefinitionAdded,
			Update: announcements.FilterDefinitionUpdated,
			Delete: announcements.FilterDefinitionDeleted,
		}
	case fsminformers.InformerKeyListenerFilter:
		return &k8s.EventTypes{
			Add:    announcements.ListenerFilterAdded,
			Update: announcements.ListenerFilterUpdated,
			Delete: announcements.ListenerFilterDeleted,
		}
	case fsminformers.InformerKeyCircuitBreaker:
		return &k8s.EventTypes{
			Add:    announcements.CircuitBreakerAdded,
			Update: announcements.CircuitBreakerUpdated,
			Delete: announcements.CircuitBreakerDeleted,
		}
	case fsminformers.InformerKeyFaultInjection:
		return &k8s.EventTypes{
			Add:    announcements.FaultInjectionAdded,
			Update: announcements.FaultInjectionUpdated,
			Delete: announcements.FaultInjectionDeleted,
		}
	case fsminformers.InformerKeyRateLimit:
		return &k8s.EventTypes{
			Add:    announcements.RateLimitAdded,
			Update: announcements.RateLimitUpdated,
			Delete: announcements.RateLimitDeleted,
		}
	case fsminformers.InformerKeyGatewayHTTPLog:
		return &k8s.EventTypes{
			Add:    announcements.GatewayHTTPLogAdded,
			Update: announcements.GatewayHTTPLogUpdated,
			Delete: announcements.GatewayHTTPLogDeleted,
		}
	case fsminformers.InformerKeyGatewayMetrics:
		return &k8s.EventTypes{
			Add:    announcements.GatewayMetricsAdded,
			Update: announcements.GatewayMetricsUpdated,
			Delete: announcements.GatewayMetricsDeleted,
		}
	case fsminformers.InformerKeyGatewayZipkin:
		return &k8s.EventTypes{
			Add:    announcements.GatewayZipkinAdded,
			Update: announcements.GatewayZipkinUpdated,
			Delete: announcements.GatewayZipkinDeleted,
		}
	case fsminformers.InformerKeyFilterConfig:
		return &k8s.EventTypes{
			Add:    announcements.FilterConfigAdded,
			Update: announcements.FilterConfigUpdated,
			Delete: announcements.FilterConfigDeleted,
		}
	case fsminformers.InformerKeyGatewayProxyTag:
		return &k8s.EventTypes{
			Add:    announcements.GatewayProxyTagAdded,
			Update: announcements.GatewayProxyTagUpdated,
			Delete: announcements.GatewayProxyTagDeleted,
		}
	case fsminformers.InformerKeyGatewayExternalRateLimit:
		return &k8s.EventTypes{
			Add:    announcements.GatewayExternalRateLimitAdded,
			Update: announcements.GatewayExternalRateLimitUpdated,
			Delete: announcements.GatewayExternalRateLimitDeleted,
		}
	case fsminformers.InformerKeyGatewayIPRestriction:
		return &k8s.EventTypes{
			Add:    announcements.GatewayIPRestrictionAdded,
			Update: announcements.GatewayIPRestrictionUpdated,
			Delete: announcements.GatewayIPRestrictionDeleted,
		}
	case fsminformers.InformerKeyGatewayRequestTermination:
		return &k8s.EventTypes{
			Add:    announcements.GatewayRequestTerminationAdded,
			Update: announcements.GatewayRequestTerminationUpdated,
			Delete: announcements.GatewayRequestTerminationDeleted,
		}
	case fsminformers.InformerKeyGatewayConcurrencyLimit:
		return &k8s.EventTypes{
			Add:    announcements.GatewayConcurrencyLimitAdded,
			Update: announcements.GatewayConcurrencyLimitUpdated,
			Delete: announcements.GatewayConcurrencyLimitDeleted,
		}
	case fsminformers.InformerKeyGatewayDNSModifier:
		return &k8s.EventTypes{
			Add:    announcements.GatewayDNSModifierAdded,
			Update: announcements.GatewayDNSModifierUpdated,
			Delete: announcements.GatewayDNSModifierDeleted,
		}
	}

	return nil
}
