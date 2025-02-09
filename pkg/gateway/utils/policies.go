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
