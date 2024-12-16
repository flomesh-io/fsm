package driver

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/driver"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/bootstrap"
)

func getPlatformSpecificSpecComponents(injCtx *driver.InjectorContext, cfg configurator.Configurator, pod *corev1.Pod) (podSecurityContext *corev1.SecurityContext, pipyContainer string) {
	podSecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer.BoolPtr(false),
		RunAsUser: func() *int64 {
			uid := constants.SidecarUID
			return &uid
		}(),
	}

	if podAnnotations := pod.GetAnnotations(); len(podAnnotations) > 0 {
		if podSidecarImage, exists := podAnnotations[constants.SidecarImageAnnotation]; exists {
			if len(podSidecarImage) > 0 {
				pipyContainer = podSidecarImage
				return
			}
		}
	}

	if ns, err := injCtx.KubeClient.CoreV1().Namespaces().Get(context.Background(), injCtx.PodNamespace, metav1.GetOptions{}); err == nil {
		if nsAnnotations := ns.GetAnnotations(); len(nsAnnotations) > 0 {
			if nsSidecarImage, exists := nsAnnotations[constants.SidecarImageAnnotation]; exists {
				if len(nsSidecarImage) > 0 {
					pipyContainer = nsSidecarImage
					return
				}
			}
		}
	}

	pipyContainer = cfg.GetSidecarImage()
	return
}

func getPipySidecarContainerSpec(
	injCtx *driver.InjectorContext,
	pod *corev1.Pod,
	cfg configurator.Configurator,
	cnPrefix string,
	originalHealthProbes models.HealthProbes,
	podOS string) (corev1.Container, bool) {
	securityContext, containerImage := getPlatformSpecificSpecComponents(injCtx, cfg, pod)

	podControllerKind := ""
	podControllerName := ""
	for _, ref := range pod.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			podControllerKind = ref.Kind
			podControllerName = ref.Name
			break
		}
	}
	// Assume ReplicaSets are controlled by a Deployment unless their names
	// do not contain a hyphen. This aligns with the behavior of the
	// Prometheus config in the FSM Helm chart.
	if podControllerKind == "ReplicaSet" {
		if hyp := strings.LastIndex(podControllerName, "-"); hyp >= 0 {
			podControllerKind = "Deployment"
			podControllerName = podControllerName[:hyp]
		}
	}

	repoServerIPAddr := cfg.GetRepoServerIPAddr()
	if strings.HasPrefix(repoServerIPAddr, "127.") || strings.EqualFold(strings.ToLower(repoServerIPAddr), "localhost") {
		repoServerIPAddr = fmt.Sprintf("%s.%s", constants.FSMControllerName, injCtx.FsmNamespace)
	}

	var repoServer string
	if len(cfg.GetRepoServerCodebase()) > 0 {
		repoServer = fmt.Sprintf("%s://%s:%v/repo/%s/fsm-sidecar/%s/",
			constants.ProtocolHTTP, repoServerIPAddr, cfg.GetProxyServerPort(), cfg.GetRepoServerCodebase(), cnPrefix)
	} else {
		repoServer = fmt.Sprintf("%s://%s:%v/repo/fsm-sidecar/%s/",
			constants.ProtocolHTTP, repoServerIPAddr, cfg.GetProxyServerPort(), cnPrefix)
	}

	var nsAnnotations, podAnnotations map[string]string
	podAnnotations = pod.GetAnnotations()
	if ns, err := injCtx.KubeClient.CoreV1().Namespaces().Get(context.Background(), injCtx.PodNamespace, metav1.GetOptions{}); err == nil {
		nsAnnotations = ns.GetAnnotations()
	}

	sidecarContainer := corev1.Container{
		Name:            constants.SidecarContainerName,
		Image:           containerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: securityContext,
		Ports:           getPipyContainerPorts(originalHealthProbes),
		VolumeMounts: []corev1.VolumeMount{{
			Name:      injector.SidecarBootstrapConfigVolume,
			ReadOnly:  true,
			MountPath: bootstrap.PipyProxyConfigPath,
		}},
		Resources: getPipySidecarResource(cfg, nsAnnotations, podAnnotations),
		Args: []string{
			"pipy",
			fmt.Sprintf("--log-level=%s", injCtx.Configurator.GetSidecarLogLevel()),
			fmt.Sprintf("--admin-port=%d", cfg.GetProxyServerPort()),
			repoServer,
		},
		Env: []corev1.EnvVar{
			{
				Name:  "MESH_NAME",
				Value: injCtx.MeshName,
			},
			{
				Name: "POD_UID",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.uid",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: "SERVICE_ACCOUNT",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.serviceAccountName",
					},
				},
			},
			{
				Name:  "POD_CONTROLLER_KIND",
				Value: podControllerKind,
			},
			{
				Name:  "POD_CONTROLLER_NAME",
				Value: podControllerName,
			},
		},
	}

	if injCtx.Configurator.IsLocalDNSProxyEnabled() {
		if fsmControllerSvc, err := getFSMControllerSvc(injCtx.KubeClient, injCtx.FsmNamespace); err == nil {
			pod.Spec.HostAliases = append(pod.Spec.HostAliases, corev1.HostAlias{
				IP:        fsmControllerSvc.Spec.ClusterIP,
				Hostnames: []string{fmt.Sprintf("%s.%s", constants.FSMControllerName, injCtx.FsmNamespace)},
			})

			pod.Spec.DNSPolicy = "None"
			trustDomain := injCtx.CertManager.GetTrustDomain()
			dots := fmt.Sprintf("%d", len(strings.Split(trustDomain, `.`))+3)
			searches := make([]string, 0)
			if len(pod.Namespace) > 0 {
				searches = append(searches, fmt.Sprintf("%s.svc.%s", pod.Namespace, trustDomain))
			} else if len(injCtx.PodNamespace) > 0 {
				searches = append(searches, fmt.Sprintf("%s.svc.%s", injCtx.PodNamespace, trustDomain))
			}

			searches = append(searches, fmt.Sprintf("svc.%s", trustDomain))
			searches = append(searches, trustDomain)

			pod.Spec.DNSConfig = &corev1.PodDNSConfig{
				Nameservers: []string{fsmControllerSvc.Spec.ClusterIP},
				Searches:    searches,
				Options: []corev1.PodDNSConfigOption{
					{Name: "ndots", Value: &dots},
				},
			}
		}
	}

	holdApp := isAnnotatedForHoldApplication(cfg, nsAnnotations, podAnnotations)
	holdProxy := isAnnotatedForHoldProxy(cfg, nsAnnotations, podAnnotations)
	if holdApp || holdProxy {
		lifecycle := new(corev1.Lifecycle)
		if holdApp {
			lifecycle.PostStart = &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						`sh`,
						`-c`,
						`until [ $(curl -s -o /dev/null -I -w "%{http_code}" http://127.0.0.1:15000/config_dump) -eq 200 ]; do sleep 5; done;`,
					},
				},
			}
		}
		if holdProxy {
			lifecycle.PreStop = &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						`sh`,
						`-c`,
						`kill -SIGTERM 1`,
					},
				},
			}
		}
		sidecarContainer.Lifecycle = lifecycle
	}

	return sidecarContainer, holdApp
}

func getPipyContainerPorts(originalHealthProbes models.HealthProbes) []corev1.ContainerPort {
	containerPorts := []corev1.ContainerPort{
		{
			Name:          constants.SidecarAdminPortName,
			ContainerPort: constants.SidecarAdminPort,
		},
		{
			Name:          constants.SidecarInboundListenerPortName,
			ContainerPort: constants.SidecarInboundListenerPort,
		},
		{
			Name:          constants.SidecarInboundPrometheusListenerPortName,
			ContainerPort: constants.SidecarPrometheusInboundListenerPort,
		},
	}

	if originalHealthProbes.Liveness != nil {
		livenessPort := corev1.ContainerPort{
			// Name must be no more than 15 characters
			Name:          "liveness-port",
			ContainerPort: constants.LivenessProbePort,
		}
		containerPorts = append(containerPorts, livenessPort)
	}

	if originalHealthProbes.Readiness != nil {
		readinessPort := corev1.ContainerPort{
			// Name must be no more than 15 characters
			Name:          "readiness-port",
			ContainerPort: constants.ReadinessProbePort,
		}
		containerPorts = append(containerPorts, readinessPort)
	}

	if originalHealthProbes.Startup != nil {
		startupPort := corev1.ContainerPort{
			// Name must be no more than 15 characters
			Name:          "startup-port",
			ContainerPort: constants.StartupProbePort,
		}
		containerPorts = append(containerPorts, startupPort)
	}

	return containerPorts
}

// getFSMControllerSvc returns the fsm-controller service.
// The pod name is inferred from the 'CONTROLLER_SVC_NAME' env variable which is set during deployment.
func getFSMControllerSvc(kubeClient kubernetes.Interface, fsmNamespace string) (*corev1.Service, error) {
	svcName := os.Getenv("CONTROLLER_SVC_NAME")
	if svcName == "" {
		return nil, fmt.Errorf("CONTROLLER_SVC_NAME env variable cannot be empty")
	}

	svc, err := kubeClient.CoreV1().Services(fsmNamespace).Get(context.TODO(), svcName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingControllerSvc)).
			Msgf("Error retrieving fsm-controller service %s", svcName)
		return nil, err
	}

	return svc, nil
}

func getPipySidecarResource(cfg configurator.Configurator, nsAnnotations, podAnnotations map[string]string) corev1.ResourceRequirements {
	cfgResources := cfg.GetProxyResources()
	resources := corev1.ResourceRequirements{}
	if cfgResources.Limits != nil {
		resources.Limits = make(corev1.ResourceList)
		for k, v := range cfgResources.Limits {
			resources.Limits[k] = v
		}
	}
	if cfgResources.Requests != nil {
		resources.Requests = make(corev1.ResourceList)
		for k, v := range cfgResources.Requests {
			resources.Requests[k] = v
		}
	}

	resourceNames := []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceStorage, corev1.ResourceEphemeralStorage}
	for _, resourceName := range resourceNames {
		podResourceLimitsExist := false
		resourceLimitsAnnotation := fmt.Sprintf("%s-%s", constants.SidecarResourceLimitsAnnotationPrefix, resourceName)
		if len(podAnnotations) > 0 {
			if resourceLimits, exists := podAnnotations[resourceLimitsAnnotation]; exists {
				if resources.Limits == nil {
					resources.Limits = make(corev1.ResourceList)
				}
				if quantity, quantityErr := resource.ParseQuantity(resourceLimits); quantityErr == nil {
					resources.Limits[resourceName] = quantity
					podResourceLimitsExist = true
				} else {
					log.Error().Err(quantityErr)
				}
			}
		}
		if !podResourceLimitsExist && len(nsAnnotations) > 0 {
			if resourceLimits, exists := nsAnnotations[resourceLimitsAnnotation]; exists {
				if resources.Limits == nil {
					resources.Limits = make(corev1.ResourceList)
				}
				if quantity, quantityErr := resource.ParseQuantity(resourceLimits); quantityErr == nil {
					resources.Limits[resourceName] = quantity
				} else {
					log.Error().Err(quantityErr)
				}
			}
		}
	}

	for _, resourceName := range resourceNames {
		podResourceRequestsExist := false
		resourceRequestAnnotation := fmt.Sprintf("%s-%s", constants.SidecarResourceRequestsAnnotationPrefix, resourceName)
		if len(podAnnotations) > 0 {
			if resourceRequests, exists := podAnnotations[resourceRequestAnnotation]; exists {
				if resources.Requests == nil {
					resources.Requests = make(corev1.ResourceList)
				}
				if quantity, quantityErr := resource.ParseQuantity(resourceRequests); quantityErr == nil {
					resources.Requests[resourceName] = quantity
					podResourceRequestsExist = true
				} else {
					log.Error().Err(quantityErr)
				}
			}
		}
		if !podResourceRequestsExist && len(nsAnnotations) > 0 {
			if resourceRequests, exists := nsAnnotations[resourceRequestAnnotation]; exists {
				if resources.Requests == nil {
					resources.Requests = make(corev1.ResourceList)
				}
				if quantity, quantityErr := resource.ParseQuantity(resourceRequests); quantityErr == nil {
					resources.Requests[resourceName] = quantity
				} else {
					log.Error().Err(quantityErr)
				}
			}
		}
	}
	return resources
}

func isAnnotatedForHoldApplication(cfg configurator.Configurator, nsAnnotations, podAnnotations map[string]string) (enabled bool) {
	holdApp, exists := podAnnotations[constants.HoldApplicationUntilProxyStartsAnnotation]
	if !exists {
		holdApp, exists = nsAnnotations[constants.HoldApplicationUntilProxyStartsAnnotation]
		if !exists {
			return cfg.IsHoldApplicationUntilProxyStarts()
		}
	}

	switch strings.ToLower(holdApp) {
	case "enabled", "yes", "true":
		enabled = true
	case "disabled", "no", "false":
		enabled = false
	default:
		log.Error().Msgf("invalid annotation value for key %q: %s", constants.HoldApplicationUntilProxyStartsAnnotation, holdApp)
	}
	return
}

func isAnnotatedForHoldProxy(cfg configurator.Configurator, nsAnnotations, podAnnotations map[string]string) (enabled bool) {
	holdProxy, exists := podAnnotations[constants.GracefulExitUntilDownstreamEndsAnnotation]
	if !exists {
		holdProxy, exists = nsAnnotations[constants.GracefulExitUntilDownstreamEndsAnnotation]
		if !exists {
			return cfg.IsGracefulExitUntilDownstreamEnds()
		}
	}

	switch strings.ToLower(holdProxy) {
	case "enabled", "yes", "true":
		enabled = true
	case "disabled", "no", "false":
		enabled = false
	default:
		log.Error().Msgf("invalid annotation value for key %q: %s", constants.GracefulExitUntilDownstreamEndsAnnotation, holdProxy)
	}
	return
}
