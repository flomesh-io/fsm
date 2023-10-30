package sessionsticky

import (
	"net/http"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
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
	log = logger.New("webhook/sessionsticky")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new SessionStickyPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for SessionStickyPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"gateway.flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"sessionstickypolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"msessionstickypolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.SessionStickyPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vsessionstickypolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.SessionStickyPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for SessionStickyPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.SessionStickyPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.SessionStickyPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
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
	return &gwpav1alpha1.SessionStickyPolicy{}
}

// SetDefaults sets the default values for the SessionStickyPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.SessionStickyPolicy)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", policy.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", policy.Spec)

	targetRef := policy.Spec.TargetRef
	if (targetRef.Group == "" && targetRef.Kind == "Service") ||
		(targetRef.Group == "flomesh.io" && targetRef.Kind == "ServiceImport") {

		if policy.Spec.CookieName == nil {
			policy.Spec.CookieName = pointer.String("_srv_id")
		}

		if policy.Spec.Expires == nil {
			policy.Spec.Expires = pointer.Int32(3600)
		}
	}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.SessionStickyPolicy{}
}

// ValidateCreate validates the creation of the SessionStickyPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update of the SessionStickyPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the deletion of the SessionStickyPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.SessionStickyPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	errorList = append(errorList, validateExpires(policy)...)

	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	return nil
}

func validateTargetRef(ref gwv1alpha2.PolicyTargetReference) field.ErrorList {
	var errs field.ErrorList

	if ref.Group != "" && ref.Group != "flomesh.io" {
		path := field.NewPath("spec").Child("targetRef").Child("group")
		errs = append(errs, field.Invalid(path, ref.Group, "group must be set to flomesh.io or core"))
	}

	if (ref.Group == "" && ref.Kind == "Service") ||
		(ref.Group == "flomesh.io" && ref.Kind == "ServiceImport") {
		// do nothing
	} else {
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Service for group core or ServiceImport for group flomesh.io"))
	}

	return errs
}

func validateExpires(policy *gwpav1alpha1.SessionStickyPolicy) field.ErrorList {
	var errs field.ErrorList

	if *policy.Spec.Expires < 0 {
		path := field.NewPath("spec").Child("expires")
		errs = append(errs, field.Invalid(path, policy.Spec.Expires, "must be greater than or equal to 0"))
	}

	return errs
}
