package injector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/models"
	sidecarv1 "github.com/flomesh-io/fsm/pkg/sidecar/v1"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/driver"
)

func (wh *mutatingWebhook) createPodPatch(pod *corev1.Pod, req *admissionv1.AdmissionRequest, proxyUUID uuid.UUID) ([]byte, error) {
	// This will append a label to the pod, which points to the unique Sidecar ID used in the
	// xDS certificate for that Sidecar. This label will help xDS match the actual pod to the Sidecar that
	// connects to xDS (with the certificate's CN matching this label).
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[constants.SidecarUniqueIDLabelName] = proxyUUID.String()

	if wh.configurator.GetTrafficInterceptionMode() == constants.TrafficInterceptionModeNodeLevel {
		return json.Marshal(makePodPatches(req, pod))
	}

	namespace := req.Namespace

	podOS := pod.Spec.NodeSelector["kubernetes.io/os"]
	if err := wh.verifyPrerequisites(podOS); err != nil {
		return nil, err
	}

	enableMetrics, err := IsMetricsEnabled(wh.kubeController, namespace)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking if namespace %s is enabled for metrics", namespace)
		return nil, err
	}
	if enableMetrics {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[constants.PrometheusScrapeAnnotation] = strconv.FormatBool(true)
		pod.Annotations[constants.PrometheusPortAnnotation] = strconv.Itoa(constants.SidecarPrometheusInboundListenerPort)
		pod.Annotations[constants.PrometheusPathAnnotation] = constants.PrometheusScrapePath
	}

	// Issue a certificate for the proxy sidecar - used for Sidecar to connect to XDS (not Sidecar-to-Sidecar connections)
	cnPrefix := sidecarv1.NewCertCNPrefix(proxyUUID, models.KindSidecar, identity.New(pod.Spec.ServiceAccountName, namespace))
	log.Debug().Msgf("Patching POD spec: service-account=%s, namespace=%s with certificate CN prefix=%s", pod.Spec.ServiceAccountName, namespace, cnPrefix)
	startTime := time.Now()
	bootstrapCertificate, err := wh.certManager.IssueCertificate(cnPrefix, certificate.Internal)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing bootstrap certificate for Sidecar with CN prefix=%s", cnPrefix)
		return nil, err
	}
	elapsed := time.Since(startTime)

	metricsstore.DefaultMetricsStore.CertIssuedCount.Inc()
	metricsstore.DefaultMetricsStore.CertIssuedTime.WithLabelValues().Observe(elapsed.Seconds())

	background := driver.InjectorContext{
		KubeClient:                   wh.kubeClient,
		MeshName:                     wh.meshName,
		FsmNamespace:                 wh.fsmNamespace,
		FsmContainerPullPolicy:       wh.fsmContainerPullPolicy,
		Configurator:                 wh.configurator,
		CertManager:                  wh.certManager,
		Pod:                          pod,
		PodOS:                        podOS,
		PodNamespace:                 namespace,
		ProxyUUID:                    proxyUUID,
		BootstrapCertificateCNPrefix: cnPrefix,
		BootstrapCertificate:         bootstrapCertificate,
		DryRun:                       req.DryRun != nil && *req.DryRun,
	}
	ctx, cancel := context.WithCancel(&background)
	defer cancel()

	if err = sidecarv1.Patch(ctx); err != nil {
		return nil, err
	}

	return json.Marshal(makePodPatches(req, pod))
}

// verifyPrerequisites verifies if the prerequisites to patch the request are met by returning an error if unmet
func (wh *mutatingWebhook) verifyPrerequisites(podOS string) error {
	// Verify that the required images are configured
	if image := wh.configurator.GetSidecarImage(); image == "" {
		// Linux pods require Sidecar Linux image
		return fmt.Errorf("MeshConfig sidecar.sidecarImage not set")
	}
	if image := wh.configurator.GetInitContainerImage(); image == "" {
		// Linux pods require init container image
		return fmt.Errorf("MeshConfig sidecar.initContainerImage not set")
	}

	return nil
}

// ConfigurePodInit patch the init container to pod.
func ConfigurePodInit(cfg configurator.Configurator, podOS string, pod *corev1.Pod, fsmContainerPullPolicy corev1.PullPolicy) error {
	// Build outbound port exclusion list
	podOutboundPortExclusionList, err := GetPortExclusionListForPod(pod, OutboundPortExclusionListAnnotation)
	if err != nil {
		return err
	}
	globalOutboundPortExclusionList := cfg.GetMeshConfig().Spec.Traffic.OutboundPortExclusionList
	outboundPortExclusionList := MergePortExclusionLists(podOutboundPortExclusionList, globalOutboundPortExclusionList)

	// Build inbound port exclusion list
	podInboundPortExclusionList, err := GetPortExclusionListForPod(pod, InboundPortExclusionListAnnotation)
	if err != nil {
		return err
	}
	globalInboundPortExclusionList := cfg.GetMeshConfig().Spec.Traffic.InboundPortExclusionList
	inboundPortExclusionList := MergePortExclusionLists(podInboundPortExclusionList, globalInboundPortExclusionList)

	// Build the outbound IP range exclusion list
	podOutboundIPRangeExclusionList, err := GetOutboundIPRangeListForPod(pod, OutboundIPRangeExclusionListAnnotation)
	if err != nil {
		return err
	}
	globalOutboundIPRangeExclusionList := cfg.GetMeshConfig().Spec.Traffic.OutboundIPRangeExclusionList
	outboundIPRangeExclusionList := MergeIPRangeLists(podOutboundIPRangeExclusionList, globalOutboundIPRangeExclusionList)

	// Build the outbound IP range inclusion list
	podOutboundIPRangeInclusionList, err := GetOutboundIPRangeListForPod(pod, OutboundIPRangeInclusionListAnnotation)
	if err != nil {
		return err
	}
	globalOutboundIPRangeInclusionList := cfg.GetMeshConfig().Spec.Traffic.OutboundIPRangeInclusionList
	outboundIPRangeInclusionList := MergeIPRangeLists(podOutboundIPRangeInclusionList, globalOutboundIPRangeInclusionList)

	networkInterfaceExclusionList := cfg.GetMeshConfig().Spec.Traffic.NetworkInterfaceExclusionList

	// Add the init container to the pod spec
	initContainer := GetInitContainerSpec(constants.InitContainerName, cfg, outboundIPRangeExclusionList, outboundIPRangeInclusionList, outboundPortExclusionList, inboundPortExclusionList, cfg.IsPrivilegedInitContainer(), fsmContainerPullPolicy, networkInterfaceExclusionList)
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)

	return nil
}

func makePodPatches(req *admissionv1.AdmissionRequest, pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	original := req.Object.Raw
	current, err := json.Marshal(pod)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingKubernetesResource)).
			Msgf("Error marshaling Pod with UID=%s", pod.ObjectMeta.UID)
	}
	admissionResponse := admission.PatchResponseFromRaw(original, current)
	return admissionResponse.Patches
}

// GetProxyUUID return proxy uuid retrieved from sidecar bootstrap config volume.
func GetProxyUUID(pod *corev1.Pod) (string, bool) {
	// kubectl debug does not recreate the object with the same metadata
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == SidecarBootstrapConfigVolume {
			return strings.TrimPrefix(volume.Secret.SecretName, BootstrapSecretPrefix), true
		}
	}
	return "", false
}

func (wh *mutatingWebhook) createVmPatch(vm *machinev1alpha1.VirtualMachine, req *admissionv1.AdmissionRequest, proxyUUID uuid.UUID) ([]byte, error) {
	// This will append a label to the vm, which points to the unique Sidecar ID used in the
	// xDS certificate for that Sidecar. This label will help xDS match the actual vm to the Sidecar that
	// connects to xDS (with the certificate's CN matching this label).
	if vm.Labels == nil {
		vm.Labels = make(map[string]string)
	}
	vm.Labels[constants.SidecarUniqueIDLabelName] = proxyUUID.String()

	if wh.configurator.GetTrafficInterceptionMode() == constants.TrafficInterceptionModeNodeLevel {
		return json.Marshal(makeVmPatches(req, vm))
	}

	return json.Marshal(makeVmPatches(req, vm))
}

func makeVmPatches(req *admissionv1.AdmissionRequest, vm *machinev1alpha1.VirtualMachine) []jsonpatch.JsonPatchOperation {
	original := req.Object.Raw
	current, err := json.Marshal(vm)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingKubernetesResource)).
			Msgf("Error marshaling VM with UID=%s", vm.ObjectMeta.UID)
	}
	admissionResponse := admission.PatchResponseFromRaw(original, current)
	return admissionResponse.Patches
}
