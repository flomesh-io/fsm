package injector

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/configurator"
)

// GetInitContainerSpec returns the spec of init container.
func GetInitContainerSpec(containerName string, cfg configurator.Configurator, outboundIPRangeExclusionList []string,
	outboundIPRangeInclusionList []string, outboundPortExclusionList []int,
	inboundPortExclusionList []int, enablePrivilegedInitContainer bool, pullPolicy corev1.PullPolicy, networkInterfaceExclusionList []string) corev1.Container {
	proxyMode := cfg.GetMeshConfig().Spec.Sidecar.LocalProxyMode
	enabledDNSProxy := cfg.IsLocalDNSProxyEnabled()
	iptablesInitCommand := GenerateIptablesCommands(proxyMode, enabledDNSProxy, outboundIPRangeExclusionList, outboundIPRangeInclusionList, outboundPortExclusionList, inboundPortExclusionList, networkInterfaceExclusionList)

	return corev1.Container{
		Name:            containerName,
		Image:           cfg.GetInitContainerImage(),
		ImagePullPolicy: pullPolicy,
		SecurityContext: &corev1.SecurityContext{
			Privileged: &enablePrivilegedInitContainer,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
			RunAsNonRoot: pointer.BoolPtr(false),
			// User ID 0 corresponds to root
			RunAsUser: pointer.Int64Ptr(0),
		},
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			iptablesInitCommand,
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.podIP",
					},
				},
			},
		},
	}
}
