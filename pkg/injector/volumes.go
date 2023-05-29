package injector

import (
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeSpec returns a volume to add to the POD
func GetVolumeSpec(sidecarBootstrapConfigName string) corev1.Volume {
	return corev1.Volume{
		Name: SidecarBootstrapConfigVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: sidecarBootstrapConfigName,
			},
		},
	}
}
