package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func FindEndpointPort(ports []corev1.EndpointPort, svcPort corev1.ServicePort) int32 {
	for i, epPort := range ports {
		if svcPort.Name == "" {
			// port.Name is optional if there is only one port
			return epPort.Port
		}

		if svcPort.Name == epPort.Name {
			return epPort.Port
		}

		if i == len(ports)-1 && svcPort.TargetPort.Type == intstr.Int {
			return svcPort.TargetPort.IntVal
		}
	}

	return 0
}
