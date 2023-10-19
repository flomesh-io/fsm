package ratelimit

import (
	"fmt"
	"net/http"

	"k8s.io/utils/pointer"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/webhook"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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

	if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && policy.Spec.TargetRef.Kind == "Gateway" {
		// do nothing
	}

	if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && (policy.Spec.TargetRef.Kind == "HTTPRoute" || policy.Spec.TargetRef.Kind == "GRPCRoute") {
		if policy.Spec.HostnameBasedRateLimit != nil {
			if policy.Spec.HostnameBasedRateLimit.Mode == nil {
				policy.Spec.HostnameBasedRateLimit.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
			}

			if policy.Spec.HostnameBasedRateLimit.Backlog == nil {
				policy.Spec.HostnameBasedRateLimit.Backlog = pointer.Int(10)
			}

			if policy.Spec.HostnameBasedRateLimit.Burst == nil {
				policy.Spec.HostnameBasedRateLimit.Burst = &policy.Spec.HostnameBasedRateLimit.Requests
			}
		}

		if policy.Spec.RouteBasedRateLimit != nil {
			if policy.Spec.RouteBasedRateLimit.Mode == nil {
				policy.Spec.RouteBasedRateLimit.Mode = rateLimitPolicyModePointer(gwpav1alpha1.RateLimitPolicyModeLocal)
			}

			if policy.Spec.RouteBasedRateLimit.Backlog == nil {
				policy.Spec.RouteBasedRateLimit.Backlog = pointer.Int(10)
			}

			if policy.Spec.RouteBasedRateLimit.Burst == nil {
				policy.Spec.RouteBasedRateLimit.Burst = &policy.Spec.RouteBasedRateLimit.Requests
			}
		}
	}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
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

	if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && policy.Spec.TargetRef.Kind == "Gateway" {
		if policy.Spec.Match.Port == nil {
			return fmt.Errorf("port is required for TargetRef Gateway")
		}

		if policy.Spec.RateLimit.L4RateLimit == nil {
			return fmt.Errorf("bps is required for TargetRef Gateway")
		}

		if policy.Spec.RateLimit.L7RateLimit != nil {
			return fmt.Errorf("either hostnameBasedRateLimit or routeBasedRateLimit is required for TargetRef Gateway")
		}
	}

	if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && (policy.Spec.TargetRef.Kind == "HTTPRoute" || policy.Spec.TargetRef.Kind == "GRPCRoute") {

		if policy.Spec.RateLimit.L7RateLimit == nil {
			return fmt.Errorf("either hostnameBasedRateLimit or routeBasedRateLimit is required for TargetRef HTTPRoute or GRPCRoute")
		}
	}

	return nil
}
