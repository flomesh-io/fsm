package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("ConfigurePodImagePullSecrets", func() {
	It("appends configured secrets while preserving existing entries and order", func() {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "application-secret"},
				},
			},
		}

		ConfigurePodImagePullSecrets(pod, []corev1.LocalObjectReference{
			{Name: "sidecar-secret"},
			{Name: "application-secret"},
			{Name: ""},
			{Name: "sidecar-secret"},
			{Name: "fallback-secret"},
		})

		Expect(pod.Spec.ImagePullSecrets).To(Equal([]corev1.LocalObjectReference{
			{Name: "application-secret"},
			{Name: "sidecar-secret"},
			{Name: "fallback-secret"},
		}))
	})

	It("does not change the pod when no secrets are configured", func() {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				ImagePullSecrets: []corev1.LocalObjectReference{
					{Name: "application-secret"},
				},
			},
		}

		ConfigurePodImagePullSecrets(pod, nil)

		Expect(pod.Spec.ImagePullSecrets).To(Equal([]corev1.LocalObjectReference{
			{Name: "application-secret"},
		}))
	})
})
