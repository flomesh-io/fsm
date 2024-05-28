package cache

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/gateway/policy"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//func (c *GatewayCache) policyAttachments() globalPolicyAttachments {
//	return globalPolicyAttachments{
//		rateLimits:      c.rateLimits(),
//		accessControls:  c.accessControls(),
//		faultInjections: c.faultInjections(),
//	}
//}

func (c *GatewayProcessor) getPortPolicyEnrichers(gateway *gwv1.Gateway) []policy.PortPolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(gateway).String()
	selector := fields.OneTermEqualSelector(constants.PortPolicyAttachmentIndex, key)

	return []policy.PortPolicyEnricher{
		policy.NewRateLimitPortEnricher(cc, selector),
		policy.NewAccessControlPortEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getHostnamePolicyEnrichers(route client.Object) []policy.HostnamePolicyEnricher {
	cc := c.cache.client
	key := fmt.Sprintf("%s/%s/%s", route.GetObjectKind().GroupVersionKind().Kind, route.GetNamespace(), route.GetName())
	selector := fields.OneTermEqualSelector(constants.HostnamePolicyAttachmentIndex, key)

	return []policy.HostnamePolicyEnricher{
		policy.NewRateLimitHostnameEnricher(cc, selector),
		policy.NewAccessControlHostnameEnricher(cc, selector),
		policy.NewFaultInjectionHostnameEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getHTTPRoutePolicyEnrichers(route *gwv1.HTTPRoute) []policy.HTTPRoutePolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(route).String()
	selector := fields.OneTermEqualSelector(constants.HTTPRoutePolicyAttachmentIndex, key)

	return []policy.HTTPRoutePolicyEnricher{
		policy.NewRateLimitHTTPRouteEnricher(cc, selector),
		policy.NewAccessControlHTTPRouteEnricher(cc, selector),
		policy.NewFaultInjectionHTTPRouteEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getGRPCRoutePolicyEnrichers(route *gwv1.GRPCRoute) []policy.GRPCRoutePolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(route).String()
	selector := fields.OneTermEqualSelector(constants.GRPCRoutePolicyAttachmentIndex, key)

	return []policy.GRPCRoutePolicyEnricher{
		policy.NewRateLimitGRPCRouteEnricher(cc, selector),
		policy.NewAccessControlGRPCRouteEnricher(cc, selector),
		policy.NewFaultInjectionGRPCRouteEnricher(cc, selector),
	}
}

func (c *GatewayProcessor) getServicePolicyEnrichers(svc *corev1.Service) []policy.ServicePolicyEnricher {
	cc := c.cache.client
	key := client.ObjectKeyFromObject(svc).String()
	selector := fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, key)

	return []policy.ServicePolicyEnricher{
		policy.NewSessionStickyPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewLoadBalancerPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewCircuitBreakingPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewUpstreamTLSPolicyEnricher(cc, selector, c.targetRefToServicePortName, c.secretRefToSecret),
		policy.NewRetryPolicyEnricher(cc, selector, c.targetRefToServicePortName),
		policy.NewHealthCheckPolicyEnricher(cc, selector, c.targetRefToServicePortName),
	}
}

//func (c *GatewayCache) rateLimits() map[gwtypes.PolicyMatchType][]*gwpav1alpha1.RateLimitPolicy {
//	policies := make(map[gwtypes.PolicyMatchType][]*gwpav1alpha1.RateLimitPolicy)
//
//	for _, p := range []struct {
//		matchType gwtypes.PolicyMatchType
//		fn        func(cache.Cache, fields.Selector) []client.Object
//		selector  fields.Selector
//	}{
//		{
//			matchType: gwtypes.PolicyMatchTypePort,
//			fn:        gwutils.GetRateLimitsMatchTypePort,
//			selector:  gwtypes.OneTermSelector(constants.PortPolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeHostnames,
//			fn:        gwutils.GetRateLimitsMatchTypeHostname,
//			selector:  gwtypes.OneTermSelector(constants.HostnamePolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeHTTPRoute,
//			fn:        gwutils.GetRateLimitsMatchTypeHTTPRoute,
//			selector:  gwtypes.OneTermSelector(constants.HTTPRoutePolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeGRPCRoute,
//			fn:        gwutils.GetRateLimitsMatchTypeGRPCRoute,
//			selector:  gwtypes.OneTermSelector(constants.GRPCRoutePolicyAttachmentIndex),
//		},
//	} {
//		if result := p.fn(c.client, p.selector); len(result) > 0 {
//			for _, r := range result {
//				policies[p.matchType] = append(policies[p.matchType], r.(*gwpav1alpha1.RateLimitPolicy))
//			}
//		}
//	}
//
//	return policies
//	//rateLimits := make(map[gwtypes.PolicyMatchType][]gwpav1alpha1.RateLimitPolicy)
//	//for _, matchType := range []gwtypes.PolicyMatchType{
//	//	gwtypes.PolicyMatchTypePort,
//	//	gwtypes.PolicyMatchTypeHostnames,
//	//	gwtypes.PolicyMatchTypeHTTPRoute,
//	//	gwtypes.PolicyMatchTypeGRPCRoute,
//	//} {
//	//	rateLimits[matchType] = make([]gwpav1alpha1.RateLimitPolicy, 0)
//	//}
//	//
//	//for _, p := range c.getResourcesFromCache(informers.RateLimitPoliciesResourceType, true) {
//	//	p := p.(*gwpav1alpha1.RateLimitPolicy)
//	//
//	//	if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
//	//		spec := p.Spec
//	//		targetRef := spec.TargetRef
//	//
//	//		switch {
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.GatewayGVK) && len(spec.Ports) > 0:
//	//			rateLimits[gwtypes.PolicyMatchTypePort] = append(rateLimits[gwtypes.PolicyMatchTypePort], *p)
//	//		case (gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK)) && len(spec.Hostnames) > 0:
//	//			rateLimits[gwtypes.PolicyMatchTypeHostnames] = append(rateLimits[gwtypes.PolicyMatchTypeHostnames], *p)
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) && len(spec.HTTPRateLimits) > 0:
//	//			rateLimits[gwtypes.PolicyMatchTypeHTTPRoute] = append(rateLimits[gwtypes.PolicyMatchTypeHTTPRoute], *p)
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) && len(spec.GRPCRateLimits) > 0:
//	//			rateLimits[gwtypes.PolicyMatchTypeGRPCRoute] = append(rateLimits[gwtypes.PolicyMatchTypeGRPCRoute], *p)
//	//		}
//	//	}
//	//}
//	//
//	//return rateLimits
//	//c := c.client
//	//policies := make(map[gwtypes.PolicyMatchType][]client.Object)
//	//
//	//for matchType, f := range map[gwtypes.PolicyMatchType]func(cache.Cache, client.Object) []client.Object{
//	//	gwtypes.PolicyMatchTypePort:      gwutils.GetRateLimitsMatchTypePort,
//	//	gwtypes.PolicyMatchTypeHostnames: gwutils.GetRateLimitsMatchTypeHostname,
//	//	gwtypes.PolicyMatchTypeHTTPRoute: gwutils.GetRateLimitsMatchTypeHTTPRoute,
//	//	gwtypes.PolicyMatchTypeGRPCRoute: gwutils.GetRateLimitsMatchTypeGRPCRoute,
//	//} {
//	//	if result := f(c.client, target); len(result) > 0 {
//	//		policies[matchType] = result
//	//	}
//	//}
//	//
//	//return policies
//}
//
//func (c *GatewayCache) accessControls() map[gwtypes.PolicyMatchType][]*gwpav1alpha1.AccessControlPolicy {
//	policies := make(map[gwtypes.PolicyMatchType][]*gwpav1alpha1.AccessControlPolicy)
//
//	for _, p := range []struct {
//		matchType gwtypes.PolicyMatchType
//		fn        func(cache.Cache, fields.Selector) []client.Object
//		selector  fields.Selector
//	}{
//		{
//			matchType: gwtypes.PolicyMatchTypePort,
//			fn:        gwutils.GetAccessControlsMatchTypePort,
//			selector:  gwtypes.OneTermSelector(constants.PortPolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeHostnames,
//			fn:        gwutils.GetAccessControlsMatchTypeHostname,
//			selector:  gwtypes.OneTermSelector(constants.HostnamePolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeHTTPRoute,
//			fn:        gwutils.GetAccessControlsMatchTypeHTTPRoute,
//			selector:  gwtypes.OneTermSelector(constants.HTTPRoutePolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeGRPCRoute,
//			fn:        gwutils.GetAccessControlsMatchTypeGRPCRoute,
//			selector:  gwtypes.OneTermSelector(constants.GRPCRoutePolicyAttachmentIndex),
//		},
//	} {
//		if result := p.fn(c.client, p.selector); len(result) > 0 {
//			for _, r := range result {
//				policies[p.matchType] = append(policies[p.matchType], r.(*gwpav1alpha1.AccessControlPolicy))
//			}
//		}
//	}
//
//	return policies
//	//accessControls := make(map[gwtypes.PolicyMatchType][]gwpav1alpha1.AccessControlPolicy)
//	//for _, matchType := range []gwtypes.PolicyMatchType{
//	//	gwtypes.PolicyMatchTypePort,
//	//	gwtypes.PolicyMatchTypeHostnames,
//	//	gwtypes.PolicyMatchTypeHTTPRoute,
//	//	gwtypes.PolicyMatchTypeGRPCRoute,
//	//} {
//	//	accessControls[matchType] = make([]gwpav1alpha1.AccessControlPolicy, 0)
//	//}
//	//
//	//for _, p := range c.getResourcesFromCache(informers.AccessControlPoliciesResourceType, true) {
//	//	p := p.(*gwpav1alpha1.AccessControlPolicy)
//	//
//	//	if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
//	//		spec := p.Spec
//	//		targetRef := spec.TargetRef
//	//
//	//		switch {
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.GatewayGVK) && len(spec.Ports) > 0:
//	//			accessControls[gwtypes.PolicyMatchTypePort] = append(accessControls[gwtypes.PolicyMatchTypePort], *p)
//	//		case (gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK)) && len(spec.Hostnames) > 0:
//	//			accessControls[gwtypes.PolicyMatchTypeHostnames] = append(accessControls[gwtypes.PolicyMatchTypeHostnames], *p)
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) && len(spec.HTTPAccessControls) > 0:
//	//			accessControls[gwtypes.PolicyMatchTypeHTTPRoute] = append(accessControls[gwtypes.PolicyMatchTypeHTTPRoute], *p)
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) && len(spec.GRPCAccessControls) > 0:
//	//			accessControls[gwtypes.PolicyMatchTypeGRPCRoute] = append(accessControls[gwtypes.PolicyMatchTypeGRPCRoute], *p)
//	//		}
//	//	}
//	//}
//	//
//	//return accessControls
//}
//
//func (c *GatewayCache) faultInjections() map[gwtypes.PolicyMatchType][]*gwpav1alpha1.FaultInjectionPolicy {
//	policies := make(map[gwtypes.PolicyMatchType][]*gwpav1alpha1.FaultInjectionPolicy)
//
//	for _, p := range []struct {
//		matchType gwtypes.PolicyMatchType
//		fn        func(cache.Cache, fields.Selector) []client.Object
//		selector  fields.Selector
//	}{
//		{
//			matchType: gwtypes.PolicyMatchTypeHostnames,
//			fn:        gwutils.GetFaultInjectionsMatchTypeHostname,
//			selector:  gwtypes.OneTermSelector(constants.HostnamePolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeHTTPRoute,
//			fn:        gwutils.GetFaultInjectionsMatchTypeHTTPRoute,
//			selector:  gwtypes.OneTermSelector(constants.HTTPRoutePolicyAttachmentIndex),
//		},
//		{
//			matchType: gwtypes.PolicyMatchTypeGRPCRoute,
//			fn:        gwutils.GetFaultInjectionsMatchTypeGRPCRoute,
//			selector:  gwtypes.OneTermSelector(constants.GRPCRoutePolicyAttachmentIndex),
//		},
//	} {
//		if result := p.fn(c.client, p.selector); len(result) > 0 {
//			for _, r := range result {
//				policies[p.matchType] = append(policies[p.matchType], r.(*gwpav1alpha1.FaultInjectionPolicy))
//			}
//		}
//	}
//
//	return policies
//	//faultInjections := make(map[gwtypes.PolicyMatchType][]gwpav1alpha1.FaultInjectionPolicy)
//	//for _, matchType := range []gwtypes.PolicyMatchType{
//	//	gwtypes.PolicyMatchTypeHostnames,
//	//	gwtypes.PolicyMatchTypeHTTPRoute,
//	//	gwtypes.PolicyMatchTypeGRPCRoute,
//	//} {
//	//	faultInjections[matchType] = make([]gwpav1alpha1.FaultInjectionPolicy, 0)
//	//}
//	//
//	//for _, p := range c.getResourcesFromCache(informers.FaultInjectionPoliciesResourceType, true) {
//	//	p := p.(*gwpav1alpha1.FaultInjectionPolicy)
//	//
//	//	if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
//	//		spec := p.Spec
//	//		targetRef := spec.TargetRef
//	//
//	//		switch {
//	//		case (gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK)) && len(spec.Hostnames) > 0:
//	//			faultInjections[gwtypes.PolicyMatchTypeHostnames] = append(faultInjections[gwtypes.PolicyMatchTypeHostnames], *p)
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) && len(spec.HTTPFaultInjections) > 0:
//	//			faultInjections[gwtypes.PolicyMatchTypeHTTPRoute] = append(faultInjections[gwtypes.PolicyMatchTypeHTTPRoute], *p)
//	//		case gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) && len(spec.GRPCFaultInjections) > 0:
//	//			faultInjections[gwtypes.PolicyMatchTypeGRPCRoute] = append(faultInjections[gwtypes.PolicyMatchTypeGRPCRoute], *p)
//	//		}
//	//	}
//	//}
//	//
//	//return faultInjections
//}

//func filterPoliciesByRoute(referenceGrants []*gwv1beta1.ReferenceGrant, policies globalPolicyAttachments, route client.Object) routePolicies {
//	result := routePolicies{
//		hostnamesRateLimits:      make([]gwpav1alpha1.RateLimitPolicy, 0),
//		httpRouteRateLimits:      make([]gwpav1alpha1.RateLimitPolicy, 0),
//		grpcRouteRateLimits:      make([]gwpav1alpha1.RateLimitPolicy, 0),
//		hostnamesAccessControls:  make([]gwpav1alpha1.AccessControlPolicy, 0),
//		httpRouteAccessControls:  make([]gwpav1alpha1.AccessControlPolicy, 0),
//		grpcRouteAccessControls:  make([]gwpav1alpha1.AccessControlPolicy, 0),
//		hostnamesFaultInjections: make([]gwpav1alpha1.FaultInjectionPolicy, 0),
//		httpRouteFaultInjections: make([]gwpav1alpha1.FaultInjectionPolicy, 0),
//		grpcRouteFaultInjections: make([]gwpav1alpha1.FaultInjectionPolicy, 0),
//	}
//
//	if len(policies.rateLimits[gwtypes.PolicyMatchTypeHostnames]) > 0 {
//		for _, rateLimit := range policies.rateLimits[gwtypes.PolicyMatchTypeHostnames] {
//			rateLimit := rateLimit
//			if !gwutils.IsTargetRefToTarget(rateLimit.Spec.TargetRef, route) {
//				continue
//			}
//			if !gwutils.HasAccessToTargetRef(&rateLimit, rateLimit.Spec.TargetRef, referenceGrants) {
//				continue
//			}
//			if gwutils.HasAccessToTarget(referenceGrants, &rateLimit, rateLimit.Spec.TargetRef, route) {
//				result.hostnamesRateLimits = append(result.hostnamesRateLimits, rateLimit)
//			}
//		}
//	}
//
//	if len(policies.rateLimits[gwtypes.PolicyMatchTypeHTTPRoute]) > 0 {
//		for _, rateLimit := range policies.rateLimits[gwtypes.PolicyMatchTypeHTTPRoute] {
//			rateLimit := rateLimit
//			if gwutils.HasAccessToTarget(referenceGrants, &rateLimit, rateLimit.Spec.TargetRef, route) {
//				result.httpRouteRateLimits = append(result.httpRouteRateLimits, rateLimit)
//			}
//		}
//	}
//
//	if len(policies.rateLimits[gwtypes.PolicyMatchTypeGRPCRoute]) > 0 {
//		for _, rateLimit := range policies.rateLimits[gwtypes.PolicyMatchTypeGRPCRoute] {
//			rateLimit := rateLimit
//			if gwutils.HasAccessToTarget(referenceGrants, &rateLimit, rateLimit.Spec.TargetRef, route) {
//				result.grpcRouteRateLimits = append(result.grpcRouteRateLimits, rateLimit)
//			}
//		}
//	}
//
//	if len(policies.accessControls[gwtypes.PolicyMatchTypeHostnames]) > 0 {
//		for _, ac := range policies.accessControls[gwtypes.PolicyMatchTypeHostnames] {
//			ac := ac
//			if gwutils.HasAccessToTarget(referenceGrants, &ac, ac.Spec.TargetRef, route) {
//				result.hostnamesAccessControls = append(result.hostnamesAccessControls, ac)
//			}
//		}
//	}
//
//	if len(policies.accessControls[gwtypes.PolicyMatchTypeHTTPRoute]) > 0 {
//		for _, ac := range policies.accessControls[gwtypes.PolicyMatchTypeHTTPRoute] {
//			ac := ac
//			if gwutils.HasAccessToTarget(referenceGrants, &ac, ac.Spec.TargetRef, route) {
//				result.httpRouteAccessControls = append(result.httpRouteAccessControls, ac)
//			}
//		}
//	}
//
//	if len(policies.accessControls[gwtypes.PolicyMatchTypeGRPCRoute]) > 0 {
//		for _, ac := range policies.accessControls[gwtypes.PolicyMatchTypeGRPCRoute] {
//			ac := ac
//			if gwutils.HasAccessToTarget(referenceGrants, &ac, ac.Spec.TargetRef, route) {
//				result.grpcRouteAccessControls = append(result.grpcRouteAccessControls, ac)
//			}
//		}
//	}
//
//	if len(policies.faultInjections[gwtypes.PolicyMatchTypeHostnames]) > 0 {
//		for _, fj := range policies.faultInjections[gwtypes.PolicyMatchTypeHostnames] {
//			fj := fj
//			if gwutils.HasAccessToTarget(referenceGrants, &fj, fj.Spec.TargetRef, route) {
//				result.hostnamesFaultInjections = append(result.hostnamesFaultInjections, fj)
//			}
//		}
//	}
//
//	if len(policies.faultInjections[gwtypes.PolicyMatchTypeHTTPRoute]) > 0 {
//		for _, fj := range policies.faultInjections[gwtypes.PolicyMatchTypeHTTPRoute] {
//			fj := fj
//			if gwutils.HasAccessToTarget(referenceGrants, &fj, fj.Spec.TargetRef, route) {
//				result.httpRouteFaultInjections = append(result.httpRouteFaultInjections, fj)
//			}
//		}
//	}
//
//	if len(policies.faultInjections[gwtypes.PolicyMatchTypeGRPCRoute]) > 0 {
//		for _, fj := range policies.faultInjections[gwtypes.PolicyMatchTypeGRPCRoute] {
//			fj := fj
//			if gwutils.HasAccessToTarget(referenceGrants, &fj, fj.Spec.TargetRef, route) {
//				result.grpcRouteFaultInjections = append(result.grpcRouteFaultInjections, fj)
//			}
//		}
//	}
//
//	return result
//}

//func (c *GatewayProcessor) sessionStickies(cache cache.Cache, selector fields.Selector) map[string]*gwpav1alpha1.SessionStickyConfig {
//	sessionStickies := make(map[string]*gwpav1alpha1.SessionStickyConfig)
//
//	gwutils.SortResources(gwutils.GetSessionStickies(cache, selector))
//
//	for _, sessionSticky := range c.getResourcesFromCache(informers.SessionStickyPoliciesResourceType, true) {
//		sessionSticky := sessionSticky.(*gwpav1alpha1.SessionStickyPolicy)
//
//		if gwutils.IsAcceptedPolicyAttachment(sessionSticky.Status.Conditions) {
//			for _, p := range sessionSticky.Spec.Ports {
//				if svcPortName := c.targetRefToServicePortName(sessionSticky, sessionSticky.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
//					cfg := sessionsticky.ComputeSessionStickyConfig(p.Config, sessionSticky.Spec.DefaultConfig)
//
//					if cfg == nil {
//						continue
//					}
//
//					if _, ok := sessionStickies[svcPortName.String()]; ok {
//						log.Warn().Msgf("Policy is already defined for service port %s, SessionStickyPolicy %s/%s:%d will be dropped", svcPortName.String(), sessionSticky.Namespace, sessionSticky.Name, p.Port)
//						continue
//					}
//
//					sessionStickies[svcPortName.String()] = cfg
//				}
//			}
//		}
//	}
//
//	return sessionStickies
//}

//func (c *GatewayProcessor) loadBalancers() map[string]*gwpav1alpha1.LoadBalancerType {
//	loadBalancers := make(map[string]*gwpav1alpha1.LoadBalancerType)
//
//	for _, lb := range c.getResourcesFromCache(informers.LoadBalancerPoliciesResourceType, true) {
//		lb := lb.(*gwpav1alpha1.LoadBalancerPolicy)
//
//		if gwutils.IsAcceptedPolicyAttachment(lb.Status.Conditions) {
//			for _, p := range lb.Spec.Ports {
//				if svcPortName := c.targetRefToServicePortName(lb, lb.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
//					t := loadbalancer.ComputeLoadBalancerType(p.Type, lb.Spec.DefaultType)
//
//					if t == nil {
//						continue
//					}
//
//					if _, ok := loadBalancers[svcPortName.String()]; ok {
//						log.Warn().Msgf("Policy is already defined for service port %s, LoadBalancerPolicy %s/%s:%d will be dropped", svcPortName.String(), lb.Namespace, lb.Name, p.Port)
//						continue
//					}
//
//					loadBalancers[svcPortName.String()] = t
//				}
//			}
//		}
//	}
//
//	return loadBalancers
//}

//func (c *GatewayProcessor) circuitBreakings() map[string]*gwpav1alpha1.CircuitBreakingConfig {
//	configs := make(map[string]*gwpav1alpha1.CircuitBreakingConfig)
//
//	list := &gwpav1alpha1.CircuitBreakingPolicyList{}
//	err := c.cache.client.List(context.Background(), list)
//	if err != nil {
//		log.Error().Msgf("Failed to list CircuitBreakingPolicies: %v", err)
//		return nil
//	}
//
//	for _, circuitBreaking := range gwutils.SortResources(toSlicePtr(list.Items)) {
//		//circuitBreaking := circuitBreaking.(*gwpav1alpha1.CircuitBreakingPolicy)
//
//		if gwutils.IsAcceptedPolicyAttachment(circuitBreaking.Status.Conditions) {
//			for _, p := range circuitBreaking.Spec.Ports {
//				if svcPortName := c.targetRefToServicePortName(circuitBreaking, circuitBreaking.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
//					cfg := circuitbreaking.ComputeCircuitBreakingConfig(p.Config, circuitBreaking.Spec.DefaultConfig)
//
//					if cfg == nil {
//						continue
//					}
//
//					if _, ok := configs[svcPortName.String()]; ok {
//						log.Warn().Msgf("Policy is already defined for service port %s, CircuitBreakingPolicy %s/%s:%d will be dropped", svcPortName.String(), circuitBreaking.Namespace, circuitBreaking.Name, p.Port)
//						continue
//					}
//
//					configs[svcPortName.String()] = cfg
//				}
//			}
//		}
//	}
//
//	return configs
//}

//func (c *GatewayProcessor) healthChecks() map[string]*gwpav1alpha1.HealthCheckConfig {
//	configs := make(map[string]*gwpav1alpha1.HealthCheckConfig)
//
//	for _, healthCheck := range c.getResourcesFromCache(informers.HealthCheckPoliciesResourceType, true) {
//		healthCheck := healthCheck.(*gwpav1alpha1.HealthCheckPolicy)
//
//		if gwutils.IsAcceptedPolicyAttachment(healthCheck.Status.Conditions) {
//			for _, p := range healthCheck.Spec.Ports {
//				if svcPortName := c.targetRefToServicePortName(healthCheck, healthCheck.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
//					cfg := healthcheck.ComputeHealthCheckConfig(p.Config, healthCheck.Spec.DefaultConfig)
//
//					if cfg == nil {
//						continue
//					}
//
//					if _, ok := configs[svcPortName.String()]; ok {
//						log.Warn().Msgf("Policy is already defined for service port %s, HealthCheckPolicy %s/%s:%d will be dropped", svcPortName.String(), healthCheck.Namespace, healthCheck.Name, p.Port)
//						continue
//					}
//
//					configs[svcPortName.String()] = cfg
//				}
//			}
//		}
//	}
//
//	return configs
//}

//func (c *GatewayProcessor) upstreamTLS() map[string]*policy.UpstreamTLSConfig {
//	configs := make(map[string]*policy.UpstreamTLSConfig)
//
//	for _, upstreamTLS := range c.getResourcesFromCache(informers.UpstreamTLSPoliciesResourceType, true) {
//		upstreamTLS := upstreamTLS.(*gwpav1alpha1.UpstreamTLSPolicy)
//
//		if gwutils.IsAcceptedPolicyAttachment(upstreamTLS.Status.Conditions) {
//			for _, p := range upstreamTLS.Spec.Ports {
//				if svcPortName := c.targetRefToServicePortName(upstreamTLS, upstreamTLS.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
//					cfg := upstreamtls.ComputeUpstreamTLSConfig(p.Config, upstreamTLS.Spec.DefaultConfig)
//
//					if cfg == nil {
//						continue
//					}
//
//					secret, err := c.secretRefToSecret(upstreamTLS, cfg.CertificateRef)
//					if err != nil {
//						log.Error().Msgf("Failed to resolve Secret: %s", err)
//						continue
//					}
//
//					if _, ok := configs[svcPortName.String()]; ok {
//						log.Warn().Msgf("Policy is already defined for service port %s, UpstreamTLSPolicy %s/%s:%d will be dropped", svcPortName.String(), upstreamTLS.Namespace, upstreamTLS.Name, p.Port)
//						continue
//					}
//
//					configs[svcPortName.String()] = &policy.UpstreamTLSConfig{
//						MTLS:   cfg.MTLS,
//						Secret: secret,
//					}
//				}
//			}
//		}
//	}
//
//	return configs
//}

//func (c *GatewayProcessor) retryConfigs() map[string]*gwpav1alpha1.RetryConfig {
//	configs := make(map[string]*gwpav1alpha1.RetryConfig)
//
//	for _, retryPolicy := range c.getResourcesFromCache(informers.RetryPoliciesResourceType, true) {
//		retryPolicy := retryPolicy.(*gwpav1alpha1.RetryPolicy)
//
//		if gwutils.IsAcceptedPolicyAttachment(retryPolicy.Status.Conditions) {
//			for _, p := range retryPolicy.Spec.Ports {
//				if svcPortName := c.targetRefToServicePortName(retryPolicy, retryPolicy.Spec.TargetRef, int32(p.Port)); svcPortName != nil {
//					cfg := retry.ComputeRetryConfig(p.Config, retryPolicy.Spec.DefaultConfig)
//
//					if cfg == nil {
//						continue
//					}
//
//					if _, ok := configs[svcPortName.String()]; ok {
//						log.Warn().Msgf("Policy is already defined for service port %s, RetryPolicy %s/%s:%d will be dropped", svcPortName.String(), retryPolicy.Namespace, retryPolicy.Name, p.Port)
//						continue
//					}
//
//					configs[svcPortName.String()] = cfg
//				}
//			}
//		}
//	}
//
//	return configs
//}
