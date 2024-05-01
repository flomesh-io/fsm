package status

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// NotFoundCondition returns the not found condition with the given message for the policy
func NotFoundCondition(policy client.Object, message string) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1alpha2.PolicyReasonTargetNotFound),
		Message:            message,
	}
}

// InvalidCondition returns the invalid condition with the given message for the policy
func InvalidCondition(policy client.Object, message string) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1alpha2.PolicyReasonInvalid),
		Message:            message,
	}
}

// ConflictCondition returns the conflict condition with the given message for the policy
func ConflictCondition(policy client.Object, message string) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1alpha2.PolicyReasonConflicted),
		Message:            message,
	}
}

// AcceptedCondition returns the accepted condition with the given message for the policy
func AcceptedCondition(policy client.Object) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1alpha2.PolicyReasonAccepted),
		Message:            string(gwv1alpha2.PolicyReasonAccepted),
	}
}

// NoAccessCondition returns the no access condition with the given message for the policy
func NoAccessCondition(policy client.Object, message string) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "NoAccessToTarget",
		Message:            message,
	}
}

// ConditionPointer returns the pointer of the given condition
func ConditionPointer(condition metav1.Condition) *metav1.Condition {
	return &condition
}
