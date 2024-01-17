package faultinjection

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
	log = logger.New("webhook/faultinjection")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new FaultInjectionPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for FaultInjectionPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.FlomeshGatewayAPIGroup},
		[]string{"v1alpha1"},
		[]string{"faultinjectionpolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mfaultinjectionpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.FaultInjectionPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vfaultinjectionpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.FaultInjectionPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for FaultInjectionPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.FaultInjectionPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Config)),
		constants.FaultInjectionPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient)),
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
	return &gwpav1alpha1.FaultInjectionPolicy{}
}

// SetDefaults sets the default values for the FaultInjectionPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.FaultInjectionPolicy)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", policy.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", policy.Spec)

	//if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup {
	//	if policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind ||
	//		policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind {
	//		if len(policy.Spec.Hostnames) > 0 ||
	//			len(policy.Spec.HTTPFaultInjections) > 0 ||
	//			len(policy.Spec.GRPCFaultInjections) > 0 {
	//			setDefaults(policy)
	//		}
	//	}
	//}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.FaultInjectionPolicy{}
}

// ValidateCreate validates the creation of the FaultInjectionPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update of the FaultInjectionPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the deletion of the FaultInjectionPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.FaultInjectionPolicy)
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

	if ref.Group != constants.GatewayAPIGroup {
		path := field.NewPath("spec").Child("targetRef").Child("group")
		errs = append(errs, field.Invalid(path, ref.Group, "group must be set to gateway.networking.k8s.io"))
	}

	switch ref.Kind {
	case constants.GatewayAPIHTTPRouteKind, constants.GatewayAPIGRPCRouteKind:
		// do nothing
	default:
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to HTTPRoute or GRPCRoute"))
	}

	return errs
}

func validateSpec(policy *gwpav1alpha1.FaultInjectionPolicy) field.ErrorList {
	errs := validateL7FaultInjection(policy)
	errs = append(errs, validateConfig(policy)...)

	return errs
}

func validateConfig(policy *gwpav1alpha1.FaultInjectionPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind ||
			policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind) {
		if policy.Spec.DefaultConfig != nil {
			path := field.NewPath("spec").Child("config")
			errs = append(errs, validateConfigFormat(path, policy.Spec.DefaultConfig)...)
		}

		if len(policy.Spec.Hostnames) > 0 {
			for i, h := range policy.Spec.Hostnames {
				if h.Config != nil {
					path := field.NewPath("spec").Child("hostnames").Index(i).Child("config")
					errs = append(errs, validateConfigFormat(path, h.Config)...)
				}
			}
		}

		if len(policy.Spec.HTTPFaultInjections) > 0 {
			for i, h := range policy.Spec.HTTPFaultInjections {
				if h.Config != nil {
					path := field.NewPath("spec").Child("http").Index(i).Child("config")
					errs = append(errs, validateConfigFormat(path, h.Config)...)
				}
			}
		}

		if len(policy.Spec.GRPCFaultInjections) > 0 {
			for i, g := range policy.Spec.GRPCFaultInjections {
				if g.Config != nil {
					path := field.NewPath("spec").Child("grpc").Index(i).Child("config")
					errs = append(errs, validateConfigFormat(path, g.Config)...)
				}
			}
		}
	}

	return errs
}

func validateConfigFormat(path *field.Path, config *gwpav1alpha1.FaultInjectionConfig) field.ErrorList {
	var errs field.ErrorList

	if config.Delay == nil && config.Abort == nil {
		errs = append(errs, field.Required(path, "either delay or abort must be set"))
	}

	if config.Delay != nil && config.Abort != nil {
		errs = append(errs, field.Invalid(path, config, "only one of delay or abort can be set at the same time"))
	}

	if config.Delay != nil {
		if config.Delay.Fixed == nil && config.Delay.Range == nil {
			errs = append(errs, field.Required(path.Child("delay"), "either fixed or range must be set"))
		}

		if config.Delay.Fixed != nil && config.Delay.Range != nil {
			errs = append(errs, field.Invalid(path.Child("delay"), config.Delay, "only one of fixed or range can be set for delay at the same time"))
		}

		if config.Delay.Range != nil {
			if config.Delay.Range.Min >= config.Delay.Range.Max {
				errs = append(errs, field.Invalid(path.Child("delay").Child("range").Child("min"), config.Delay.Range.Min, "min must be less than max"))
			}
		}
	}

	return errs
}

func validateL7FaultInjection(policy *gwpav1alpha1.FaultInjectionPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind || policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind) {
		if len(policy.Spec.Hostnames) == 0 && len(policy.Spec.HTTPFaultInjections) == 0 && len(policy.Spec.GRPCFaultInjections) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, policy.Spec, "any one of hostnames, http or grpc must be set for HTTPRoute/GRPCRoute target"))
		}

		if len(policy.Spec.Hostnames) > 0 {
			errs = append(errs, validateHostnames(policy)...)
		}

		if policy.Spec.DefaultConfig == nil {
			if len(policy.Spec.Hostnames) > 0 {
				for i, h := range policy.Spec.Hostnames {
					if h.Config == nil {
						path := field.NewPath("spec").Child("hostnames").Index(i).Child("config")
						errs = append(errs, field.Required(path, fmt.Sprintf("config must be set for hostname %q, as there's no default config", h.Hostname)))
					}
				}
			}

			if len(policy.Spec.HTTPFaultInjections) > 0 {
				for i, h := range policy.Spec.HTTPFaultInjections {
					if h.Config == nil {
						path := field.NewPath("spec").Child("http").Index(i).Child("config")
						errs = append(errs, field.Required(path, "config must be set, as there's no default config"))
					}
				}
			}

			if len(policy.Spec.GRPCFaultInjections) > 0 {
				for i, g := range policy.Spec.GRPCFaultInjections {
					if g.Config == nil {
						path := field.NewPath("spec").Child("grpc").Index(i).Child("config")
						errs = append(errs, field.Required(path, "config must be set, as there's no default config"))
					}
				}
			}
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind {
		if len(policy.Spec.HTTPFaultInjections) == 0 && len(policy.Spec.Hostnames) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, nil, "either hostnames or http must be set for HTTPRoute target"))
		}

		if len(policy.Spec.GRPCFaultInjections) > 0 {
			path := field.NewPath("spec").Child("grpc")
			errs = append(errs, field.Invalid(path, policy.Spec.GRPCFaultInjections, "must be empty for HTTPRoute target"))
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind {
		if len(policy.Spec.GRPCFaultInjections) == 0 && len(policy.Spec.Hostnames) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, nil, "either hostnames or grpc must be set for GRPCRoute target"))
		}

		if len(policy.Spec.HTTPFaultInjections) > 0 {
			path := field.NewPath("spec").Child("http")
			errs = append(errs, field.Invalid(path, policy.Spec.HTTPFaultInjections, "must be empty for GRPCRoute target"))
		}
	}

	return errs
}

func validateHostnames(policy *gwpav1alpha1.FaultInjectionPolicy) field.ErrorList {
	var errs field.ErrorList

	for i, r := range policy.Spec.Hostnames {
		h := string(r.Hostname)
		if err := webhook.IsValidHostname(h); err != nil {
			path := field.NewPath("spec").
				Child("hostnames").
				Index(i).
				Child("hostname")

			errs = append(errs, field.Invalid(path, h, fmt.Sprintf("%s", err)))
		}
	}

	return errs
}
