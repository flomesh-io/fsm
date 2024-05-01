package healthcheck

import (
	"fmt"
	"net/http"

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
	log = logger.New("webhook/healthcheck")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new HealthCheckPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for HealthCheckPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.FlomeshGatewayAPIGroup},
		[]string{"v1alpha1"},
		[]string{"healthcheckpolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mhealthcheckpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.HealthCheckPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vhealthcheckpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.HealthCheckPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for HealthCheckPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.HealthCheckPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Configurator)),
		constants.HealthCheckPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient)),
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
	return &gwpav1alpha1.HealthCheckPolicy{}
}

// SetDefaults sets the default values for the HealthCheckPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.HealthCheckPolicy)
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

//func setDefaults(config *gwpav1alpha1.HealthCheckConfig, defaultConfig *gwpav1alpha1.HealthCheckConfig) *gwpav1alpha1.HealthCheckConfig {
//	switch {
//	case config == nil && defaultConfig == nil:
//		return nil
//	case config == nil && defaultConfig != nil:
//		return setDefaultValues(defaultConfig.DeepCopy())
//	case config != nil && defaultConfig == nil:
//		return setDefaultValues(config.DeepCopy())
//	case config != nil && defaultConfig != nil:
//		return mergeConfig(config, defaultConfig)
//	}
//
//	return nil
//}
//
//func mergeConfig(config *gwpav1alpha1.HealthCheckConfig, defaultConfig *gwpav1alpha1.HealthCheckConfig) *gwpav1alpha1.HealthCheckConfig {
//	cfgCopy := config.DeepCopy()
//
//	if cfgCopy.Path == nil && defaultConfig.Path != nil {
//		cfgCopy.Path = defaultConfig.Path
//	}
//
//	if len(cfgCopy.Matches) == 0 && len(defaultConfig.Matches) > 0 {
//		cfgCopy.Matches = make([]gwpav1alpha1.HealthCheckMatch, 0)
//		cfgCopy.Matches = append(cfgCopy.Matches, defaultConfig.Matches...)
//	}
//
//	if cfgCopy.FailTimeout == nil && defaultConfig.FailTimeout != nil {
//		cfgCopy.FailTimeout = defaultConfig.FailTimeout
//	}
//
//	return cfgCopy
//}
//
//func setDefaultValues(config *gwpav1alpha1.HealthCheckConfig) *gwpav1alpha1.HealthCheckConfig {
//	cfg := config.DeepCopy()
//
//	if cfg.Path != nil && len(cfg.Matches) == 0 {
//		cfg.Matches = []gwpav1alpha1.HealthCheckMatch{
//			{
//				StatusCodes: []int32{200},
//			},
//		}
//	}
//
//	return cfg
//}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.HealthCheckPolicy{}
}

// ValidateCreate validates the creation of the HealthCheckPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update of the HealthCheckPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the deletion of the HealthCheckPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.HealthCheckPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, validateSpec(policy)...)
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

func validateSpec(policy *gwpav1alpha1.HealthCheckPolicy) field.ErrorList {
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
		errs = validateConfig(field.NewPath("spec").Child("config"), policy.Spec.DefaultConfig)
	}

	if len(policy.Spec.Ports) > 0 {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			if port.Config != nil {
				errs = append(errs, validateConfig(path.Index(i).Child("config"), port.Config)...)
			}
		}
	}

	return errs
}

func validateConfig(path *field.Path, config *gwpav1alpha1.HealthCheckConfig) field.ErrorList {
	var errs field.ErrorList

	if config.Path != nil && len(config.Matches) == 0 {
		errs = append(errs, field.Invalid(path.Child("matches"), config.Matches, "must be set if path is set"))
	}

	if len(config.Matches) != 0 && config.Path == nil {
		errs = append(errs, field.Invalid(path.Child("path"), config.Path, "must be set if matches is set"))
	}

	if len(config.Matches) > 0 {
		for i, match := range config.Matches {
			if len(match.StatusCodes) == 0 && match.Body == nil && len(match.Headers) == 0 {
				errs = append(errs, field.Invalid(path.Child("matches").Index(i), match, "must have at least one of statusCodes, body or headers"))
			}
		}
	}

	return errs
}
