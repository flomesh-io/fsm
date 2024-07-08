package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetServicePort returns the ServicePort for the given port.
func GetServicePort(svc *corev1.Service, port *int32) (corev1.ServicePort, error) {
	if port == nil && len(svc.Spec.Ports) == 1 {
		return svc.Spec.Ports[0], nil
	}

	if port != nil {
		for _, p := range svc.Spec.Ports {
			if p.Port == *port {
				return p, nil
			}
		}
	}

	return corev1.ServicePort{}, fmt.Errorf("no matching port for Service %s and port %d", svc.Name, port)
}

// GetDefaultPort returns the default port for the given ServicePort.
func GetDefaultPort(svcPort corev1.ServicePort) int32 {
	if svcPort.TargetPort.Type == intstr.Int && svcPort.TargetPort.IntVal != 0 {
		return svcPort.TargetPort.IntVal
	}

	return svcPort.Port
}
