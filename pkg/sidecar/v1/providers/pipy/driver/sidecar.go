package driver

import (
	"context"
	"os"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/driver"
	registry2 "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/registry"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/repo"
)

// PipySidecarDriver is the pipy sidecar driver
type PipySidecarDriver struct {
}

// Start is the implement for ControllerDriver.Start
func (sd PipySidecarDriver) Start(ctx context.Context) (health.Probes, error) {
	parentCtx := ctx.Value(&fctx.ControllerCtxKey)
	if parentCtx == nil {
		return nil, errors.New("missing Controller Context")
	}
	ctrlCtx := parentCtx.(*fctx.ControllerContext)
	cfg := ctrlCtx.Configurator
	certManager := ctrlCtx.CertManager
	k8sClient := ctrlCtx.MeshCatalog.GetKubeController()
	proxyServerPort := ctrlCtx.ProxyServerPort
	proxyServiceCert := ctrlCtx.ProxyServiceCert

	proxyMapper := &registry2.KubeProxyServiceMapper{KubeController: k8sClient}
	proxyRegistry := registry2.NewProxyRegistry(proxyMapper, ctrlCtx.MsgBroker)
	go proxyRegistry.ReleaseCertificateHandler(certManager, ctrlCtx.Stop)
	// Create and start the pipy repo http service
	repoServer := repo.NewRepoServer(ctrlCtx.MeshCatalog, proxyRegistry, ctrlCtx.FsmNamespace, cfg, certManager, k8sClient, ctrlCtx.MsgBroker)
	return repoServer, repoServer.Start(proxyServerPort, proxyServiceCert)
}

// Patch is the implement for InjectorDriver.Patch
func (sd PipySidecarDriver) Patch(ctx context.Context) error {
	parentCtx := ctx.Value(&driver.InjectorCtxKey)
	if parentCtx == nil {
		return errors.New("missing Injector Context")
	}
	injCtx := parentCtx.(*driver.InjectorContext)
	if injCtx.Configurator.GetTrafficInterceptionMode() == constants.TrafficInterceptionModeNodeLevel {
		return nil
	}

	configurator := injCtx.Configurator
	fsmNamespace := injCtx.FsmNamespace
	fsmContainerPullPolicy := injCtx.FsmContainerPullPolicy
	namespace := injCtx.PodNamespace
	pod := injCtx.Pod
	podOS := injCtx.PodOS
	proxyUUID := injCtx.ProxyUUID
	bootstrapCertificate := injCtx.BootstrapCertificate
	cnPrefix := injCtx.BootstrapCertificateCNPrefix
	dryRun := injCtx.DryRun

	originalHealthProbes := injector.RewriteHealthProbes(pod)

	// Create the bootstrap configuration for the Pipy proxy for the given pod
	pipyBootstrapConfigName := injector.BootstrapSecretPrefix + proxyUUID.String()

	// This needs to occur before replacing the label below.
	originalUUID, alreadyInjected := injector.GetProxyUUID(pod)
	switch {
	case dryRun:
		// The webhook has a side effect (making out-of-band changes) of creating k8s secret
		// corresponding to the Pipy bootstrap config. Such a side effect needs to be skipped
		// when the request is a DryRun.
		// Ref: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#side-effects
		log.Debug().Msgf("Skipping pipy bootstrap config creation for dry-run request: service-account=%s, namespace=%s", pod.Spec.ServiceAccountName, namespace)
	case alreadyInjected:
		// Pod definitions can be copied via the `kubectl debug` command, which can lead to a pod being created that
		// has already had injection occur. We could simply do nothing and return early, but that would leave 2 pods
		// with the same UUID, so instead we change the UUID, and create a new bootstrap config, copied from the original,
		// with the proxy UUID changed.
		oldConfigName := injector.BootstrapSecretPrefix + originalUUID
		if _, err := createPipyBootstrapFromExisting(injCtx, pipyBootstrapConfigName, oldConfigName, namespace, bootstrapCertificate); err != nil {
			log.Error().Err(err).Msgf("Failed to create Pipy bootstrap config for already-injected pod: service-account=%s, namespace=%s, certificate CN prefix=%s", pod.Spec.ServiceAccountName, namespace, cnPrefix)
			return err
		}
	default:
		if _, err := createPipyBootstrapConfig(injCtx, pipyBootstrapConfigName, namespace, fsmNamespace, bootstrapCertificate, originalHealthProbes); err != nil {
			log.Error().Err(err).Msgf("Failed to create Pipy bootstrap config for pod: service-account=%s, namespace=%s, certificate CN prefix=%s", pod.Spec.ServiceAccountName, namespace, cnPrefix)
			return err
		}
	}

	if alreadyInjected {
		// replace the volume and we're done.
		for i, volume := range pod.Spec.Volumes {
			// It should be the last, but we check all for posterity.
			if volume.Name == injector.SidecarBootstrapConfigVolume {
				pod.Spec.Volumes[i] = injector.GetVolumeSpec(pipyBootstrapConfigName)
				break
			}
		}
		return nil
	}

	// Create volume for the pipy bootstrap config Secret
	pod.Spec.Volumes = append(pod.Spec.Volumes, injector.GetVolumeSpec(pipyBootstrapConfigName))

	err := injector.ConfigurePodInit(configurator, podOS, pod, fsmContainerPullPolicy)
	if err != nil {
		return err
	}

	if originalHealthProbes.UsesTCP() {
		healthcheckContainer := corev1.Container{
			Name:            "fsm-healthcheck",
			Image:           os.Getenv("FSM_DEFAULT_HEALTHCHECK_CONTAINER_IMAGE"),
			ImagePullPolicy: fsmContainerPullPolicy,
			Resources:       getInjectedHealthcheckResources(configurator),
			Args: []string{
				"--verbosity", log.GetLevel().String(),
			},
			Command: []string{
				"/fsm-healthcheck",
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: constants.HealthcheckPort,
				},
			},
		}
		pod.Spec.Containers = append(pod.Spec.Containers, healthcheckContainer)
	}

	// Add the Pipy sidecar
	sidecar := getPipySidecarContainerSpec(injCtx, pod, configurator, cnPrefix, originalHealthProbes, podOS)
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	return nil
}

func getInjectedHealthcheckResources(cfg configurator.Configurator) corev1.ResourceRequirements {
	cfgResources := cfg.GetInjectedHealthcheckResources()
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
	return resources
}
