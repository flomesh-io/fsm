package routes

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"github.com/flomesh-io/fsm/pkg/gateway/status/policies"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (p *RouteStatusProcessor) computeRouteRuleFilterPolicyStatus(route client.Object, rule *gwv1.SectionName, routeParentRef gwv1.ParentReference) {
	if rule == nil {
		return
	}

	targetRef := gwpav1alpha2.LocalFilterPolicyTargetReference{
		Group: gwv1.GroupName,
		Kind:  gwv1.Kind(route.GetObjectKind().GroupVersionKind().Kind),
		Name:  gwv1.ObjectName(route.GetName()),
		Rule:  *rule,
	}

	policy, found := gwutils.FindRouteRuleFilterPolicy(p.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	psu := policies.NewPolicyStatusUpdateWithLocalFilterPolicyTargetReference(
		policy,
		policy.GroupVersionKind(),
		policy.Spec.TargetRefs,
		gwutils.ToSlicePtr(policy.Status.Ancestors),
	)

	ancestorStatus := psu.StatusUpdateFor(routeParentRef)
	defer func() {
		p.statusUpdater.Send(status.Update{
			Resource:       psu.GetResource(),
			NamespacedName: psu.GetFullName(),
			Mutator:        psu,
		})
	}()

	if !ancestorStatus.ConditionExists(gwv1alpha2.PolicyConditionAccepted) {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionTrue,
			gwv1alpha2.PolicyReasonAccepted,
			fmt.Sprintf("Policy is accepted for ancestor %s/%s", gwutils.NamespaceDerefOr(routeParentRef.Namespace, route.GetNamespace()), routeParentRef.Name),
		)
	}
}
