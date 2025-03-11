package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Test volume functions", func() {
	Context("Test GetVolumeSpec", func() {
		It("creates volume spec", func() {
			actual := GetVolumeSpec("-sidecar-config-")
			expected := corev1.Volume{
				Name: "sidecar-bootstrap-config-volume",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "-sidecar-config-",
					},
				}}
			Expect(actual).To(Equal(expected))
		})
	})
})
