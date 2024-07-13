package utils

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

// HasAccessToBackendTargetRef checks if the policy has access to the target reference which is a backend service
func HasAccessToBackendTargetRef(client cache.Cache, policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference, pca status.PolicyAncestorStatusObject) bool {
	if !IsValidTargetRefToGroupKindOfService(targetRef) {
		parentRef := pca.GetAncestorRef()
		pca.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf("Unsupported backend group %s and kind %s for ancestor %s/%s", targetRef.Group, targetRef.Kind, NamespaceDerefOr(parentRef.Namespace, policy.GetNamespace()), parentRef.Name),
		)

		return false
	}

	gvk := policy.GetObjectKind().GroupVersionKind()
	if targetRef.Namespace != nil && string(*targetRef.Namespace) != policy.GetNamespace() && !ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     gvk.Group,
			Kind:      gvk.Kind,
			Namespace: policy.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(targetRef.Group),
			Kind:      string(targetRef.Kind),
			Namespace: string(*targetRef.Namespace),
			Name:      string(targetRef.Name),
		},
		GetServiceRefGrants(client),
	) {
		//log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
		//	gvk.Kind, gvk.Group, policy.GetNamespace(), policy.GetName(),
		//	string(targetRef.Kind), string(targetRef.Group), string(*targetRef.Namespace), targetRef.Name)
		pca.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonTargetNotFound,
			fmt.Sprintf("Target reference to %s/%s is not allowed", string(*targetRef.Namespace), targetRef.Name),
		)

		return false
	}

	return true
}

func IsPolicyAcceptedForAncestor(ancestorRef gwv1.ParentReference, ancestors []gwv1alpha2.PolicyAncestorStatus) bool {
	for _, ancestor := range ancestors {
		if cmp.Equal(ancestor.AncestorRef, ancestorRef) {
			return metautil.IsStatusConditionTrue(ancestor.Conditions, string(gwv1alpha2.PolicyConditionAccepted))
		}
	}

	return false
}

// ---------------------------- Access Control ----------------------------

// GetAccessControlsMatchTypePort returns a list of AccessControlPolicy objects that match the given selector
//func GetAccessControlsMatchTypePort(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.AccessControlPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getGatewayRefGrants(cache),
//		isAcceptedAccessControlPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.AccessControlPolicy)
//			return len(p.Spec.Ports) == 0
//		},
//		accessControlPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetAccessControlsMatchTypeHostname returns a list of AccessControlPolicy objects that match the given selector
//func GetAccessControlsMatchTypeHostname(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.AccessControlPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getHostnameRefGrants(cache),
//		isAcceptedAccessControlPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.AccessControlPolicy)
//			return len(p.Spec.Hostnames) == 0
//		},
//		accessControlPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetAccessControlsMatchTypeHTTPRoute returns a list of AccessControlPolicy objects that match the given selector
//func GetAccessControlsMatchTypeHTTPRoute(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.AccessControlPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getHTTPRouteRefGrants(cache),
//		isAcceptedAccessControlPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.AccessControlPolicy)
//			return len(p.Spec.HTTPAccessControls) == 0
//		},
//		accessControlPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetAccessControlsMatchTypeGRPCRoute returns a list of AccessControlPolicy objects that match the given selector
//func GetAccessControlsMatchTypeGRPCRoute(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.AccessControlPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getGRPCRouteRefGrants(cache),
//		isAcceptedAccessControlPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.AccessControlPolicy)
//			return len(p.Spec.GRPCAccessControls) == 0
//		},
//		accessControlPolicyHasAccessToTargetRef,
//	)
//}
//
//func isAcceptedAccessControlPolicy(policy client.Object) bool {
//	p := policy.(*gwpav1alpha1.AccessControlPolicy)
//	return IsAcceptedPolicyAttachment(p.Status.Conditions)
//}
//
//func accessControlPolicyHasAccessToTargetRef(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//	p := policy.(*gwpav1alpha1.AccessControlPolicy)
//	return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//}
//
//// ---------------------------- Rate Limit ----------------------------
//
//// GetRateLimitsMatchTypePort returns a list of RateLimitPolicy objects that match the given selector
//func GetRateLimitsMatchTypePort(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.RateLimitPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getGatewayRefGrants(cache),
//		isAcceptedRateLimitPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.RateLimitPolicy)
//			return len(p.Spec.Ports) == 0
//		},
//		rateLimitPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetRateLimitsMatchTypeHostname returns a list of RateLimitPolicy objects that match the given selector
//func GetRateLimitsMatchTypeHostname(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.RateLimitPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getHostnameRefGrants(cache),
//		isAcceptedRateLimitPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.RateLimitPolicy)
//			return len(p.Spec.Hostnames) == 0
//		},
//		rateLimitPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetRateLimitsMatchTypeHTTPRoute returns a list of RateLimitPolicy objects that match the given selector
//func GetRateLimitsMatchTypeHTTPRoute(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.RateLimitPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getHTTPRouteRefGrants(cache),
//		isAcceptedRateLimitPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.RateLimitPolicy)
//			return len(p.Spec.HTTPRateLimits) == 0
//		},
//		rateLimitPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetRateLimitsMatchTypeGRPCRoute returns a list of RateLimitPolicy objects that match the given selector
//func GetRateLimitsMatchTypeGRPCRoute(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.RateLimitPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getGRPCRouteRefGrants(cache),
//		isAcceptedRateLimitPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.RateLimitPolicy)
//			return len(p.Spec.GRPCRateLimits) == 0
//		},
//		rateLimitPolicyHasAccessToTargetRef,
//	)
//}
//
//func isAcceptedRateLimitPolicy(policy client.Object) bool {
//	p := policy.(*gwpav1alpha1.RateLimitPolicy)
//	return IsAcceptedPolicyAttachment(p.Status.Conditions)
//}
//
//func rateLimitPolicyHasAccessToTargetRef(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//	p := policy.(*gwpav1alpha1.RateLimitPolicy)
//	return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//}

// ---------------------------- Fault Injection ----------------------------

// GetFaultInjectionsMatchTypeHostname returns a list of FaultInjectionPolicy objects that match the given selector
//func GetFaultInjectionsMatchTypeHostname(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.FaultInjectionPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getHostnameRefGrants(cache),
//		isAcceptedFaultInjectionPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
//			return len(p.Spec.Hostnames) == 0
//		},
//		faultInjectionPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetFaultInjectionsMatchTypeHTTPRoute returns a list of FaultInjectionPolicy objects that match the given selector
//func GetFaultInjectionsMatchTypeHTTPRoute(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.FaultInjectionPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getHTTPRouteRefGrants(cache),
//		isAcceptedFaultInjectionPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
//			return len(p.Spec.HTTPFaultInjections) == 0
//		},
//		faultInjectionPolicyHasAccessToTargetRef,
//	)
//}
//
//// GetFaultInjectionsMatchTypeGRPCRoute returns a list of FaultInjectionPolicy objects that match the given selector
//func GetFaultInjectionsMatchTypeGRPCRoute(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.FaultInjectionPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		getGRPCRouteRefGrants(cache),
//		isAcceptedFaultInjectionPolicy,
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
//			return len(p.Spec.GRPCFaultInjections) == 0
//		},
//		faultInjectionPolicyHasAccessToTargetRef,
//	)
//}
//
//func isAcceptedFaultInjectionPolicy(policy client.Object) bool {
//	p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
//	return IsAcceptedPolicyAttachment(p.Status.Conditions)
//}
//
//func faultInjectionPolicyHasAccessToTargetRef(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//	p := policy.(*gwpav1alpha1.FaultInjectionPolicy)
//	return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//}

// ---------------------------- Session Sticky ----------------------------

// GetSessionStickies returns a list of SessionStickyPolicy objects that match the given selector
//func GetSessionStickies(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.SessionStickyPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.SessionStickyPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwpav1alpha1.SessionStickyPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}
//
//// ---------------------------- Circuit Breaking ----------------------------
//
//// GetCircuitBreakings returns a list of CircuitBreakingPolicy objects that match the given selector
//func GetCircuitBreakings(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.CircuitBreakingPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.CircuitBreakingPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwpav1alpha1.CircuitBreakingPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}
//
//// ---------------------------- Health Check ----------------------------
//
//// GetHealthChecks returns a list of HealthCheckPolicy objects that match the given selector
//func GetHealthChecks(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.HealthCheckPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.HealthCheckPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwpav1alpha1.HealthCheckPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}

// ---------------------------- Load Balancer ----------------------------

// GetLoadBalancers returns a list of LoadBalancerPolicy objects that match the given selector
//func GetLoadBalancers(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.LoadBalancerPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}
//
//// ---------------------------- Retry ----------------------------
//
//// GetRetries returns a list of RetryPolicy objects that match the given selector
//func GetRetries(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.RetryPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.RetryPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwpav1alpha1.RetryPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}
//
//// ---------------------------- Upstream TLS ----------------------------
//
//// GetUpStreamTLSes returns a list of UpstreamTLSPolicy objects that match the given selector
//func GetUpStreamTLSes(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwpav1alpha1.UpstreamTLSPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwpav1alpha1.UpstreamTLSPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwpav1alpha1.UpstreamTLSPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}

// ---------------------------- Backend TLS ----------------------------

// GetBackendTLSes returns a list of BackendTLSPolicy objects that match the given selector
//func GetBackendTLSes(cache cache.Cache, selector fields.Selector) []client.Object {
//	list := &gwv1alpha3.BackendTLSPolicyList{}
//	if err := cache.List(context.Background(), list, &client.ListOptions{FieldSelector: selector}); err != nil {
//		return nil
//	}
//
//	return filterValidPolicies(
//		toClientObjects(ToSlicePtr(list.Items)),
//		GetServiceRefGrants(cache),
//		func(policy client.Object) bool {
//			p := policy.(*gwv1alpha3.BackendTLSPolicy)
//			return IsAcceptedPolicyAttachment(p.Status.Conditions)
//		},
//		func(policy client.Object) bool {
//			return false
//		},
//		func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
//			p := policy.(*gwv1alpha3.BackendTLSPolicy)
//			return HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
//		},
//	)
//}

// ---------------------------- Common methods ----------------------------

//func toClientObjects[T client.Object](policies []T) []client.Object {
//	objects := make([]client.Object, 0)
//	for _, p := range policies {
//		p := p
//		objects = append(objects, p)
//	}
//
//	return objects
//}
//
//type isAcceptedFunc func(policy client.Object) bool
//type noDataFunc func(policy client.Object) bool
//type hasAccessFunc func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool
//
//func filterValidPolicies[T client.Object](
//	policies []T,
//	refGrants []*gwv1beta1.ReferenceGrant,
//	isAccepted isAcceptedFunc,
//	noData noDataFunc,
//	hasAccess hasAccessFunc,
//) []client.Object {
//	validPolicies := make([]client.Object, 0)
//	for _, p := range policies {
//		if !isAccepted(p) {
//			continue
//		}
//
//		if noData(p) {
//			continue
//		}
//
//		if !hasAccess(p, refGrants) {
//			continue
//		}
//
//		validPolicies = append(validPolicies, p)
//	}
//
//	return validPolicies
//}
