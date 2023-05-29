package injector

import (
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/models"
)

var errNoMatchingPort = errors.New("no matching port")

// RewriteHealthProbes rewrites health probes
func RewriteHealthProbes(pod *corev1.Pod) models.HealthProbes {
	probes := models.HealthProbes{}
	for idx := range pod.Spec.Containers {
		if probe := rewriteLiveness(&pod.Spec.Containers[idx]); probe != nil {
			probes.Liveness = probe
		}
		if probe := rewriteReadiness(&pod.Spec.Containers[idx]); probe != nil {
			probes.Readiness = probe
		}
		if probe := rewriteStartup(&pod.Spec.Containers[idx]); probe != nil {
			probes.Startup = probe
		}
	}
	return probes
}

func rewriteLiveness(container *corev1.Container) *models.HealthProbe {
	return rewriteProbe(container.LivenessProbe, "liveness", constants.LivenessProbePath, constants.LivenessProbePort, &container.Ports)
}

func rewriteReadiness(container *corev1.Container) *models.HealthProbe {
	return rewriteProbe(container.ReadinessProbe, "readiness", constants.ReadinessProbePath, constants.ReadinessProbePort, &container.Ports)
}

func rewriteStartup(container *corev1.Container) *models.HealthProbe {
	return rewriteProbe(container.StartupProbe, "startup", constants.StartupProbePath, constants.StartupProbePort, &container.Ports)
}

func rewriteProbe(probe *corev1.Probe, probeType, path string, port int32, containerPorts *[]corev1.ContainerPort) *models.HealthProbe {
	if probe == nil {
		return nil
	}

	originalProbe := &models.HealthProbe{}
	var newPath string
	var definedPort *intstr.IntOrString
	var isHTTPS bool
	if probe.HTTPGet != nil {
		definedPort = &probe.HTTPGet.Port
		originalProbe.IsHTTP = len(probe.HTTPGet.Scheme) == 0 || probe.HTTPGet.Scheme == corev1.URISchemeHTTP
		originalProbe.IsTCPSocket = probe.HTTPGet.Scheme == corev1.URISchemeHTTPS
		originalProbe.Path = probe.HTTPGet.Path
		probe.HTTPGet.Path = constants.HealthcheckPath
		if originalProbe.IsHTTP {
			newPath = probe.HTTPGet.Path
		} else if originalProbe.IsTCPSocket {
			probe.HTTPGet.Scheme = corev1.URISchemeHTTP
			isHTTPS = true
		}
	} else if probe.TCPSocket != nil {
		// Transform the TCPSocket probe into a HttpGet probe
		originalProbe.IsTCPSocket = true
		probe.HTTPGet = &corev1.HTTPGetAction{
			Port:        probe.TCPSocket.Port,
			Path:        constants.HealthcheckPath,
			HTTPHeaders: []corev1.HTTPHeader{},
		}
		newPath = probe.HTTPGet.Path
		definedPort = &probe.HTTPGet.Port
		port = constants.HealthcheckPort
		probe.TCPSocket = nil
	} else {
		return nil
	}

	var err error
	originalProbe.Port, err = getPort(*definedPort, containerPorts)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding a matching port for %+v on container %+v", *definedPort, containerPorts)
	}
	if originalProbe.IsTCPSocket {
		probePort := originalProbe.Port
		if probePort == 0 {
			if isHTTPS {
				probePort = 443
			}
		}
		probe.HTTPGet.HTTPHeaders = append(probe.HTTPGet.HTTPHeaders, corev1.HTTPHeader{Name: "Original-Tcp-Port", Value: fmt.Sprint(probePort)})
	} else if originalProbe.IsHTTP {
		probePort := originalProbe.Port
		if probePort == 0 {
			probePort = 80
		}
		probe.HTTPGet.HTTPHeaders = append(probe.HTTPGet.HTTPHeaders, corev1.HTTPHeader{Name: "Original-Http-Port", Value: fmt.Sprint(probePort)})
		probe.HTTPGet.HTTPHeaders = append(probe.HTTPGet.HTTPHeaders, corev1.HTTPHeader{Name: "Original-Http-Path", Value: originalProbe.Path})
	}
	*definedPort = intstr.IntOrString{Type: intstr.Int, IntVal: port}
	originalProbe.Timeout = time.Duration(probe.TimeoutSeconds) * time.Second

	log.Debug().Msgf(
		"Rewriting %s probe (:%d%s) to :%d%s",
		probeType,
		originalProbe.Port, originalProbe.Path,
		port, newPath,
	)

	return originalProbe
}

// getPort returns the int32 of an IntOrString port; It looks for port's name matches in the full list of container ports
func getPort(namedPort intstr.IntOrString, containerPorts *[]corev1.ContainerPort) (int32, error) {
	// Maybe this is not a named port
	intPort := int32(namedPort.IntValue())
	if intPort != 0 {
		return intPort, nil
	}

	if containerPorts == nil {
		return 0, errNoMatchingPort
	}

	// Find an integer match for the name of the port in the list of container ports
	portName := namedPort.String()
	for _, p := range *containerPorts {
		if p.Name != "" && p.Name == portName {
			return p.ContainerPort, nil
		}
	}

	return 0, errNoMatchingPort
}
