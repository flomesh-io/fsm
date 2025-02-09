package v2

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ---

type RouteRuleFilterPolicyProcessor struct {
	generator *ConfigGenerator
}

func NewRouteRuleFilterPolicyProcessor(c *ConfigGenerator) FilterPolicyProcessor {
	return &RouteRuleFilterPolicyProcessor{
		generator: c,
	}
}

func (p RouteRuleFilterPolicyProcessor) Process(route client.Object, routeParentRef gwv1.ParentReference, rule *gwv1.SectionName) []gwpav1alpha2.LocalFilterReference {
	if rule == nil {
		return nil
	}

	targetRef := gwpav1alpha2.LocalFilterPolicyTargetReference{
		Group: gwv1.GroupName,
		Kind:  gwv1.Kind(route.GetObjectKind().GroupVersionKind().Kind),
		Name:  gwv1.ObjectName(route.GetName()),
		Rule:  *rule,
	}

	policy, found := gwutils.FindRouteRuleFilterPolicy(p.generator.client, targetRef, route.GetNamespace())
	if !found {
		return nil
	}

	if !gwutils.IsPolicyAcceptedForAncestor(routeParentRef, policy.Status.Ancestors) {
		return nil
	}

	return policy.Spec.FilterRefs
}
