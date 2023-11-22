package accesscontrol

import (
	"context"
	"fmt"
	"net/http"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	log = logger.New("webhook/accesscontrol")
)

type register struct {
	*webhook.RegisterConfig
	gatewayAPIClient gatewayApiClientset.Interface
}

// NewRegister creates a new AccessControlPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig:   cfg,
		gatewayAPIClient: gatewayApiClientset.NewForConfigOrDie(cfg.KubeConfig),
	}
}

// GetWebhooks returns the webhooks to be registered for AccessControlPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.FlomeshGatewayAPIGroup},
		[]string{"v1alpha1"},
		[]string{"accesscontrolpolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"maccesscontrolpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.AccessControlPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vaccesscontrolpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.AccessControlPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for AccessControlPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.AccessControlPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.AccessControlPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient, r.gatewayAPIClient)),
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
	return &gwpav1alpha1.AccessControlPolicy{}
}

// SetDefaults sets the default values for the AccessControlPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.AccessControlPolicy)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", policy.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", policy.Spec)

	//if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup {
	//	if policy.Spec.TargetRef.Kind == constants.GatewayAPIGatewayKind ||
	//		policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind ||
	//		policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind {
	//		if len(policy.Spec.Ports) > 0 ||
	//			len(policy.Spec.Hostnames) > 0 ||
	//			len(policy.Spec.HTTPAccessControls) > 0 ||
	//			len(policy.Spec.GRPCAccessControls) > 0 {
	//			setDefaults(policy)
	//		}
	//	}
	//}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

//func setDefaults(policy *gwpav1alpha1.AccessControlPolicy) {
//	if len(policy.Spec.Ports) > 0 {
//		for i, port := range policy.Spec.Ports {
//			if port.Config != nil {
//				policy.Spec.Ports[i].Config = configDefaults(port.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if len(policy.Spec.Hostnames) > 0 {
//		for i, hostname := range policy.Spec.Hostnames {
//			if hostname.Config != nil {
//				policy.Spec.Hostnames[i].Config = configDefaults(hostname.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if len(policy.Spec.HTTPAccessControls) > 0 {
//		for i, hr := range policy.Spec.HTTPAccessControls {
//			if hr.Config != nil {
//				policy.Spec.HTTPAccessControls[i].Config = configDefaults(hr.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if len(policy.Spec.GRPCAccessControls) > 0 {
//		for i, gr := range policy.Spec.GRPCAccessControls {
//			if gr.Config != nil {
//				policy.Spec.GRPCAccessControls[i].Config = configDefaults(gr.Config, policy.Spec.DefaultConfig)
//			}
//		}
//	}
//
//	if policy.Spec.DefaultConfig != nil {
//		policy.Spec.DefaultConfig = setDefaultValues(policy.Spec.DefaultConfig)
//	}
//}
//
//func configDefaults(config *gwpav1alpha1.AccessControlConfig, defaultConfig *gwpav1alpha1.AccessControlConfig) *gwpav1alpha1.AccessControlConfig {
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
//func mergeConfig(config *gwpav1alpha1.AccessControlConfig, defaultConfig *gwpav1alpha1.AccessControlConfig) *gwpav1alpha1.AccessControlConfig {
//	cfgCopy := config.DeepCopy()
//
//	if cfgCopy.EnableXFF == nil {
//		if defaultConfig.EnableXFF != nil {
//			// use port config
//			cfgCopy.EnableXFF = defaultConfig.EnableXFF
//		} else {
//			// all nil, set to false
//			cfgCopy.EnableXFF = pointer.Bool(false)
//		}
//	}
//
//	if cfgCopy.Message == nil {
//		if defaultConfig.Message != nil {
//			// use port config
//			cfgCopy.Message = defaultConfig.Message
//		} else {
//			// all nil, set to false
//			cfgCopy.Message = pointer.String("")
//		}
//	}
//
//	if cfgCopy.StatusCode == nil {
//		if defaultConfig.StatusCode != nil {
//			// use port config
//			cfgCopy.StatusCode = defaultConfig.StatusCode
//		} else {
//			// all nil, set to false
//			cfgCopy.StatusCode = pointer.Int32(403)
//		}
//	}
//
//	return cfgCopy
//}
//
//func setDefaultValues(config *gwpav1alpha1.AccessControlConfig) *gwpav1alpha1.AccessControlConfig {
//	cfg := config.DeepCopy()
//
//	if cfg.EnableXFF == nil {
//		cfg.EnableXFF = pointer.Bool(false)
//	}
//
//	if cfg.StatusCode == nil {
//		cfg.StatusCode = pointer.Int32(403)
//	}
//
//	if cfg.Message == nil {
//		cfg.Message = pointer.String("")
//	}
//
//	return cfg
//}

type validator struct {
	kubeClient       kubernetes.Interface
	gatewayAPIClient gatewayApiClientset.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.AccessControlPolicy{}
}

// ValidateCreate validates the creation of the AccessControlPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of the AccessControlPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the AccessControlPolicy
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
	policy, ok := obj.(*gwpav1alpha1.AccessControlPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, w.validateSpec(policy)...)
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
	case constants.GatewayAPIGatewayKind, constants.GatewayAPIHTTPRouteKind, constants.GatewayAPIGRPCRouteKind:
		// do nothing
	default:
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Gateway, HTTPRoute or GRPCRoute"))
	}

	return errs
}

func (w *validator) validateSpec(policy *gwpav1alpha1.AccessControlPolicy) field.ErrorList {
	errs := w.validateL4AccessControl(policy)
	errs = append(errs, validateL7AccessControl(policy)...)
	errs = append(errs, validateConfig(policy)...)

	return errs
}

func validateConfig(policy *gwpav1alpha1.AccessControlPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.GatewayAPIGatewayKind ||
			policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind ||
			policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind) {
		if policy.Spec.DefaultConfig != nil {
			path := field.NewPath("spec").Child("config")
			errs = append(errs, validateACLs(path, policy.Spec.DefaultConfig)...)
			errs = append(errs, validateIPAddresses(path, policy.Spec.DefaultConfig)...)
		}

		if len(policy.Spec.Ports) > 0 {
			for i, p := range policy.Spec.Ports {
				if p.Config != nil {
					path := field.NewPath("spec").Child("ports").Index(i).Child("config")
					errs = append(errs, validateACLs(path, p.Config)...)
					errs = append(errs, validateIPAddresses(path, p.Config)...)
				}
			}
		}

		if len(policy.Spec.Hostnames) > 0 {
			for i, h := range policy.Spec.Hostnames {
				if h.Config != nil {
					path := field.NewPath("spec").Child("hostnames").Index(i).Child("config")
					errs = append(errs, validateACLs(path, h.Config)...)
					errs = append(errs, validateIPAddresses(path, h.Config)...)
				}
			}
		}

		if len(policy.Spec.HTTPAccessControls) > 0 {
			for i, h := range policy.Spec.HTTPAccessControls {
				if h.Config != nil {
					path := field.NewPath("spec").Child("http").Index(i).Child("config")
					errs = append(errs, validateACLs(path, h.Config)...)
					errs = append(errs, validateIPAddresses(path, h.Config)...)
				}
			}
		}

		if len(policy.Spec.GRPCAccessControls) > 0 {
			for i, g := range policy.Spec.GRPCAccessControls {
				if g.Config != nil {
					path := field.NewPath("spec").Child("grpc").Index(i).Child("config")
					errs = append(errs, validateACLs(path, g.Config)...)
					errs = append(errs, validateIPAddresses(path, g.Config)...)
				}
			}
		}
	}

	return errs
}

func validateACLs(path *field.Path, config *gwpav1alpha1.AccessControlConfig) field.ErrorList {
	var errs field.ErrorList

	if len(config.Blacklist) == 0 && len(config.Whitelist) == 0 {
		errs = append(errs, field.Invalid(path, config, "blacklist and whitelist cannot be empty at the same time"))
	}

	return errs
}

func (w *validator) validateL4AccessControl(policy *gwpav1alpha1.AccessControlPolicy) field.ErrorList {
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

		if len(policy.Spec.HTTPAccessControls) > 0 {
			path := field.NewPath("spec").Child("http")
			errs = append(errs, field.Invalid(path, policy.Spec.HTTPAccessControls, "must be empty for Gateway target"))
		}

		if len(policy.Spec.GRPCAccessControls) > 0 {
			path := field.NewPath("spec").Child("grpc")
			errs = append(errs, field.Invalid(path, policy.Spec.GRPCAccessControls, "must be empty for Gateway target"))
		}

		if policy.Spec.DefaultConfig == nil {
			for i, port := range policy.Spec.Ports {
				if port.Config == nil {
					path := field.NewPath("spec").Child("ports").Index(i).Child("config")
					errs = append(errs, field.Required(path, fmt.Sprintf("config must be set for port %d, as there's no default config", port.Port)))
				}
			}
		}

		if len(policy.Spec.Ports) > 0 {
			gwName := string(policy.Spec.TargetRef.Name)
			gwNs := policy.Namespace
			if policy.Spec.TargetRef.Namespace != nil {
				gwNs = string(*policy.Spec.TargetRef.Namespace)
			}

			gateway, err := w.gatewayAPIClient.GatewayV1beta1().Gateways(gwNs).Get(context.TODO(), gwName, metav1.GetOptions{})
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

				if listener.Protocol != gwv1beta1.HTTPSProtocolType &&
					listener.Protocol != gwv1beta1.TLSProtocolType &&
					listener.Protocol != gwv1beta1.TCPProtocolType &&
					listener.Protocol != gwv1beta1.UDPProtocolType &&
					listener.Protocol != gwv1beta1.HTTPProtocolType {
					path := field.NewPath("spec").Child("ports").Index(i).Child("port")
					errs = append(errs, field.Invalid(path, p.Port, fmt.Sprintf("Protocol of port %d is %s, it must be HTTP, HTTPS, TLS, TCP or UDP in Gateway %s/%s", p.Port, listener.Protocol, gwNs, gwName)))
					continue
				}
			}
		}
	}

	return errs
}

func validateIPAddresses(path *field.Path, config *gwpav1alpha1.AccessControlConfig) field.ErrorList {
	var errs field.ErrorList

	if len(config.Blacklist) > 0 {
		for i, ip := range config.Blacklist {
			if err := webhook.IsValidIPOrCIDR(ip); err != nil {
				errs = append(errs, field.Invalid(path.Child("blacklist").Index(i), ip, fmt.Sprintf("%s", err)))
			}
		}
	}

	if len(config.Whitelist) > 0 {
		for i, ip := range config.Whitelist {
			if err := webhook.IsValidIPOrCIDR(ip); err != nil {
				errs = append(errs, field.Invalid(path.Child("whitelist").Index(i), ip, fmt.Sprintf("%s", err)))
			}
		}
	}

	return errs
}

func validateL7AccessControl(policy *gwpav1alpha1.AccessControlPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		(policy.Spec.TargetRef.Kind == constants.GatewayAPIHTTPRouteKind || policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind) {
		if len(policy.Spec.Ports) > 0 {
			path := field.NewPath("spec").Child("ports")
			errs = append(errs, field.Invalid(path, policy.Spec.Ports, "must be empty for HTTPRoute/GRPCRoute target"))
		}

		if len(policy.Spec.Hostnames) == 0 && len(policy.Spec.HTTPAccessControls) == 0 && len(policy.Spec.GRPCAccessControls) == 0 {
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

			if len(policy.Spec.HTTPAccessControls) > 0 {
				for i, h := range policy.Spec.HTTPAccessControls {
					if h.Config == nil {
						path := field.NewPath("spec").Child("http").Index(i).Child("config")
						errs = append(errs, field.Required(path, "config must be set, as there's no default config"))
					}
				}
			}

			if len(policy.Spec.GRPCAccessControls) > 0 {
				for i, g := range policy.Spec.GRPCAccessControls {
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
		if len(policy.Spec.HTTPAccessControls) == 0 && len(policy.Spec.Hostnames) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, nil, "either hostnames or http must be set for HTTPRoute target"))
		}

		if len(policy.Spec.GRPCAccessControls) > 0 {
			path := field.NewPath("spec").Child("grpc")
			errs = append(errs, field.Invalid(path, policy.Spec.GRPCAccessControls, "must be empty for HTTPRoute target"))
		}
	}

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIGRPCRouteKind {
		if len(policy.Spec.GRPCAccessControls) == 0 && len(policy.Spec.Hostnames) == 0 {
			path := field.NewPath("spec")
			errs = append(errs, field.Invalid(path, nil, "either hostnames or grpc must be set for GRPCRoute target"))
		}

		if len(policy.Spec.HTTPAccessControls) > 0 {
			path := field.NewPath("spec").Child("http")
			errs = append(errs, field.Invalid(path, policy.Spec.HTTPAccessControls, "must be empty for GRPCRoute target"))
		}
	}

	return errs
}

func validateHostnames(policy *gwpav1alpha1.AccessControlPolicy) field.ErrorList {
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
