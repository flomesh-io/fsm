package retry

import (
	"fmt"
	"net/http"
	"regexp"

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
	log = logger.New("webhook/retry")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new RetryPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for RetryPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.FlomeshGatewayAPIGroup},
		[]string{"v1alpha1"},
		[]string{"retrypolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mretrypolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.RetryPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vretrypolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.RetryPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for RetryPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.RetryPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.RetryPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
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
	return &gwpav1alpha1.RetryPolicy{}
}

// SetDefaults sets the default values for the RetryPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.RetryPolicy)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", policy.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", policy.Spec)

	//targetRef := policy.Spec.TargetRef
	//if (targetRef.Group == constants.KubernetesCoreGroup && targetRef.Kind == constants.KubernetesServiceKind) ||
	//	(targetRef.Group == constants.FlomeshAPIGroup && targetRef.Kind == constants.FlomeshAPIServiceImportKind) {
	//	if len(policy.Spec.Ports) > 0 {
	//		for i, p := range policy.Spec.Ports {
	//			if p.Config != nil {
	//				policy.Spec.Ports[i].Config = setDefaults(p.Config, policy.Spec.DefaultConfig)
	//			}
	//		}
	//	}
	//
	//	if policy.Spec.DefaultConfig != nil {
	//		policy.Spec.DefaultConfig = setDefaultValues(policy.Spec.DefaultConfig)
	//	}
	//}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.RetryPolicy{}
}

// ValidateCreate validates the creation of the RetryPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update of the RetryPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the deletion of the RetryPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.RetryPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, validateConfig(policy)...)
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

func validateConfig(policy *gwpav1alpha1.RetryPolicy) field.ErrorList {
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

	if policy.Spec.DefaultConfig != nil {
		path := field.NewPath("spec").Child("config")
		errs = append(errs, validateRetryConfig(path, policy.Spec.DefaultConfig)...)
	}

	if len(policy.Spec.Ports) > 0 {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			errs = append(errs, validateRetryConfig(path.Index(i).Child("config"), port.Config)...)
		}
	}

	return errs
}

// retryOnMaxLength is the maximum length each of the retryOn item
const retryOnMaxLength int = 3

const retryOnStatusCodeFmt = "[1-9]?[0-9][0-9]|[1-9][x][x]"
const retryOnStatusCodeErrorMsg = "length of status code must be 3, with leading digit and last 2 digits being 0-9 or x, i.e. 5xx, 500, 502, 503, 504"

var retryOnStatusCodeFmtRegexp = regexp.MustCompile("^" + retryOnStatusCodeFmt + "$")

func validateRetryConfig(path *field.Path, config *gwpav1alpha1.RetryConfig) field.ErrorList {
	var errs field.ErrorList

	if len(config.RetryOn) > 0 {
		for _, code := range config.RetryOn {
			if len(code) > retryOnMaxLength {
				errs = append(errs, field.TooLongMaxLength(path.Child("retryOn"), code, retryOnMaxLength))
			}
			if !retryOnStatusCodeFmtRegexp.MatchString(code) {
				errs = append(errs, field.Invalid(path.Child("retryOn"), code, retryOnStatusCodeErrorMsg))
			}
		}
	} else {
		errs = append(errs, field.Required(path.Child("retryOn"), "retryOn cannot be empty"))
	}

	return errs
}
