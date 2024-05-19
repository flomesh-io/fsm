package ratelimit

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

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
	log = logger.New("webhook/ratelimit")
)

type register struct {
	*webhook.RegisterConfig
	gatewayAPIClient gatewayApiClientset.Interface
}

// NewRegister creates a new RateLimitPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig:   cfg,
		gatewayAPIClient: gatewayApiClientset.NewForConfigOrDie(cfg.KubeConfig),
	}
}

// GetWebhooks returns the webhooks to be registered for RateLimitPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.FlomeshGatewayAPIGroup},
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
		constants.RateLimitPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Configurator)),
		constants.RateLimitPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient, r.gatewayAPIClient)),
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

	//if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup {
	//	if policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind ||
	//		policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind {
	//		if len(policy.Spec.Hostnames) > 0 || len(policy.Spec.HTTPRateLimits) > 0 || len(policy.Spec.GRPCRateLimits) > 0 {
	//			setDefaults(policy)
	//		}
	//	}
	//}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

//func setDefaults(policy *gwpav1alpha1.RateLimitPolicy) {
//	if len(policy.Spec.Hostnames) > 0 {
//		for i, hostname := range policy.Spec.Hostnames {
//			if hostname.Config != nil {
//				policy.Spec.Hostnames[i].Config = l7RateLimitDefaults(hostname.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if len(policy.Spec.HTTPRateLimits) > 0 {
//		for i, hr := range policy.Spec.HTTPRateLimits {
//			if hr.Config != nil {
//				policy.Spec.HTTPRateLimits[i].Config = l7RateLimitDefaults(hr.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if len(policy.Spec.GRPCRateLimits) > 0 {
//		for i, gr := range policy.Spec.GRPCRateLimits {
//			if gr.Config != nil {
//				policy.Spec.GRPCRateLimits[i].Config = l7RateLimitDefaults(gr.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if policy.Spec.DefaultConfig != nil {
//		policy.Spec.DefaultConfig = setDefaultValues(policy.Spec.DefaultConfig)
//	}
//}

//func l7RateLimitDefaults(rateLimit *gwpav1alpha1.L7RateLimit, defaultRateLimit *gwpav1alpha1.L7RateLimit) *gwpav1alpha1.L7RateLimit {
//	switch {
//	case rateLimit == nil && defaultRateLimit == nil:
//		return nil
//	case rateLimit == nil && defaultRateLimit != nil:
//		return setDefaultValues(defaultRateLimit.DeepCopy())
//	case rateLimit != nil && defaultRateLimit == nil:
//		return setDefaultValues(rateLimit.DeepCopy())
//	case rateLimit != nil && defaultRateLimit != nil:
//		return mergeConfig(rateLimit, defaultRateLimit)
//	}
//
//	return nil
//}
//
//func mergeConfig(config *gwpav1alpha1.L7RateLimit, defaultConfig *gwpav1alpha1.L7RateLimit) *gwpav1alpha1.L7RateLimit {
//	cfgCopy := config.DeepCopy()
//
//	if cfgCopy.Mode == nil {
//		if defaultConfig.Mode != nil {
//			cfgCopy.Mode = defaultConfig.Mode
//		} else {
//			cfgCopy.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
//		}
//	}
//
//	if cfgCopy.Backlog == nil {
//		if defaultConfig.Backlog != nil {
//			cfgCopy.Backlog = defaultConfig.Backlog
//		} else {
//			cfgCopy.Backlog = pointer.Int32(10)
//		}
//	}
//
//	if cfgCopy.Burst == nil {
//		if defaultConfig.Burst != nil {
//			cfgCopy.Burst = defaultConfig.Burst
//		} else {
//			cfgCopy.Burst = &cfgCopy.Requests
//		}
//	}
//
//	if cfgCopy.ResponseStatusCode == nil {
//		if defaultConfig.ResponseStatusCode != nil {
//			cfgCopy.ResponseStatusCode = defaultConfig.ResponseStatusCode
//		} else {
//			cfgCopy.ResponseStatusCode = pointer.Int32(429)
//		}
//	}
//
//	if len(config.ResponseHeadersToAdd) == 0 && len(defaultConfig.ResponseHeadersToAdd) > 0 {
//		cfgCopy.ResponseHeadersToAdd = make([]gwv1.HTTPHeader, 0)
//		cfgCopy.ResponseHeadersToAdd = append(cfgCopy.ResponseHeadersToAdd, defaultConfig.ResponseHeadersToAdd...)
//	}
//
//	return cfgCopy
//}
//
//func setDefaultValues(rateLimit *gwpav1alpha1.L7RateLimit) *gwpav1alpha1.L7RateLimit {
//	result := rateLimit.DeepCopy()
//
//	if result.Mode == nil {
//		result.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
//	}
//
//	if result.Backlog == nil {
//		result.Backlog = pointer.Int32(10)
//	}
//
//	if result.Burst == nil {
//		result.Burst = &result.Requests
//	}
//
//	if result.ResponseStatusCode == nil {
//		result.ResponseStatusCode = pointer.Int32(429)
//	}
//
//	return result
//}
//
//func rateLimitPolicyModePointer(mode gwpav1alpha1.RateLimitPolicyMode) *gwpav1alpha1.RateLimitPolicyMode {
//	return &mode
//}

type validator struct {
	kubeClient       kubernetes.Interface
	gatewayAPIClient gatewayApiClientset.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.RateLimitPolicy{}
}

// ValidateCreate validates the creation of the RateLimitPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of the RateLimitPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the RateLimitPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface, gatewayAPIClient gatewayApiClientset.Interface) *validator {
	return &validator{
		kubeClient:       kubeClient,
		gatewayAPIClient: gatewayAPIClient,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, w.validateConfig(policy)...)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	return nil
}

func validateTargetRef(ref gwv1alpha2.NamespacedPolicyTargetReference) field.ErrorList {
	var errs field.ErrorList

	if ref.Group != constants.GatewayAPIGroup {
		path := field.NewPath("spec").Child("targetRef").Child("group")
		errs = append(errs, field.Invalid(path, ref.Group, "group must be set to gateway.networking.k8s.io"))
	}

	switch ref.Kind {
	case constants.GatewayAPIGatewayKind, constants.GatewayAPIHTTPRouteKind, constants.GatewayAPIGRPCRouteKind:
		// do nothing
	default:
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Gateway, HTTPRoute or GRPCRoute"))
	}

	return errs
}

func (w *validator) validateConfig(policy *gwpav1alpha1.RateLimitPolicy) field.ErrorList {
	errs := w.validateL4RateLimits(policy)
	errs = append(errs, validateL7RateLimits(policy)...)

	return errs
}

func (w *validator) validateL4RateLimits(policy *gwpav1alpha1.RateLimitPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIGatewayKind {
		if len(policy.Spec.Ports) == 0 {
			path := field.NewPath("spec").Child("ports")
			errs = append(errs, field.Invalid(path, policy.Spec.Ports, "cannot be empty for Gateway target"))
		}

		if len(policy.Spec.Hostnames) > 0 {
			path := field.NewPath("spec").Child("hostnames")
			errs = append(errs, field.Invalid(path, policy.Spec.Hostnames, "must be empty for Gateway target"))
		}

		if len(policy.Spec.HTTPRateLimits) > 0 {
			path := field.NewPath("spec").Child("http")
			errs = append(errs, field.Invalid(path, policy.Spec.HTTPRateLimits, "must be empty for Gateway target"))
		}

		if len(policy.Spec.GRPCRateLimits) > 0 {
			path := field.NewPath("spec").Child("grpc")
			errs = append(errs, field.Invalid(path, policy.Spec.GRPCRateLimits, "must be empty for Gateway target"))
		}

		if policy.Spec.DefaultConfig != nil {
			path := field.NewPath("spec").Child("rateLimit")
			errs = append(errs, field.Invalid(path, policy.Spec.DefaultConfig, "must not be set for Gateway target"))
		}

		if policy.Spec.DefaultBPS == nil {
			path := field.NewPath("spec").Child("ports")
			for i, port := range policy.Spec.Ports {
				if port.BPS == nil {
					errs = append(errs, field.Required(path.Index(i).Child("bps"), fmt.Sprintf("bps must be set for port %d, as there's no default BPS", port.Port)))
				}
			}
		}

		if len(policy.Spec.Ports) > 0 {
			gwName := string(policy.Spec.TargetRef.Name)
			gwNs := policy.Namespace
			if policy.Spec.TargetRef.Namespace != nil {
				gwNs = string(*policy.Spec.TargetRef.Namespace)
			}

			gateway, err := w.gatewayAPIClient.GatewayV1().Gateways(gwNs).Get(context.TODO(), gwName, metav1.GetOptions{})
			if err != nil {
				path := field.NewPath("spec").Child("targetRef")
				if errors.IsNotFound(err) {
					errs = append(errs, field.Invalid(path, policy.Spec.TargetRef, fmt.Sprintf("Gateway %s/%s not found", gwNs, gwName)))
					return errs
				}

				errs = append(errs, field.Invalid(path, policy.Spec.TargetRef, fmt.Sprintf("Failed to get Gateway %s/%s: %v", gwNs, gwName, err)))
				return errs
			}

			for i, p := range policy.Spec.Ports {
				listener := webhook.GetListenerIfHasMatchingPort(p.Port, gateway.Spec.Listeners)
				if listener == nil {
					path := field.NewPath("spec").Child("ports").Index(i).Child("port")
					errs = append(errs, field.Invalid(path, p.Port, fmt.Sprintf("port %d is not defined in Gateway %s/%s", p.Port, gwNs, gwName)))
					continue
				}

				if listener.Protocol != gwv1.HTTPSProtocolType &&
					listener.Protocol != gwv1.TLSProtocolType &&
					listener.Protocol != gwv1.TCPProtocolType &&
					listener.Protocol != gwv1.HTTPProtocolType {
					path := field.NewPath("spec").Child("ports").Index(i).Child("port")
					errs = append(errs, field.Invalid(path, p.Port, fmt.Sprintf("Protocol of port %d is %s, it must be HTTP, HTTPS, TLS or TCP in Gateway %s/%s", p.Port, listener.Protocol, gwNs, gwName)))
					continue
				}
			}
		}
	}
	return errs
}

func validateL7RateLimits(policy *gwpav1alpha1.RateLimitPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind || policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind) {
		if len(policy.Spec.Ports) > 0 {
			path := field.NewPath("spec").Child("ports")
			errs = append(errs, field.Invalid(path, policy.Spec.Ports, "must be empty for HTTPRoute/GRPCRoute target"))
		}

		if policy.Spec.DefaultBPS != nil {
			path := field.NewPath("spec").Child("bps")
			errs = append(errs, field.Invalid(path, policy.Spec.DefaultBPS, "must not be set for HTTPRoute/GRPCRoute target"))
		}

		if len(policy.Spec.Hostnames) == 0 && len(policy.Spec.HTTPRateLimits) == 0 && len(policy.Spec.GRPCRateLimits) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, policy.Spec, "any one of hostnames, http or grpc must be set for HTTPRoute/GRPCRoute target"))
		}

		if len(policy.Spec.Hostnames) > 0 {
			errs = append(errs, validateHostnames(policy)...)
		}

		if policy.Spec.DefaultConfig == nil {
			if len(policy.Spec.Hostnames) > 0 {
				path := field.NewPath("spec").Child("hostnames")
				for i, h := range policy.Spec.Hostnames {
					if h.Config == nil {
						errs = append(errs, field.Required(path.Index(i).Child("rateLimit"), fmt.Sprintf("rateLimit must be set for hostname %q, as there's no default rate limit", h.Hostname)))
					}
				}
			}

			if len(policy.Spec.HTTPRateLimits) > 0 {
				path := field.NewPath("spec").Child("http")
				for i, h := range policy.Spec.HTTPRateLimits {
					if h.Config == nil {
						errs = append(errs, field.Required(path.Index(i).Child("rateLimit"), "rateLimit must be set, as there's no default rate limit"))
					}
				}
			}

			if len(policy.Spec.GRPCRateLimits) > 0 {
				path := field.NewPath("spec").Child("grpc")
				for i, g := range policy.Spec.GRPCRateLimits {
					if g.Config == nil {
						errs = append(errs, field.Required(path.Index(i).Child("rateLimit"), "rateLimit must be set, as there's no default rate limit"))
					}
				}
			}
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind {
		if len(policy.Spec.HTTPRateLimits) == 0 && len(policy.Spec.Hostnames) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, nil, "either hostnames or http must be set for HTTPRoute target"))
		}

		if len(policy.Spec.GRPCRateLimits) > 0 {
			path := field.NewPath("spec").Child("grpc")
			errs = append(errs, field.Invalid(path, policy.Spec.GRPCRateLimits, "must be empty for HTTPRoute target"))
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind {
		if len(policy.Spec.GRPCRateLimits) == 0 && len(policy.Spec.Hostnames) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, nil, "either hostnames or grpc must be set for GRPCRoute target"))
		}

		if len(policy.Spec.HTTPRateLimits) > 0 {
			path := field.NewPath("spec").Child("http")
			errs = append(errs, field.Invalid(path, policy.Spec.HTTPRateLimits, "must be empty for GRPCRoute target"))
		}
	}

	return errs
}

func validateHostnames(policy *gwpav1alpha1.RateLimitPolicy) field.ErrorList {
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
