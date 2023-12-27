package utils

import corev1 "k8s.io/api/core/v1"

// IsPodStatusConditionTrue returns true if the pod is ready.
func IsPodStatusConditionTrue(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) bool {
	return IsPodStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionTrue)
}

// IsPodStatusConditionPresentAndEqual returns true if the pod has the given condition and status.
func IsPodStatusConditionPresentAndEqual(conditions []corev1.PodCondition, conditionType corev1.PodConditionType, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}
