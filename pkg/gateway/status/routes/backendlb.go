package routes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"github.com/flomesh-io/fsm/pkg/gateway/status/policies"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (p *RouteStatusProcessor) computeBackendLBPolicyStatus(route client.Object, backendRef gwv1.BackendObjectReference, _ *fgwv2.ServicePortName, routeParentRef gwv1.ParentReference) {
	targetRef := gwv1alpha2.LocalPolicyTargetReference{
		Group: ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Name:  backendRef.Name,
	}

	policy, found := gwutils.FindBackendLBPolicy(p.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	psu := policies.NewPolicyStatusUpdateWithLocalPolicyTargetReference(
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
