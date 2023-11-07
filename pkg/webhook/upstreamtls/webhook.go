package upstreamtls

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/pointer"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

var (
	log = logger.New("webhook/upstreamtls")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new UpstreamTLSPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for UpstreamTLSPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"gateway.flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"upstreamtlspolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mupstreamtlspolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.UpstreamTLSPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vupstreamtlspolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.UpstreamTLSPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for UpstreamTLSPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.UpstreamTLSPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.UpstreamTLSPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
	}
}

type defaulter struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
}

func newDefaulter(kubeClient kubernetes.Interface, cfg configurator.Configurator) *defaulter {
	return &defaulter{
		kubeClient: kubeClient,
		cfg:        cfg,
	}
}

// RuntimeObject returns the runtime object for the webhook
func (w *defaulter) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.UpstreamTLSPolicy{}
}

// SetDefaults sets the default values for the UpstreamTLSPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.UpstreamTLSPolicy)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", policy.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", policy.Spec)

	targetRef := policy.Spec.TargetRef
	if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
		(targetRef.Group == constants.FlomeshAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
		if len(policy.Spec.Ports) > 0 {
			for i, p := range policy.Spec.Ports {
				if p.Config != nil {
					policy.Spec.Ports[i].Config = setDefaults(p.Config, policy.Spec.DefaultConfig)
				}
			}
		}

		if policy.Spec.DefaultConfig != nil {
			policy.Spec.DefaultConfig = setDefaultValues(policy.Spec.DefaultConfig)
		}
	}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

func setDefaults(config *gwpav1alpha1.UpstreamTLSConfig, defaultConfig *gwpav1alpha1.UpstreamTLSConfig) *gwpav1alpha1.UpstreamTLSConfig {
	switch {
	case config == nil && defaultConfig == nil:
		return nil
	case config == nil && defaultConfig != nil:
		return setDefaultValues(defaultConfig.DeepCopy())
	case config != nil && defaultConfig == nil:
		return setDefaultValues(config.DeepCopy())
	case config != nil && defaultConfig != nil:
		return mergeConfig(config, defaultConfig)
	}

	return nil
}

func mergeConfig(config *gwpav1alpha1.UpstreamTLSConfig, defaultConfig *gwpav1alpha1.UpstreamTLSConfig) *gwpav1alpha1.UpstreamTLSConfig {
	cfgCopy := config.DeepCopy()

	if cfgCopy.MTLS == nil {
		if defaultConfig.MTLS != nil {
			// use port config
			cfgCopy.MTLS = defaultConfig.MTLS
		} else {
			// all nil, set to false
			cfgCopy.MTLS = pointer.Bool(false)
		}
	}

	return cfgCopy
}

func setDefaultValues(config *gwpav1alpha1.UpstreamTLSConfig) *gwpav1alpha1.UpstreamTLSConfig {
	cfg := config.DeepCopy()

	if cfg.MTLS == nil {
		cfg.MTLS = pointer.Bool(false)
	}

	return cfg
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.UpstreamTLSPolicy{}
}

// ValidateCreate validates the creation of the UpstreamTLSPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of the UpstreamTLSPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the UpstreamTLSPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.UpstreamTLSPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	errorList = append(errorList, validateConfig(policy)...)
	errorList = append(errorList, w.validateConfigDetails(policy)...)

	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	return nil
}

func validateTargetRef(ref gwv1alpha2.PolicyTargetReference) field.ErrorList {
	var errs field.ErrorList

	if ref.Group != constants.KubernetesCoreGroup && ref.Group != constants.FlomeshAPIGroup {
		path := field.NewPath("spec").Child("targetRef").Child("group")
		errs = append(errs, field.Invalid(path, ref.Group, "group must be set to flomesh.io or core"))
	}

	if (ref.Group == constants.KubernetesCoreGroup && ref.Kind == constants.KubernetesServiceKind) ||
		(ref.Group == constants.FlomeshAPIGroup && ref.Kind == constants.FlomeshAPIServiceImportKind) {
		// do nothing
	} else {
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Service for group core or ServiceImport for group flomesh.io"))
	}

	// TODO: validate ports exist in the referenced service
	//if ref.Group == constants.KubernetesCoreGroup && ref.Kind == constants.KubernetesServiceKind {
	//
	//}
	//
	//if ref.Group == constants.FlomeshAPIGroup && ref.Kind == constants.FlomeshAPIServiceImportKind {
	//
	//}

	return errs
}

func validateConfig(policy *gwpav1alpha1.UpstreamTLSPolicy) field.ErrorList {
	var errs field.ErrorList

	if len(policy.Spec.Ports) == 0 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "cannot be empty"))
	}

	if len(policy.Spec.Ports) > 16 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "max port items cannot be greater than 16"))
	}

	if policy.Spec.DefaultConfig == nil {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			if port.Config == nil {
				errs = append(errs, field.Required(path.Index(i).Child("config"), fmt.Sprintf("config must be set for port %d, as there's no default config", port.Port)))
			}
		}
	}

	return errs
}

func (w *validator) validateConfigDetails(policy *gwpav1alpha1.UpstreamTLSPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.DefaultConfig != nil {
		path := field.NewPath("spec").Child("defaultConfig").Child("certificateRef")
		errs = append(errs, w.validateCertificateRef(path, policy.Spec.DefaultConfig.CertificateRef, policy.Spec.DefaultConfig.MTLS, policy.Namespace)...)
	}

	for i, port := range policy.Spec.Ports {
		if port.Config == nil {
			continue
		}

		path := field.NewPath("spec").Child("ports").Index(i).Child("config").Child("certificateRef")
		errs = append(errs, w.validateCertificateRef(path, port.Config.CertificateRef, port.Config.MTLS, policy.Namespace)...)
	}

	return errs
}

func (w *validator) validateCertificateRef(path *field.Path, certificateRef gwv1beta1.SecretObjectReference, mTLS *bool, ownerNs string) field.ErrorList {
	var errs field.ErrorList

	if certificateRef.Group == nil {
		errs = append(errs, field.Required(path.Child("group"), "group must be set"))
	}

	if certificateRef.Kind == nil {
		errs = append(errs, field.Required(path.Child("kind"), "kind must be set"))
	}

	if certificateRef.Group != nil && string(*certificateRef.Group) != constants.KubernetesCoreGroup {
		errs = append(errs, field.Invalid(path.Child("group"), certificateRef.Group, "group must be set to core"))
	}

	if certificateRef.Kind != nil && string(*certificateRef.Kind) != constants.KubernetesSecretKind {
		errs = append(errs, field.Invalid(path.Child("kind"), certificateRef.Kind, "kind must be set to Secret"))
	}

	if certificateRef.Group != nil &&
		string(*certificateRef.Group) == constants.KubernetesCoreGroup &&
		certificateRef.Kind != nil &&
		string(*certificateRef.Kind) == constants.KubernetesSecretKind {
		errs = append(errs, w.validateSecret(path, certificateRef, mTLS, ownerNs)...)
	}

	return errs
}

func (w *validator) validateSecret(path *field.Path, certificateRef gwv1beta1.SecretObjectReference, mTLS *bool, ownerNs string) field.ErrorList {
	var errs field.ErrorList

	ns := getSecretNamespace(certificateRef, ownerNs)
	name := string(certificateRef.Name)

	secret, err := w.kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		errs = append(errs, field.NotFound(path, fmt.Sprintf("Failed to get Secret %s/%s: %s", ns, name, err)))
	}

	v, ok := secret.Data[corev1.TLSCertKey]
	if ok {
		if string(v) == "" {
			errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s is empty", ns, name, corev1.TLSCertKey)))
		}
	} else {
		errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s doesn't have required data by key %s", ns, name, corev1.TLSCertKey)))
	}

	v, ok = secret.Data[corev1.TLSPrivateKeyKey]
	if ok {
		if string(v) == "" {
			errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s is empty", ns, name, corev1.TLSPrivateKeyKey)))
		}
	} else {
		errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s doesn't have required data by key %s", ns, name, corev1.TLSPrivateKeyKey)))
	}

	if mTLS != nil && *mTLS {
		v, ok = secret.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			if string(v) == "" {
				errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s cannot be empty if mTLS is enabled.", ns, name, corev1.ServiceAccountRootCAKey)))
			}
		} else {
			errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s must have required data by key %s if mTLS is enabled.", ns, name, corev1.ServiceAccountRootCAKey)))
		}
	}

	return errs
}

func getSecretNamespace(certificateRef gwv1beta1.SecretObjectReference, ownerNs string) string {
	if certificateRef.Namespace == nil {
		return ownerNs
	}

	return string(*certificateRef.Namespace)
}
