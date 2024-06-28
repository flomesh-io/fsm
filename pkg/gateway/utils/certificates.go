package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

func ObjectRefToCACertificate(client cache.Cache, referer client.Object, ref gwv1.ObjectReference, ancestorStatus status.PolicyAncestorStatusObject) []byte {
	if !IsValidRefToGroupKindOfCA(ref) {
		addAncestorStatusCondition(
			ancestorStatus,
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf("Unsupported group %s and kind %s for CA Certificate", ref.Group, ref.Kind),
		)

		return nil
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     corev1.GroupName,
			Kind:      constants.KubernetesSecretKind,
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
		GetCARefGrants(client),
	) {
		addAncestorStatusCondition(
			ancestorStatus,
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf("Reference to %s %s/%s is not allowed", string(ref.Kind), string(*ref.Namespace), ref.Name),
		)

		return nil
	}

	ca := make([]byte, 0)
	switch ref.Kind {
	case constants.KubernetesSecretKind:
		getSecretFromCache := func(key types.NamespacedName) (*corev1.Secret, error) {
			obj := &corev1.Secret{}
			if err := client.Get(context.Background(), key, obj); err != nil {
				return nil, err
			}

			return obj, nil
		}

		secretKey := types.NamespacedName{
			Namespace: NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		}
		secret, err := getSecretFromCache(secretKey)
		if err != nil {
			if errors.IsNotFound(err) {
				addAncestorStatusCondition(
					ancestorStatus,
					gwv1alpha2.PolicyConditionAccepted,
					metav1.ConditionFalse,
					gwv1alpha2.PolicyReasonTargetNotFound,
					fmt.Sprintf("Secret %s not found", secretKey.String()),
				)
			} else {
				addAncestorStatusCondition(
					ancestorStatus,
					gwv1alpha2.PolicyConditionAccepted,
					metav1.ConditionFalse,
					gwv1alpha2.PolicyReasonInvalid,
					fmt.Sprintf("Failed to get Secret %s: %s", secretKey.String(), err),
				)
			}

			return nil
		}

		caBytes, ok := secret.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, caBytes...)
		} else {
			addAncestorStatusCondition(
				ancestorStatus,
				gwv1alpha2.PolicyConditionAccepted,
				metav1.ConditionFalse,
				gwv1alpha2.PolicyReasonInvalid,
				fmt.Sprintf("No required CA with key %s in Secret %s", corev1.ServiceAccountRootCAKey, secretKey.String()),
			)

			return nil
		}
	case constants.KubernetesConfigMapKind:
		getConfigMapFromCache := func(key types.NamespacedName) (*corev1.ConfigMap, error) {
			obj := &corev1.ConfigMap{}
			if err := client.Get(context.Background(), key, obj); err != nil {
				return nil, err
			}

			return obj, nil
		}

		cmKey := types.NamespacedName{
			Namespace: NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		}
		cm, err := getConfigMapFromCache(cmKey)
		if err != nil {
			if errors.IsNotFound(err) {
				addAncestorStatusCondition(
					ancestorStatus,
					gwv1alpha2.PolicyConditionAccepted,
					metav1.ConditionFalse,
					gwv1alpha2.PolicyReasonTargetNotFound,
					fmt.Sprintf("ConfigMap %s not found", cmKey.String()),
				)
			} else {
				addAncestorStatusCondition(
					ancestorStatus,
					gwv1alpha2.PolicyConditionAccepted,
					metav1.ConditionFalse,
					gwv1alpha2.PolicyReasonInvalid,
					fmt.Sprintf("Failed to get ConfigMap %s: %s", cmKey.String(), err),
				)
			}

			return nil
		}

		caBytes, ok := cm.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, []byte(caBytes)...)
		} else {
			addAncestorStatusCondition(
				ancestorStatus,
				gwv1alpha2.PolicyConditionAccepted,
				metav1.ConditionFalse,
				gwv1alpha2.PolicyReasonInvalid,
				fmt.Sprintf("No required CA with key %s in ConfigMap %s", corev1.ServiceAccountRootCAKey, cmKey.String()),
			)

			return nil
		}
	}

	if len(ca) == 0 {
		addAncestorStatusCondition(
			ancestorStatus,
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf("CA Certificate is empty in %s %s/%s", ref.Kind, NamespaceDerefOr(ref.Namespace, referer.GetNamespace()), ref.Name),
		)

		return nil
	}

	return ca
}

func addAncestorStatusCondition(ancestorStatus status.PolicyAncestorStatusObject, conditionType gwv1alpha2.PolicyConditionType, status metav1.ConditionStatus, reason gwv1alpha2.PolicyConditionReason, message string) {
	if ancestorStatus == nil {
		return
	}

	ancestorStatus.AddCondition(conditionType, status, reason, message)
}
