package ratelimit

import (
	"fmt"
	"net/http"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

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
	log = logger.New("webhook/ratelimit")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new RateLimitPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for RateLimitPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"gateway.flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"ratelimitpolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mratelimitpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.RateLimitPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vratelimitpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.RateLimitPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for RateLimitPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.RateLimitPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.RateLimitPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
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
	return &gwpav1alpha1.RateLimitPolicy{}
}

// SetDefaults sets the default values for the RateLimitPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", policy.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", policy.Spec)

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup {
		if policy.Spec.TargetRef.Kind == constants.HTTPRouteKind ||
			policy.Spec.TargetRef.Kind == constants.GRPCRouteKind {
			if len(policy.Spec.Match.Hostnames) > 0 || policy.Spec.Match.Route != nil {
				if policy.Spec.RateLimit.L7RateLimit != nil {
					setDefaults(policy)
				}
			}
		}
	}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

func setDefaults(policy *gwpav1alpha1.RateLimitPolicy) {
	if policy.Spec.RateLimit.L7RateLimit.Mode == nil {
		policy.Spec.RateLimit.L7RateLimit.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
	}

	if policy.Spec.RateLimit.L7RateLimit.Backlog == nil {
		policy.Spec.RateLimit.L7RateLimit.Backlog = pointer.Int(10)
	}

	if policy.Spec.RateLimit.L7RateLimit.Burst == nil {
		policy.Spec.RateLimit.L7RateLimit.Burst = &policy.Spec.RateLimit.L7RateLimit.Requests
	}
}

func rateLimitPolicyModePointer(mode gwpav1alpha1.RateLimitPolicyMode) *gwpav1alpha1.RateLimitPolicyMode {
	return &mode
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.RateLimitPolicy{}
}

// ValidateCreate validates the creation of the RateLimitPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update of the RateLimitPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the deletion of the RateLimitPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	errorList = append(errorList, validateMatch(policy)...)
	errorList = append(errorList, validateConfig(policy)...)

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
	case constants.GatewayKind, constants.HTTPRouteKind, constants.GRPCRouteKind:
		// do nothing
	default:
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Gateway, HTTPRoute or GRPCRoute"))
	}

	return errs
}

func validateMatch(policy *gwpav1alpha1.RateLimitPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup && policy.Spec.TargetRef.Kind == constants.GatewayKind {
		if policy.Spec.Match.Port == nil && len(policy.Spec.Match.Hostnames) == 0 {
			path := field.NewPath("spec").Child("match")
			errs = append(errs, field.Invalid(path, policy.Spec.Match, "either port or hostnames must be set for Gateway target"))
		}

		if policy.Spec.Match.Port != nil && len(policy.Spec.Match.Hostnames) > 0 {
			path := field.NewPath("spec").Child("match")
			errs = append(errs, field.Invalid(path, policy.Spec.Match, "only one of port or hostnames can be set for Gateway target"))
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.HTTPRouteKind || policy.Spec.TargetRef.Kind == constants.GRPCRouteKind) {
		if len(policy.Spec.Match.Hostnames) == 0 && policy.Spec.Match.Route == nil {
			path := field.NewPath("spec").Child("match")
			errs = append(errs, field.Invalid(path, policy.Spec.Match, "either hostnames or route must be set for HTTPRoute or GRPCRoute target"))
		}

		if len(policy.Spec.Match.Hostnames) > 0 && policy.Spec.Match.Route != nil {
			path := field.NewPath("spec").Child("match")
			errs = append(errs, field.Invalid(path, policy.Spec.Match, "only one of hostnames or route can be set for HTTPRoute or GRPCRoute target"))
		}
	}

	if len(policy.Spec.Match.Hostnames) != 0 {
		errs = append(errs, validateHostnames(policy.Spec.Match.Hostnames)...)
	}

	return errs
}

func validateHostnames(hostnames []gwv1beta1.Hostname) field.ErrorList {
	var errs field.ErrorList

	for i, hostname := range hostnames {
		h := string(hostname)
		if err := webhook.IsValidHostname(h); err != nil {
			path := field.NewPath("spec").
				Child("match").
				Child("hostnames").Index(i)

			errs = append(errs, field.Invalid(path, h, fmt.Sprintf("%s", err)))
		}
	}

	return errs
}

func validateConfig(policy *gwpav1alpha1.RateLimitPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup && policy.Spec.TargetRef.Kind == constants.GatewayKind {
		if policy.Spec.Match.Port != nil {
			if policy.Spec.RateLimit.L4RateLimit == nil {
				path := field.NewPath("spec").Child("rateLimit").Child("bps")
				errs = append(errs, field.Required(path, "bps must be set as spec.match.port is set"))
			}
		}

		if len(policy.Spec.Match.Hostnames) > 0 {
			if policy.Spec.RateLimit.L7RateLimit == nil {
				path := field.NewPath("spec").Child("rateLimit").Child("config")
				errs = append(errs, field.Required(path, "config must be set as spec.match.hostnames is set"))
			}
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.HTTPRouteKind || policy.Spec.TargetRef.Kind == constants.GRPCRouteKind) {
		if len(policy.Spec.Match.Hostnames) > 0 || policy.Spec.Match.Route != nil {
			if policy.Spec.RateLimit.L7RateLimit == nil {
				path := field.NewPath("spec").Child("rateLimit").Child("config")
				errs = append(errs, field.Required(path, "config must be set as spec.match.hostnames/route is set"))
			}
		}
	}

	return errs
}
