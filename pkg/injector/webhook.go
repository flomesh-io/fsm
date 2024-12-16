package injector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

const (
	// MutatingWebhookName is the name of the mutating webhook used for sidecar injection
	MutatingWebhookName = "fsm-inject.k8s.io"

	// webhookCreatePod is the HTTP path at which the webhook expects to receive pod creation events
	webhookCreatePod = "/mutate-pod-creation"

	// BootstrapSecretPrefix is the prefix of bootstrap Secret.
	BootstrapSecretPrefix = "sidecar-bootstrap-config-"
)

// NewMutatingWebhook starts a new web server handling requests from the injector MutatingWebhookConfiguration
func NewMutatingWebhook(ctx context.Context, kubeClient kubernetes.Interface, certManager *certificate.Manager, kubeController k8s.Controller, meshName, fsmNamespace, webhookConfigName, fsmVersion string, webhookTimeout int32, enableReconciler bool, cfg configurator.Configurator, fsmContainerPullPolicy corev1.PullPolicy) error {
	wh := mutatingWebhook{
		kubeClient:             kubeClient,
		certManager:            certManager,
		kubeController:         kubeController,
		fsmNamespace:           fsmNamespace,
		meshName:               meshName,
		configurator:           cfg,
		fsmContainerPullPolicy: fsmContainerPullPolicy,

		// Sidecars should never be injected in these namespaces
		nonInjectNamespaces: mapset.NewSet(
			metav1.NamespaceSystem,
			metav1.NamespacePublic,
			fsmNamespace,
		),
	}

	// We know that the events arriving at this handler are CREATE POD only
	// because of the specifics of MutatingWebhookConfiguration template in this repository.

	// Start the MutatingWebhook web server
	srv, err := webhook.NewServer(constants.FSMInjectorName, fsmNamespace, constants.InjectorWebhookPort, certManager, map[string]http.HandlerFunc{
		webhookCreatePod: http.HandlerFunc(wh.podCreationHandler),
	},
		func(cert *certificate.Certificate) error {
			if err := createOrUpdateMutatingWebhook(kubeClient, cert, webhookTimeout, webhookConfigName, meshName, fsmNamespace, fsmVersion, enableReconciler); err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		return err
	}
	go srv.Run(ctx)
	return nil
}

func (wh *mutatingWebhook) getAdmissionReqResp(proxyUUID uuid.UUID, admissionRequestBody []byte) (requestForNamespace string, admissionResp admissionv1.AdmissionReview) {
	var admissionReq admissionv1.AdmissionReview
	if _, _, err := webhook.Deserializer.Decode(admissionRequestBody, nil, &admissionReq); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingAdmissionReqBody)).
			Msg("Error decoding admission request body")
		admissionResp.Response = webhook.AdmissionError(err)
	} else {
		admissionResp.Response = wh.mutate(admissionReq.Request, proxyUUID)
	}
	admissionResp.TypeMeta = admissionReq.TypeMeta
	admissionResp.Kind = admissionReq.Kind

	if admissionReq.Request != nil {
		requestForNamespace = admissionReq.Request.Namespace
	}

	webhook.RecordAdmissionMetrics(admissionReq.Request, admissionResp.Response)

	return
}

// podCreationHandler is a MutatingWebhookConfiguration handler exclusive to POD CREATE events.
func (wh *mutatingWebhook) podCreationHandler(w http.ResponseWriter, req *http.Request) {
	log.Trace().Msgf("Received mutating webhook request: Method=%v, URL=%v", req.Method, req.URL)

	if contentType := req.Header.Get(webhook.HTTPHeaderContentType); contentType != webhook.ContentTypeJSON {
		err := fmt.Errorf("Invalid content type %s; Expected %s", contentType, webhook.ContentTypeJSON)
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidAdmissionReqHeader)).
			Msgf("Responded to admission request with HTTP %v", http.StatusUnsupportedMediaType)
		return
	}

	admissionRequestBody, err := webhook.GetAdmissionRequestBody(w, req)
	if err != nil {
		// Error was already logged and written to the ResponseWriter
		return
	}

	// Create the patches for the spec
	// We use req.Namespace because pod.Namespace is "" at this point
	// This string uniquely identifies the pod. Ideally this would be the pod.UID, but this is not available at this point.
	proxyUUID := uuid.New()
	requestForNamespace, admissionResp := wh.getAdmissionReqResp(proxyUUID, admissionRequestBody)

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling admission response: %s", err), http.StatusInternalServerError)
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingKubernetesResource)).
			Msgf("Error marshalling admission response; Responded to admission request for pod with UUID %s in namespace %s with HTTP %v", proxyUUID, requestForNamespace, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrWritingAdmissionResp)).
			Msgf("Error writing admission response for pod with UUID %s in namespace %s", proxyUUID, requestForNamespace)
	}

	log.Trace().Msgf("Done responding to admission request for pod with UUID %s in namespace %s", proxyUUID, requestForNamespace)
}

func (wh *mutatingWebhook) mutate(req *admissionv1.AdmissionRequest, proxyUUID uuid.UUID) *admissionv1.AdmissionResponse {
	if req == nil {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrNilAdmissionReq)).Msg("nil admission Request")
		return webhook.AdmissionError(errNilAdmissionRequest)
	}

	// Decode the Pod spec from the request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingKubernetesResource)).
			Msgf("Error unmarshaling request to pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return webhook.AdmissionError(err)
	}

	// Start building the response
	resp := &admissionv1.AdmissionResponse{
		Allowed: true,
		UID:     req.UID,
	}

	if pod.Kind == "VirtualMachine" && pod.APIVersion == "machine.flomesh.io/v1alpha1" {
		var vm machinev1alpha1.VirtualMachine
		if err := json.Unmarshal(req.Object.Raw, &vm); err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingKubernetesResource)).
				Msgf("Error unmarshaling request to VM with UUID %s in namespace %s", proxyUUID, req.Namespace)
			return webhook.AdmissionError(err)
		}

		// Check if we must inject the sidecar
		if inject, err := wh.mustVmInject(&vm, req.Namespace); err != nil {
			log.Error().Err(err).Msgf("Error checking if sidecar must be injected for VM with UUID %s in namespace %s", proxyUUID, req.Namespace)
			return webhook.AdmissionError(err)
		} else if !inject {
			log.Trace().Msgf("Skipping sidecar injection for VM with UUID %s in namespace %s", proxyUUID, req.Namespace)
			return resp
		}

		patchBytes, err := wh.createVmPatch(&vm, req, proxyUUID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create patch for VM with UUID %s in namespace %s", proxyUUID, req.Namespace)
			return webhook.AdmissionError(err)
		}

		patchAdmissionResponse(resp, patchBytes)
		log.Trace().Msgf("Done creating patch admission response for VM with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return resp
	}

	// Check if we must inject the sidecar
	if inject, err := wh.mustPodInject(&pod, req.Namespace); err != nil {
		log.Error().Err(err).Msgf("Error checking if sidecar must be injected for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return webhook.AdmissionError(err)
	} else if !inject {
		log.Trace().Msgf("Skipping sidecar injection for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return resp
	}

	patchBytes, err := wh.createPodPatch(&pod, req, proxyUUID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create patch for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
		return webhook.AdmissionError(err)
	}

	patchAdmissionResponse(resp, patchBytes)
	log.Trace().Msgf("Done creating patch admission response for pod with UUID %s in namespace %s", proxyUUID, req.Namespace)
	return resp
}

func (wh *mutatingWebhook) isNamespaceInjectable(namespace string) bool {
	// Never inject pods in the FSM Controller namespace or kube-public or kube-system
	isInjectableNS := !wh.nonInjectNamespaces.Contains(namespace)

	// Ignore namespaces not joined in the mesh.
	return isInjectableNS && wh.kubeController.IsMonitoredNamespace(namespace)
}

// mustPodInject determines whether the sidecar must be injected.
//
// The sidecar injection is performed when the namespace is labeled for monitoring and either of the following is true:
// 1. The pod is explicitly annotated with enabled/yes/true for sidecar injection, or
// 2. The namespace is annotated for sidecar injection and the pod is not explicitly annotated with disabled/no/false
//
// The function returns an error when it is unable to determine whether to perform sidecar injection.
func (wh *mutatingWebhook) mustPodInject(pod *corev1.Pod, namespace string) (bool, error) {
	// Sidecar injection is not permitted for pods on the host network.
	// Since iptables rules are created to intercept and redirect traffic via the proxy sidecar,
	// pods on the host network cannot be injected with the sidecar as the required iptables rules
	// will result in routing failures on the host's network.
	if pod.Spec.HostNetwork {
		log.Debug().Msgf("Pod with UID %s has HostNetwork enabled, cannot inject a sidecar", pod.ObjectMeta.UID)
		return false, nil
	}

	if !wh.isNamespaceInjectable(namespace) {
		log.Warn().Msgf("Mutation request is for pod with UID %s; Injection in Namespace %s is not permitted", pod.ObjectMeta.UID, namespace)
		return false, nil
	}

	// Check if the pod is annotated for injection
	injectAnnotationExists, podInject, err := isAnnotatedForInjection(pod.Annotations, "Pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingPodInjectionEnablement)).
			Msg("Error determining if the pod is enabled for sidecar injection")
		return false, err
	}

	// Check if the namespace is annotated for injection
	ns := wh.kubeController.GetNamespace(namespace)
	if ns == nil {
		log.Error().Err(errNamespaceNotFound).Msgf("Error retrieving namespace %s", namespace)
		return false, errNamespaceNotFound
	}
	nsInjectAnnotationExists, nsInject, err := isAnnotatedForInjection(ns.Annotations, "Namespace", ns.Name)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingNamespaceInjectionEnablement)).
			Msgf("Error determining if namespace %s is enabled for sidecar injection", namespace)
		return false, err
	}

	if injectAnnotationExists && podInject {
		// Pod is explicitly annotated to enable sidecar injection
		return true, nil
	} else if nsInjectAnnotationExists && nsInject {
		// Namespace is annotated to enable sidecar injection
		if !injectAnnotationExists || podInject {
			// If pod annotation doesn't exist or if an annotation exists to enable injection, enable it
			return true, nil
		}
	}

	// Conditions to inject the sidecar are not met
	return false, nil
}

func (wh *mutatingWebhook) mustVmInject(vm *machinev1alpha1.VirtualMachine, namespace string) (bool, error) {
	if !wh.isNamespaceInjectable(namespace) {
		log.Warn().Msgf("Mutation request is for VM with UID %s; Injection in Namespace %s is not permitted", vm.ObjectMeta.UID, namespace)
		return false, nil
	}

	// Check if the VM is annotated for injection
	injectAnnotationExists, vmInject, err := isAnnotatedForInjection(vm.Annotations, "VirtualMachine", fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingPodInjectionEnablement)).
			Msg("Error determining if the VM is enabled for sidecar injection")
		return false, err
	}

	// Check if the namespace is annotated for injection
	ns := wh.kubeController.GetNamespace(namespace)
	if ns == nil {
		log.Error().Err(errNamespaceNotFound).Msgf("Error retrieving namespace %s", namespace)
		return false, errNamespaceNotFound
	}
	nsInjectAnnotationExists, nsInject, err := isAnnotatedForInjection(ns.Annotations, "Namespace", ns.Name)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDeterminingNamespaceInjectionEnablement)).
			Msgf("Error determining if namespace %s is enabled for sidecar injection", namespace)
		return false, err
	}

	if injectAnnotationExists && vmInject {
		// VM is explicitly annotated to enable sidecar injection
		return true, nil
	} else if nsInjectAnnotationExists && nsInject {
		// Namespace is annotated to enable sidecar injection
		if !injectAnnotationExists || vmInject {
			// If vm annotation doesn't exist or if an annotation exists to enable injection, enable it
			return true, nil
		}
	}

	// Conditions to inject the sidecar are not met
	return false, nil
}

func isAnnotatedForInjection(annotations map[string]string, objectKind string, objectName string) (exists bool, enabled bool, err error) {
	inject, ok := annotations[constants.SidecarInjectionAnnotation]
	if !ok {
		return
	}

	log.Trace().Msgf("%s '%s' has sidecar injection annotation: '%s:%s'", objectKind, objectName, constants.SidecarInjectionAnnotation, inject)
	exists = true
	switch strings.ToLower(inject) {
	case "enabled", "yes", "true":
		enabled = true
	case "disabled", "no", "false":
		enabled = false
	default:
		err = fmt.Errorf("invalid annotation value for key %q: %s", constants.SidecarInjectionAnnotation, inject)
	}
	return
}

func patchAdmissionResponse(resp *admissionv1.AdmissionResponse, patchBytes []byte) {
	resp.Patch = patchBytes
	pt := admissionv1.PatchTypeJSONPatch
	resp.PatchType = &pt
}

func createOrUpdateMutatingWebhook(clientSet kubernetes.Interface, cert *certificate.Certificate, webhookTimeout int32, webhookName, meshName, fsmNamespace, fsmVersion string, enableReconciler bool) error {
	webhookPath := webhookCreatePod
	webhookPort := int32(constants.InjectorWebhookPort)
	failurePolicy := admissionregv1.Fail
	matchPolicy := admissionregv1.Exact

	mwhcLabels := map[string]string{
		constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
		constants.FSMAppInstanceLabelKey: meshName,
		constants.FSMAppVersionLabelKey:  fsmVersion,
		constants.AppLabel:               constants.FSMInjectorName,
	}

	if enableReconciler {
		mwhcLabels[constants.ReconcileLabel] = strconv.FormatBool(true)
	}

	mwhc := admissionregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   webhookName,
			Labels: mwhcLabels,
		},
		Webhooks: []admissionregv1.MutatingWebhook{
			{
				Name: MutatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					Service: &admissionregv1.ServiceReference{
						Namespace: fsmNamespace,
						Name:      constants.FSMInjectorName,
						Path:      &webhookPath,
						Port:      &webhookPort,
					},
					CABundle: cert.GetIssuingCA()},
				FailurePolicy: &failurePolicy,
				MatchPolicy:   &matchPolicy,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						constants.FSMKubeResourceMonitorAnnotation: meshName,
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      constants.IgnoreLabel,
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
						{
							Key:      "name",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{fsmNamespace},
						},
						{
							Key:      "control-plane",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
					},
				},
				Rules: []admissionregv1.RuleWithOperations{
					{
						Operations: []admissionregv1.OperationType{admissionregv1.Create},
						Rule: admissionregv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
					{
						Operations: []admissionregv1.OperationType{admissionregv1.Create},
						Rule: admissionregv1.Rule{
							APIGroups:   []string{"machine.flomesh.io"},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"virtualmachines"},
						},
					},
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				TimeoutSeconds:          &webhookTimeout,
				AdmissionReviewVersions: []string{"v1"}}},
	}

	if _, err := clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), &mwhc, metav1.CreateOptions{}); err != nil {
		// Webhook already exists, update the webhook in this scenario
		if apierrors.IsAlreadyExists(err) {
			existing, err := clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), mwhc.Name, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingMutatingWebhook)).
					Msgf("Error getting MutatingWebhookConfiguration %s", webhookName)
				return err
			}

			mwhc.ObjectMeta = existing.ObjectMeta // copy the object meta which includes resource version, required for updates.
			if _, err = clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.Background(), &mwhc, metav1.UpdateOptions{}); err != nil {
				// There might be conflicts when multiple injectors try to update the same resource
				// One of the injectors will successfully update the resource, hence conflicts shoud be ignored and not treated as an error
				if !apierrors.IsConflict(err) {
					log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingMutatingWebhook)).
						Msgf("Error updating MutatingWebhookConfiguration %s", webhookName)
					return err
				}
			}
		} else {
			// Webhook doesn't exist and could not be created, an error is logged and returned
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingMutatingWebhook)).
				Msgf("Error creating MutatingWebhookConfiguration %s", webhookName)
			return err
		}
	}

	log.Info().Msgf("Finished creating MutatingWebhookConfiguration %s", webhookName)
	return nil
}
