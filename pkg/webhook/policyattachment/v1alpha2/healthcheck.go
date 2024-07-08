package v1alpha2

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"k8s.io/apimachinery/pkg/util/validation/field"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/webhook"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type HealthCheckPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewHealthCheckPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &HealthCheckPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwpav1alpha2.HealthCheckPolicy{}).
		WithWebhookServiceName(cfg.WebhookSvcName).
		WithWebhookServiceNamespace(cfg.WebhookSvcNs).
		WithCABundle(cfg.CaBundle).
		Complete(); err != nil {
		return nil
	} else {
		r.CfgBuilder = blder
	}

	return r
}

func (r *HealthCheckPolicyWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *HealthCheckPolicyWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *HealthCheckPolicyWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	policy, ok := obj.(*gwpav1alpha2.HealthCheckPolicy)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errorList := r.validateTargetRefs(policy.Spec.TargetRefs)
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, r.validateSpec(policy)...)
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}

func (r *HealthCheckPolicyWebhook) validateTargetRefs(refs []gwv1alpha2.NamespacedPolicyTargetReference) field.ErrorList {
	var errs field.ErrorList

	for i, ref := range refs {
		if ref.Group != constants.KubernetesCoreGroup && ref.Group != constants.FlomeshMCSAPIGroup {
			path := field.NewPath("spec").Child("targetRefs").Index(i).Child("group")
			errs = append(errs, field.Invalid(path, ref.Group, "group must be set to flomesh.io or core"))
		}

		if (ref.Group == constants.KubernetesCoreGroup && ref.Kind == constants.KubernetesServiceKind) ||
			(ref.Group == constants.FlomeshMCSAPIGroup && ref.Kind == constants.FlomeshAPIServiceImportKind) {
			// do nothing
		} else {
			path := field.NewPath("spec").Child("targetRefs").Index(i).Child("kind")
			errs = append(errs, field.Invalid(path, ref.Kind, "kind must be set to Service for group core or ServiceImport for group flomesh.io"))
		}
	}

	return errs
}

func (r *HealthCheckPolicyWebhook) validateSpec(policy *gwpav1alpha2.HealthCheckPolicy) field.ErrorList {
	var errs field.ErrorList

	if len(policy.Spec.Ports) == 0 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "cannot be empty"))
	}

	if len(policy.Spec.Ports) > 16 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "max port items cannot be greater than 16"))
	}

	if policy.Spec.DefaultHealthCheck == nil {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			if port.HealthCheck == nil {
				errs = append(errs, field.Required(path.Index(i).Child("healthCheck"), fmt.Sprintf("healthCheck must be set for port %d, as there's no default healthCheck", port.Port)))
			}
		}
	}

	if policy.Spec.DefaultHealthCheck != nil {
		errs = r.validateConfig(field.NewPath("spec").Child("healthCheck"), policy.Spec.DefaultHealthCheck)
	}

	if len(policy.Spec.Ports) > 0 {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			if port.HealthCheck != nil {
				errs = append(errs, r.validateConfig(path.Index(i).Child("healthCheck"), port.HealthCheck)...)
			}
		}
	}

	return errs
}

func (r *HealthCheckPolicyWebhook) validateConfig(path *field.Path, config *gwpav1alpha2.HealthCheckConfig) field.ErrorList {
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
