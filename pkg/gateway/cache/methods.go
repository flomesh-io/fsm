package cache

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) getActiveGateways() []*gwv1.Gateway {
	gateways := make([]*gwv1.Gateway, 0)

	allGateways, err := c.informers.GetListers().Gateway.Gateways(corev1.NamespaceAll).List(selectAll)
	if err != nil {
		return nil
	}

	for _, gw := range allGateways {
		if gwutils.IsActiveGateway(gw) {
			gw.GetObjectKind().SetGroupVersionKind(constants.GatewayGVK)
			gateways = append(gateways, gw)
		}
	}

	return gateways
}

//gocyclo:ignore
func (c *GatewayCache) getResourcesFromCache(resourceType ResourceType, shouldSort bool) []client.Object {
	resources := make([]client.Object, 0)

	switch resourceType {
	case HTTPRoutesResourceType:
		routes, err := c.informers.GetListers().HTTPRoute.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get HTTPRoutes: %s", err)
			return resources
		}
		resources = setGroupVersionKind(routes, constants.HTTPRouteGVK)
	case GRPCRoutesResourceType:
		routes, err := c.informers.GetListers().GRPCRoute.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get GRPCRouts: %s", err)
			return resources
		}
		resources = setGroupVersionKind(routes, constants.GRPCRouteGVK)
	case TLSRoutesResourceType:
		routes, err := c.informers.GetListers().TLSRoute.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get TLSRoutes: %s", err)
			return resources
		}
		resources = setGroupVersionKind(routes, constants.TLSRouteGVK)
	case TCPRoutesResourceType:
		routes, err := c.informers.GetListers().TCPRoute.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get TCPRoutes: %s", err)
			return resources
		}
		resources = setGroupVersionKind(routes, constants.TCPRouteGVK)
	case UDPRoutesResourceType:
		routes, err := c.informers.GetListers().UDPRoute.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get UDPRoutes: %s", err)
			return resources
		}
		resources = setGroupVersionKind(routes, constants.UDPRouteGVK)
	case ReferenceGrantResourceType:
		grants, err := c.informers.GetListers().ReferenceGrant.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get ReferenceGrants: %s", err)
			return resources
		}
		resources = setGroupVersionKind(grants, constants.ReferenceGrantGVK)
	case UpstreamTLSPoliciesResourceType:
		policies, err := c.informers.GetListers().UpstreamTLSPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get UpstreamTLSPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.UpstreamTLSPolicyGVK)
	case RateLimitPoliciesResourceType:
		policies, err := c.informers.GetListers().RateLimitPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get RateLimitPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.RateLimitPolicyGVK)
	case AccessControlPoliciesResourceType:
		policies, err := c.informers.GetListers().AccessControlPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get AccessControlPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.AccessControlPolicyGVK)
	case FaultInjectionPoliciesResourceType:
		policies, err := c.informers.GetListers().FaultInjectionPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get FaultInjectionPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.FaultInjectionPolicyGVK)
	case SessionStickyPoliciesResourceType:
		policies, err := c.informers.GetListers().SessionStickyPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get SessionStickyPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.SessionStickyPolicyGVK)
	case LoadBalancerPoliciesResourceType:
		policies, err := c.informers.GetListers().LoadBalancerPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get LoadBalancerPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.LoadBalancerPolicyGVK)
	case CircuitBreakingPoliciesResourceType:
		policies, err := c.informers.GetListers().CircuitBreakingPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get CircuitBreakingPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.CircuitBreakingPolicyGVK)
	case HealthCheckPoliciesResourceType:
		policies, err := c.informers.GetListers().HealthCheckPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get HealthCheckPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.HealthCheckPolicyGVK)
	case RetryPoliciesResourceType:
		policies, err := c.informers.GetListers().RetryPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get RetryPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.RetryPolicyGVK)
	case GatewayTLSPoliciesResourceType:
		policies, err := c.informers.GetListers().GatewayTLSPolicy.List(selectAll)
		if err != nil {
			log.Error().Msgf("Failed to get GatewayTLSPolicies: %s", err)
			return resources
		}
		resources = setGroupVersionKind(policies, constants.GatewayTLSPolicyGVK)
	default:
		log.Error().Msgf("Unknown resource type: %s", resourceType)
		return nil
	}

	if shouldSort {
		sort.Slice(resources, func(i, j int) bool {
			if resources[i].GetCreationTimestamp().Time.Equal(resources[j].GetCreationTimestamp().Time) {
				return client.ObjectKeyFromObject(resources[i]).String() < client.ObjectKeyFromObject(resources[j]).String()
			}

			return resources[i].GetCreationTimestamp().Time.Before(resources[j].GetCreationTimestamp().Time)
		})
	}

	return resources
}

func (c *GatewayCache) isRoutableService(service client.ObjectKey) bool {
	for _, checkRoutableFunc := range []func(client.ObjectKey) bool{
		c.isRoutableHTTPService,
		c.isRoutableGRPCService,
		c.isRoutableTLSService,
		c.isRoutableTCPService,
		c.isRoutableUDPService,
	} {
		if checkRoutableFunc(service) {
			return true
		}
	}

	return false
}

func (c *GatewayCache) isRoutableHTTPService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(HTTPRoutesResourceType, false) {
		r := r.(*gwv1.HTTPRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}

				for _, filter := range backend.Filters {
					if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
						if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
							return true
						}
					}
				}
			}

			for _, filter := range rule.Filters {
				if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
					if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableGRPCService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(GRPCRoutesResourceType, false) {
		r := r.(*gwv1alpha2.GRPCRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}

				for _, filter := range backend.Filters {
					if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
						if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
							return true
						}
					}
				}
			}

			for _, filter := range rule.Filters {
				if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
					if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTLSService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(TLSRoutesResourceType, false) {
		r := r.(*gwv1alpha2.TLSRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTCPService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(TCPRoutesResourceType, false) {
		r := r.(*gwv1alpha2.TCPRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableUDPService(service client.ObjectKey) bool {
	for _, r := range c.getResourcesFromCache(UDPRoutesResourceType, false) {
		r := r.(*gwv1alpha2.UDPRoute)
		for _, rule := range r.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) isEffectiveRoute(parentRefs []gwv1.ParentReference) bool {
	gateways := c.getActiveGateways()

	if len(gateways) == 0 {
		return false
	}

	for _, parentRef := range parentRefs {
		for _, gw := range gateways {
			if gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(gw)) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayCache) isEffectiveTargetRef(targetRef gwv1alpha2.PolicyTargetReference) bool {
	if targetRef.Group != constants.GatewayAPIGroup {
		return false
	}

	switch targetRef.Kind {
	case constants.GatewayAPIGatewayKind:
		gateways := c.getActiveGateways()
		if len(gateways) == 0 {
			return false
		}

		for _, gateway := range gateways {
			if gwutils.IsRefToTarget(targetRef, gateway) {
				return true
			}
		}
	case constants.GatewayAPIHTTPRouteKind:
		httproutes := c.getResourcesFromCache(HTTPRoutesResourceType, false)
		if len(httproutes) == 0 {
			return false
		}

		for _, route := range httproutes {
			if gwutils.IsRefToTarget(targetRef, route) {
				return true
			}
		}
	case constants.GatewayAPIGRPCRouteKind:
		grpcroutes := c.getResourcesFromCache(GRPCRoutesResourceType, false)
		if len(grpcroutes) == 0 {
			return false
		}

		for _, route := range grpcroutes {
			if gwutils.IsRefToTarget(targetRef, route) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayCache) isRoutableTargetService(owner client.Object, targetRef gwv1alpha2.PolicyTargetReference) bool {
	key := client.ObjectKey{
		Namespace: gwutils.Namespace(targetRef.Namespace, owner.GetNamespace()),
		Name:      string(targetRef.Name),
	}

	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
		return c.isRoutableService(key)
	}

	return false
}

func (c *GatewayCache) isSecretReferred(secret client.ObjectKey) bool {
	//ctx := context.TODO()
	for _, gw := range c.getActiveGateways() {
		for _, l := range gw.Spec.Listeners {
			switch l.Protocol {
			case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
				if l.TLS == nil {
					continue
				}

				if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
					if len(l.TLS.CertificateRefs) == 0 {
						continue
					}

					for _, ref := range l.TLS.CertificateRefs {
						if isRefToSecret(ref, secret, gw.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	for _, ut := range c.getResourcesFromCache(UpstreamTLSPoliciesResourceType, false) {
		ut := ut.(*gwpav1alpha1.UpstreamTLSPolicy)

		if ut.Spec.DefaultConfig != nil {
			if isRefToSecret(ut.Spec.DefaultConfig.CertificateRef, secret, ut.Namespace) {
				return true
			}
		}

		if len(ut.Spec.Ports) > 0 {
			for _, port := range ut.Spec.Ports {
				if port.Config == nil {
					continue
				}

				if isRefToSecret(port.Config.CertificateRef, secret, ut.Namespace) {
					return true
				}
			}
		}
	}

	return false
}

func (c *GatewayCache) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj, err := c.informers.GetListers().Secret.Secrets(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(constants.SecretGVK)

	return obj, nil
}

func (c *GatewayCache) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj, err := c.informers.GetListers().Service.Services(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	obj.GetObjectKind().SetGroupVersionKind(constants.ServiceGVK)

	return obj, nil
}

func (c *GatewayCache) isHeadlessServiceWithoutSelector(key client.ObjectKey) bool {
	service, err := c.getServiceFromCache(key)
	if err != nil {
		log.Error().Msgf("failed to get service from cache: %v", err)
		return false
	}

	return service.Spec.ClusterIP == corev1.ClusterIPNone && len(service.Spec.Selector) == 0
}
