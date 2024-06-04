package policy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// notFoundCondition returns the not found condition with the given message for the policy
func notFoundCondition(message string) metav1.Condition {
	return metav1.Condition{
		Type:    string(gwv1alpha2.PolicyConditionAccepted),
		Status:  metav1.ConditionFalse,
		Reason:  string(gwv1alpha2.PolicyReasonTargetNotFound),
		Message: message,
	}
}

// invalidCondition returns the invalid condition with the given message for the policy
func invalidCondition(message string) metav1.Condition {
	return metav1.Condition{
		Type:    string(gwv1alpha2.PolicyConditionAccepted),
		Status:  metav1.ConditionFalse,
		Reason:  string(gwv1alpha2.PolicyReasonInvalid),
		Message: message,
	}
}

// conflictCondition returns the conflict condition with the given message for the policy
func conflictCondition(message string) metav1.Condition {
	return metav1.Condition{
		Type:    string(gwv1alpha2.PolicyConditionAccepted),
		Status:  metav1.ConditionFalse,
		Reason:  string(gwv1alpha2.PolicyReasonConflicted),
		Message: message,
	}
}

// acceptedCondition returns the accepted condition with the given message for the policy
func acceptedCondition() metav1.Condition {
	return metav1.Condition{
		Type:    string(gwv1alpha2.PolicyConditionAccepted),
		Status:  metav1.ConditionTrue,
		Reason:  string(gwv1alpha2.PolicyReasonAccepted),
		Message: string(gwv1alpha2.PolicyReasonAccepted),
	}
}

// noAccessCondition returns the no access condition with the given message for the policy
func noAccessCondition(message string) metav1.Condition {
	return metav1.Condition{
		Type:    string(gwv1alpha2.PolicyConditionAccepted),
		Status:  metav1.ConditionFalse,
		Reason:  "NoAccessToTarget",
		Message: message,
	}
}
