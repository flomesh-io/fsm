package routes

import (
	"fmt"

	"github.com/flomesh-io/fsm/pkg/logger"

	corev1 "k8s.io/api/core/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

var (
	log = logger.New("fsm-gateway/status/route")
)

type PolicyObjectReferenceConditionProvider struct {
	ancestorStatus status.PolicyAncestorStatusObject
}

func NewPolicyObjectReferenceConditionProvider(ancestorStatus status.PolicyAncestorStatusObject) *PolicyObjectReferenceConditionProvider {
	return &PolicyObjectReferenceConditionProvider{
		ancestorStatus: ancestorStatus,
	}
}

func (r *PolicyObjectReferenceConditionProvider) AddInvalidRefCondition(ref gwv1.ObjectReference) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("Unsupported group %s and kind %s for CA Certificate", ref.Group, ref.Kind),
	)
}

func (r *PolicyObjectReferenceConditionProvider) AddRefNotPermittedCondition(ref gwv1.ObjectReference) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("Reference to %s %s/%s is not allowed", ref.Kind, string(*ref.Namespace), ref.Name),
	)
}

func (r *PolicyObjectReferenceConditionProvider) AddRefNotFoundCondition(key types.NamespacedName, kind string) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonTargetNotFound,
		fmt.Sprintf("%s %s not found", kind, key.String()),
	)
}

func (r *PolicyObjectReferenceConditionProvider) AddGetRefErrorCondition(key types.NamespacedName, kind string, err error) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("Failed to get %s %s: %s", kind, key.String(), err),
	)
}

func (r *PolicyObjectReferenceConditionProvider) AddNoRequiredCAFileCondition(key types.NamespacedName, kind string) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("No required CA with key %s in %s %s", corev1.ServiceAccountRootCAKey, kind, key.String()),
	)
}

func (r *PolicyObjectReferenceConditionProvider) AddEmptyCACondition(ref gwv1.ObjectReference, refererNamespace string) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("CA Certificate is empty in %s %s/%s", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, refererNamespace), ref.Name),
	)
}

func (r *PolicyObjectReferenceConditionProvider) AddRefsResolvedCondition() {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionTrue,
		gwv1alpha2.PolicyReasonAccepted,
		"References resolved, policy is accepted",
	)
}

// ---

type RouteParentListenerConditionProvider struct {
	rps status.RouteParentStatusObject
}

func NewRouteParentListenerConditionProvider(rps status.RouteParentStatusObject) *RouteParentListenerConditionProvider {
	return &RouteParentListenerConditionProvider{
		rps: rps,
	}
}

func (r *RouteParentListenerConditionProvider) AddNoMatchingParentCondition(parentRef gwv1.ParentReference, routeNs string) {
	r.rps.AddCondition(
		gwv1.RouteConditionAccepted,
		metav1.ConditionFalse,
		gwv1.RouteReasonNoMatchingParent,
		fmt.Sprintf("No listeners match parent ref %s", types.NamespacedName{Namespace: gwutils.NamespaceDerefOr(parentRef.Namespace, routeNs), Name: string(parentRef.Name)}),
	)
}

func (r *RouteParentListenerConditionProvider) AddNotAllowedByListeners(parentRef gwv1.ParentReference, routeNs string) {
	r.rps.AddCondition(
		gwv1.RouteConditionAccepted,
		metav1.ConditionFalse,
		gwv1.RouteReasonNotAllowedByListeners,
		fmt.Sprintf("No matched listeners of parent ref %s", types.NamespacedName{Namespace: gwutils.NamespaceDerefOr(parentRef.Namespace, routeNs), Name: string(parentRef.Name)}),
	)
}
