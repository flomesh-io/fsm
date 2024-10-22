package routes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

type PolicyObjectReferenceResolver struct {
	ancestorStatus status.PolicyAncestorStatusObject
}

func NewPolicyObjectReferenceResolver(ancestorStatus status.PolicyAncestorStatusObject) *PolicyObjectReferenceResolver {
	return &PolicyObjectReferenceResolver{
		ancestorStatus: ancestorStatus,
	}
}

func (r *PolicyObjectReferenceResolver) AddInvalidRefCondition(ref gwv1.ObjectReference) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("Unsupported group %s and kind %s for CA Certificate", ref.Group, ref.Kind),
	)
}

func (r *PolicyObjectReferenceResolver) AddRefNotPermittedCondition(ref gwv1.ObjectReference) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("Reference to %s %s/%s is not allowed", ref.Kind, string(*ref.Namespace), ref.Name),
	)
}

func (r *PolicyObjectReferenceResolver) AddRefNotFoundCondition(key types.NamespacedName, kind string) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonTargetNotFound,
		fmt.Sprintf("%s %s not found", kind, key.String()),
	)
}

func (r *PolicyObjectReferenceResolver) AddGetRefErrorCondition(key types.NamespacedName, kind string, err error) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("Failed to get %s %s: %s", kind, key.String(), err),
	)
}

func (r *PolicyObjectReferenceResolver) AddNoRequiredCAFileCondition(key types.NamespacedName, kind string) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("No required CA with key %s in %s %s", corev1.ServiceAccountRootCAKey, kind, key.String()),
	)
}

func (r *PolicyObjectReferenceResolver) AddEmptyCACondition(ref gwv1.ObjectReference, refererNamespace string) {
	r.ancestorStatus.AddCondition(
		gwv1alpha2.PolicyConditionAccepted,
		metav1.ConditionFalse,
		gwv1alpha2.PolicyReasonInvalid,
		fmt.Sprintf("CA Certificate is empty in %s %s/%s", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, refererNamespace), ref.Name),
	)
}
