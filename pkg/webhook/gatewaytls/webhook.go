package gatewaytls

import (
	"context"
	"fmt"
	"net/http"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	corev1 "k8s.io/api/core/v1"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	log = logger.New("webhook/gatewaytls")
)

type register struct {
	*webhook.RegisterConfig
	gatewayAPIClient gatewayApiClientset.Interface
}

// NewRegister creates a new GatewayTLSPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig:   cfg,
		gatewayAPIClient: gatewayApiClientset.NewForConfigOrDie(cfg.KubeConfig),
	}
}

// GetWebhooks returns the webhooks to be registered for GatewayTLSPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.FlomeshGatewayAPIGroup},
		[]string{"v1alpha1"},
		[]string{"gatewaytlspolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mgatewaytlspolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GatewayTLSPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vgatewaytlspolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GatewayTLSPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for GatewayTLSPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.GatewayTLSPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Configurator)),
		constants.GatewayTLSPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient, r.gatewayAPIClient)),
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
	return &gwpav1alpha1.GatewayTLSPolicy{}
}

// SetDefaults sets the default values for the GatewayTLSPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.GatewayTLSPolicy)
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
	//			len(policy.Spec.HTTPGatewayTLSs) > 0 ||
	//			len(policy.Spec.GRPCGatewayTLSs) > 0 {
	//			setDefaults(policy)
	//		}
	//	}
	//}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

type validator struct {
	kubeClient       kubernetes.Interface
	gatewayAPIClient gatewayApiClientset.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.GatewayTLSPolicy{}
}

// ValidateCreate validates the creation of the GatewayTLSPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of the GatewayTLSPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the GatewayTLSPolicy
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
	policy, ok := obj.(*gwpav1alpha1.GatewayTLSPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, validateSpec(policy)...)
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
	case constants.GatewayAPIGatewayKind:
		// do nothing
	default:
		path := field.NewPath("spec").Child("targetRef").Child("kind")
		errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Gateway"))
	}

	return errs
}

func (w *validator) validateConfig(policy *gwpav1alpha1.GatewayTLSPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIGatewayKind {
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

				if listener.Protocol != gwv1.HTTPSProtocolType && listener.Protocol != gwv1.TLSProtocolType {
					path := field.NewPath("spec").Child("ports").Index(i).Child("port")
					errs = append(errs, field.Invalid(path, p.Port, fmt.Sprintf("Protocol of port %d is %s, it must be HTTPS or TLS in Gateway %s/%s", p.Port, listener.Protocol, gwNs, gwName)))
					continue
				}

				if p.Config == nil {
					continue
				}

				path := field.NewPath("spec").Child("ports").Index(i).Child("config")
				errs = append(errs, w.validateCert(path, p.Config, listener, gateway.Namespace)...)
			}
		}
	}

	return errs
}

func (w *validator) validateCert(path *field.Path, config *gwpav1alpha1.GatewayTLSConfig, listener *gwv1.Listener, gwNamespace string) field.ErrorList {
	var errs field.ErrorList

	if config.MTLS != nil && *config.MTLS {
		switch listener.Protocol {
		case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
			if listener.TLS == nil {
				errs = append(errs, field.Invalid(path, config, fmt.Sprintf("mTLS is not supported for listener port %d, as there's no TLS configuration, please check gateway spec.", listener.Port)))
				return errs
			}

			if listener.TLS.Mode != nil && *listener.TLS.Mode != gwv1.TLSModeTerminate {
				errs = append(errs, field.Invalid(path, config, fmt.Sprintf("mTLS is not supported for listener port %d, as TLS mode is %s, please check gateway spec.", listener.Port, *listener.TLS.Mode)))
				return errs
			}

			if len(listener.TLS.CertificateRefs) == 0 {
				errs = append(errs, field.Invalid(path, config, fmt.Sprintf("mTLS is not supported for listener port %d, as there's no certificateRefs, please check gateway spec.", listener.Port)))
				return errs
			}

			for _, certRef := range listener.TLS.CertificateRefs {
				errs = append(errs, w.validateSecrets(path, certRef, listener, gwNamespace)...)
			}
		default:
			errs = append(errs, field.Invalid(path, config, fmt.Sprintf("mTLS is not supported for listener protocol %s", listener.Protocol)))
		}
	}

	return errs
}

func validateSpec(policy *gwpav1alpha1.GatewayTLSPolicy) field.ErrorList {
	var errs field.ErrorList

	if policy.Spec.TargetRef.Group == constants.GatewayAPIGroup &&
		policy.Spec.TargetRef.Kind == constants.GatewayAPIGatewayKind {
		if len(policy.Spec.Ports) == 0 {
			path := field.NewPath("spec").Child("ports")
			errs = append(errs, field.Invalid(path, policy.Spec.Ports, "cannot be empty for Gateway target"))
		}

		if policy.Spec.DefaultConfig == nil {
			for i, port := range policy.Spec.Ports {
				if port.Config == nil {
					path := field.NewPath("spec").Child("ports").Index(i).Child("config")
					errs = append(errs, field.Required(path, fmt.Sprintf("config must be set for port %d, as there's no default config", port.Port)))
				}
			}
		}
	}

	return errs
}

func (w *validator) validateSecrets(path *field.Path, ref gwv1.SecretObjectReference, listener *gwv1.Listener, gwNamespace string) field.ErrorList {
	var errs field.ErrorList

	if string(*ref.Kind) == constants.KubernetesSecretKind && string(*ref.Group) == constants.KubernetesCoreGroup {
		ns := gwutils.Namespace(ref.Namespace, gwNamespace)
		name := string(ref.Name)

		secret, err := w.kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, field.NotFound(path, fmt.Sprintf("Failed to get Secret %s/%s for listener port %d: %s", ns, name, listener.Port, err)))
			return errs
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

		v, ok = secret.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			if string(v) == "" {
				errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("mTLS is enabled, but the content of Secret %s/%s by key %s is empty", ns, name, corev1.ServiceAccountRootCAKey)))
			}
		} else {
			errs = append(errs, field.NotFound(path, fmt.Sprintf("mTLS is enabled, but Secret %s/%s doesn't have required data by key %s", ns, name, corev1.ServiceAccountRootCAKey)))
		}
	}

	return errs
}
