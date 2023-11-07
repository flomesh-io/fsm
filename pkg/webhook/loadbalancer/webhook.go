package loadbalancer

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
	log = logger.New("webhook/loadbalancer")
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new LoadBalancerPolicy webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for LoadBalancerPolicy
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"gateway.flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"loadbalancerpolicies"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mloadbalancerpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.LoadBalancerPolicyMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vloadbalancerpolicy.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.LoadBalancerPolicyValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for LoadBalancerPolicy
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.LoadBalancerPolicyMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.LoadBalancerPolicyValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
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
	return &gwpav1alpha1.LoadBalancerPolicy{}
}

// SetDefaults sets the default values for the LoadBalancerPolicy
func (w *defaulter) SetDefaults(obj interface{}) {
	policy, ok := obj.(*gwpav1alpha1.LoadBalancerPolicy)
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
				if p.Type == nil {
					if policy.Spec.DefaultType == nil {
						policy.Spec.Ports[i].Type = loadBalancerType(gwpav1alpha1.RoundRobinLoadBalancer)
					} else {
						policy.Spec.Ports[i].Type = policy.Spec.DefaultType
					}
				}
			}
		}
	}

	log.Debug().Msgf("After setting default values, spec=%v", policy.Spec)
}

func loadBalancerType(t gwpav1alpha1.LoadBalancerType) *gwpav1alpha1.LoadBalancerType {
	return &t
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwpav1alpha1.LoadBalancerPolicy{}
}

// ValidateCreate validates the creation of the LoadBalancerPolicy
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update of the LoadBalancerPolicy
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the deletion of the LoadBalancerPolicy
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	policy, ok := obj.(*gwpav1alpha1.LoadBalancerPolicy)
	if !ok {
		return nil
	}

	errorList := validateTargetRef(policy.Spec.TargetRef)
	errorList = append(errorList, validateConfig(policy)...)
	// TODO: validate ports exist in the referenced service

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

	return errs
}

func validateConfig(policy *gwpav1alpha1.LoadBalancerPolicy) field.ErrorList {
	var errs field.ErrorList

	if len(policy.Spec.Ports) == 0 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "cannot be empty"))
	}

	if len(policy.Spec.Ports) > 16 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "max port items cannot be greater than 16"))
	}

	if policy.Spec.DefaultType == nil {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			if port.Type == nil {
				errs = append(errs, field.Required(path.Index(i).Child("type"), fmt.Sprintf("type must be set for port %d, as there's no default type", port.Port)))
			}
		}
	}

	return errs
}
