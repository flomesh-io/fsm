package routes

import (
	"fmt"
	"strings"

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

func (p *RouteStatusProcessor) computeBackendTLSPolicyStatus(route client.Object, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName, routeParentRef gwv1.ParentReference, backendTLSFn func(bool)) {
	targetRef := gwv1alpha2.LocalPolicyTargetReferenceWithSectionName{
		LocalPolicyTargetReference: gwv1alpha2.LocalPolicyTargetReference{
			Group: ptr.Deref(backendRef.Group, corev1.GroupName),
			Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
			Name:  backendRef.Name,
		},
		SectionName: ptr.To(gwv1alpha2.SectionName(svcPort.SectionName)),
	}

	policy, found := gwutils.FindBackendTLSPolicy(p.client, targetRef, route.GetNamespace())
	if !found {
		backendTLSFn(found)
		return
	}

	psu := policies.NewPolicyStatusUpdateWithLocalPolicyTargetReferenceWithSectionName(
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

	if policy.Spec.Validation.WellKnownCACertificates != nil && *policy.Spec.Validation.WellKnownCACertificates != "" {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			".spec.validation.wellKnownCACertificates is unsupported.",
		)
		return
	}

	hostname := string(policy.Spec.Validation.Hostname)
	if err := gwutils.IsValidHostname(hostname); err != nil {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf(".spec.validation.hostname %q is invalid. Hostname must be a valid RFC 1123 fully qualified domain name.", hostname),
		)

		return
	}

	if strings.Contains(hostname, "*") {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf(".spec.validation.hostname %q is invalid. Wildcard domains and numeric IP addresses are not allowed", hostname),
		)

		return
	}

	refs := make([]gwv1.ObjectReference, 0)
	for _, ref := range policy.Spec.Validation.CACertificateRefs {
		refs = append(refs, gwv1.ObjectReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Namespace: ptr.To(gwv1.Namespace(policy.Namespace)),
			Name:      ref.Name,
		})
	}

	resolver := gwutils.NewObjectReferenceResolver(NewPolicyObjectReferenceConditionProvider(ancestorStatus), p.client)
	if !resolver.ResolveAllRefs(policy, refs) {
		return
	}

	if !ancestorStatus.ConditionExists(gwv1alpha2.PolicyConditionAccepted) {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionTrue,
			gwv1alpha2.PolicyReasonAccepted,
			fmt.Sprintf("Policy is accepted for ancestor %s/%s", gwutils.NamespaceDerefOr(routeParentRef.Namespace, route.GetNamespace()), routeParentRef.Name),
		)
	}
}
