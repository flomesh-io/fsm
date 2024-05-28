package utils

import (
	"context"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// ---------------------------- Access Control ----------------------------

func GetAccessControlsMatchTypePort(cache cache.Cache, selector fields.Selector) []client.Object {
	portPolicyList := &gwpav1alpha1.AccessControlPolicyList{}
	if err := cache.List(context.Background(), portPolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(portPolicyList.Items)),
		getGatewayRefGrants(cache),
		isAcceptedAccessControlPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.AccessControlPolicy)
			return len(p.Spec.Ports) == 0
		},
		accessControlPolicyHasAccessToTargetRef,
	)
}

func GetAccessControlsMatchTypeHostname(cache cache.Cache, selector fields.Selector) []client.Object {
	hostnamePolicyList := &gwpav1alpha1.AccessControlPolicyList{}
	if err := cache.List(context.Background(), hostnamePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(hostnamePolicyList.Items)),
		getHostnameRefGrants(cache),
		isAcceptedAccessControlPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.AccessControlPolicy)
			return len(p.Spec.Hostnames) == 0
		},
		accessControlPolicyHasAccessToTargetRef,
	)
}

func GetAccessControlsMatchTypeHTTPRoute(cache cache.Cache, selector fields.Selector) []client.Object {
	httpRoutePolicyList := &gwpav1alpha1.AccessControlPolicyList{}
	if err := cache.List(context.Background(), httpRoutePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(httpRoutePolicyList.Items)),
		getHTTPRouteRefGrants(cache),
		isAcceptedAccessControlPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.AccessControlPolicy)
			return len(p.Spec.HTTPAccessControls) == 0
		},
		accessControlPolicyHasAccessToTargetRef,
	)
}

func GetAccessControlsMatchTypeGRPCRoute(cache cache.Cache, selector fields.Selector) []client.Object {
	grpcRoutePolicyList := &gwpav1alpha1.AccessControlPolicyList{}
	if err := cache.List(context.Background(), grpcRoutePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(grpcRoutePolicyList.Items)),
		getGRPCRouteRefGrants(cache),
		isAcceptedAccessControlPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.AccessControlPolicy)
			return len(p.Spec.GRPCAccessControls) == 0
		},
		accessControlPolicyHasAccessToTargetRef,
	)
}

func isAcceptedAccessControlPolicy(policy client.Object) bool {
	p := policy.(*gwpav1alpha1.AccessControlPolicy)
	return IsAcceptedPolicyAttachment(p.Status.Conditions)
}

func accessControlPolicyHasAccessToTargetRef(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
	p := policy.(*gwpav1alpha1.AccessControlPolicy)
	return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
}

// ---------------------------- Rate Limit ----------------------------

func GetRateLimitsMatchTypePort(cache cache.Cache, selector fields.Selector) []client.Object {
	portPolicyList := &gwpav1alpha1.RateLimitPolicyList{}
	if err := cache.List(context.Background(), portPolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(portPolicyList.Items)),
		getGatewayRefGrants(cache),
		isAcceptedRateLimitPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.RateLimitPolicy)
			return len(p.Spec.Ports) == 0
		},
		rateLimitPolicyHasAccessToTargetRef,
	)
}

func GetRateLimitsMatchTypeHostname(cache cache.Cache, selector fields.Selector) []client.Object {
	hostnamePolicyList := &gwpav1alpha1.RateLimitPolicyList{}
	if err := cache.List(context.Background(), hostnamePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(hostnamePolicyList.Items)),
		getHostnameRefGrants(cache),
		isAcceptedRateLimitPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.RateLimitPolicy)
			return len(p.Spec.Hostnames) == 0
		},
		rateLimitPolicyHasAccessToTargetRef,
	)
}

func GetRateLimitsMatchTypeHTTPRoute(cache cache.Cache, selector fields.Selector) []client.Object {
	httpRoutePolicyList := &gwpav1alpha1.RateLimitPolicyList{}
	if err := cache.List(context.Background(), httpRoutePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(httpRoutePolicyList.Items)),
		getHTTPRouteRefGrants(cache),
		isAcceptedRateLimitPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.RateLimitPolicy)
			return len(p.Spec.HTTPRateLimits) == 0
		},
		rateLimitPolicyHasAccessToTargetRef,
	)
}

func GetRateLimitsMatchTypeGRPCRoute(cache cache.Cache, selector fields.Selector) []client.Object {
	grpcRoutePolicyList := &gwpav1alpha1.RateLimitPolicyList{}
	if err := cache.List(context.Background(), grpcRoutePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(grpcRoutePolicyList.Items)),
		getGRPCRouteRefGrants(cache),
		isAcceptedRateLimitPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.RateLimitPolicy)
			return len(p.Spec.GRPCRateLimits) == 0
		},
		rateLimitPolicyHasAccessToTargetRef,
	)
}

func isAcceptedRateLimitPolicy(policy client.Object) bool {
	p := policy.(*gwpav1alpha1.RateLimitPolicy)
	return IsAcceptedPolicyAttachment(p.Status.Conditions)
}

func rateLimitPolicyHasAccessToTargetRef(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
	p := policy.(*gwpav1alpha1.RateLimitPolicy)
	return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
}

// ---------------------------- Fault Injection ----------------------------

func GetFaultInjectionsMatchTypeHostname(cache cache.Cache, selector fields.Selector) []client.Object {
	hostnamePolicyList := &gwpav1alpha1.FaultInjectionPolicyList{}
	if err := cache.List(context.Background(), hostnamePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(hostnamePolicyList.Items)),
		getHostnameRefGrants(cache),
		isAcceptedFaultInjectionPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
			return len(p.Spec.Hostnames) == 0
		},
		faultInjectionPolicyHasAccessToTargetRef,
	)
}

func GetFaultInjectionsMatchTypeHTTPRoute(cache cache.Cache, selector fields.Selector) []client.Object {
	httpRoutePolicyList := &gwpav1alpha1.FaultInjectionPolicyList{}
	if err := cache.List(context.Background(), httpRoutePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(httpRoutePolicyList.Items)),
		getHTTPRouteRefGrants(cache),
		isAcceptedFaultInjectionPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
			return len(p.Spec.HTTPFaultInjections) == 0
		},
		faultInjectionPolicyHasAccessToTargetRef,
	)
}

func GetFaultInjectionsMatchTypeGRPCRoute(cache cache.Cache, selector fields.Selector) []client.Object {
	grpcRoutePolicyList := &gwpav1alpha1.FaultInjectionPolicyList{}
	if err := cache.List(context.Background(), grpcRoutePolicyList, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(grpcRoutePolicyList.Items)),
		getGRPCRouteRefGrants(cache),
		isAcceptedFaultInjectionPolicy,
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
			return len(p.Spec.GRPCFaultInjections) == 0
		},
		faultInjectionPolicyHasAccessToTargetRef,
	)
}

func isAcceptedFaultInjectionPolicy(policy client.Object) bool {
	p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
	return IsAcceptedPolicyAttachment(p.Status.Conditions)
}

func faultInjectionPolicyHasAccessToTargetRef(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
	p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
	return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
}

// ---------------------------- Session Sticky ----------------------------

func GetSessionStickies(cache cache.Cache, selector fields.Selector) []client.Object {
	list := &gwpav1alpha1.SessionStickyPolicyList{}
	if err := cache.List(context.Background(), list, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(list.Items)),
		GetServiceRefGrants(cache),
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.SessionStickyPolicy)
			return IsAcceptedPolicyAttachment(p.Status.Conditions)
		},
		func(policy client.Object) bool {
			return false
		},
		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
			p := policy.(*gwpav1alpha1.SessionStickyPolicy)
			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
		},
	)
}

// ---------------------------- Circuit Breaking ----------------------------

func GetCircuitBreakings(cache cache.Cache, selector fields.Selector) []client.Object {
	list := &gwpav1alpha1.CircuitBreakingPolicyList{}
	if err := cache.List(context.Background(), list, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(list.Items)),
		GetServiceRefGrants(cache),
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.CircuitBreakingPolicy)
			return IsAcceptedPolicyAttachment(p.Status.Conditions)
		},
		func(policy client.Object) bool {
			return false
		},
		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
			p := policy.(*gwpav1alpha1.CircuitBreakingPolicy)
			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
		},
	)
}

// ---------------------------- Health Check ----------------------------

func GetHealthChecks(cache cache.Cache, selector fields.Selector) []client.Object {
	list := &gwpav1alpha1.HealthCheckPolicyList{}
	if err := cache.List(context.Background(), list, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(list.Items)),
		GetServiceRefGrants(cache),
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.HealthCheckPolicy)
			return IsAcceptedPolicyAttachment(p.Status.Conditions)
		},
		func(policy client.Object) bool {
			return false
		},
		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
			p := policy.(*gwpav1alpha1.HealthCheckPolicy)
			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
		},
	)
}

// ---------------------------- Load Balancer ----------------------------

func GetLoadBalancers(cache cache.Cache, selector fields.Selector) []client.Object {
	list := &gwpav1alpha1.LoadBalancerPolicyList{}
	if err := cache.List(context.Background(), list, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(list.Items)),
		GetServiceRefGrants(cache),
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
			return IsAcceptedPolicyAttachment(p.Status.Conditions)
		},
		func(policy client.Object) bool {
			return false
		},
		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
			p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
		},
	)
}

// ---------------------------- Retry ----------------------------

func GetRetries(cache cache.Cache, selector fields.Selector) []client.Object {
	list := &gwpav1alpha1.RetryPolicyList{}
	if err := cache.List(context.Background(), list, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(list.Items)),
		GetServiceRefGrants(cache),
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.RetryPolicy)
			return IsAcceptedPolicyAttachment(p.Status.Conditions)
		},
		func(policy client.Object) bool {
			return false
		},
		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
			p := policy.(*gwpav1alpha1.RetryPolicy)
			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
		},
	)
}

// ---------------------------- Upstream TLS ----------------------------

func GetUpStreamTLSes(cache cache.Cache, selector fields.Selector) []client.Object {
	list := &gwpav1alpha1.UpstreamTLSPolicyList{}
	if err := cache.List(context.Background(), list, &client.ListOptions{
		FieldSelector: selector,
	}); err != nil {
		return nil
	}

	return filterValidPolicies(
		toClientObjects(ToSlicePtr(list.Items)),
		GetServiceRefGrants(cache),
		func(policy client.Object) bool {
			p := policy.(*gwpav1alpha1.UpstreamTLSPolicy)
			return IsAcceptedPolicyAttachment(p.Status.Conditions)
		},
		func(policy client.Object) bool {
			return false
		},
		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
			p := policy.(*gwpav1alpha1.UpstreamTLSPolicy)
			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
		},
	)
}

// ---------------------------- Common methods ----------------------------

func toClientObjects[T client.Object](policies []T) []client.Object {
	objects := make([]client.Object, 0)
	for _, p := range policies {
		p := p
		objects = append(objects, p)
	}

	return objects
}

type isAcceptedFunc func(policy client.Object) bool
type noDataFunc func(policy client.Object) bool
type hasAccessFunc func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool

func filterValidPolicies[T client.Object](
	policies []T,
	refGrants []*gwv1beta1.ReferenceGrant,
	isAccepted isAcceptedFunc,
	noData noDataFunc,
	hasAccess hasAccessFunc,
) []client.Object {
	validPolicies := make([]client.Object, 0)
	for _, p := range policies {
		if !isAccepted(p) {
			continue
		}

		if noData(p) {
			continue
		}

		if !hasAccess(p, refGrants) {
			continue
		}

		validPolicies = append(validPolicies, p)
	}

	return validPolicies
}
