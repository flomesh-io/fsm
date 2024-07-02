package v1alpha2

import (
	"context"
	"fmt"
	"regexp"

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

type RetryPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewRetryPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &RetryPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwpav1alpha2.RetryPolicy{}).
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

func (r *RetryPolicyWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *RetryPolicyWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *RetryPolicyWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	policy, ok := obj.(*gwpav1alpha2.RetryPolicy)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errorList := r.validateTargetRefs(policy.Spec.TargetRefs)
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	errorList = append(errorList, r.validateConfig(policy)...)
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}

func (r *RetryPolicyWebhook) validateTargetRefs(refs []gwv1alpha2.NamespacedPolicyTargetReference) field.ErrorList {
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

func (r *RetryPolicyWebhook) validateConfig(policy *gwpav1alpha2.RetryPolicy) field.ErrorList {
	var errs field.ErrorList

	if len(policy.Spec.Ports) == 0 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "cannot be empty"))
	}

	if len(policy.Spec.Ports) > 16 {
		path := field.NewPath("spec").Child("ports")
		errs = append(errs, field.Invalid(path, policy.Spec.Ports, "max port items cannot be greater than 16"))
	}

	if policy.Spec.DefaultRetry == nil {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			if port.Retry == nil {
				errs = append(errs, field.Required(path.Index(i).Child("retry"), fmt.Sprintf("retry must be set for port %d, as there's no default retry", port.Port)))
			}
		}
	}

	if policy.Spec.DefaultRetry != nil {
		path := field.NewPath("spec").Child("retry")
		errs = append(errs, r.validateRetryConfig(path, policy.Spec.DefaultRetry)...)
	}

	if len(policy.Spec.Ports) > 0 {
		path := field.NewPath("spec").Child("ports")
		for i, port := range policy.Spec.Ports {
			errs = append(errs, r.validateRetryConfig(path.Index(i).Child("retry"), port.Retry)...)
		}
	}

	return errs
}

// retryOnMaxLength is the maximum length each of the retryOn item
const retryOnMaxLength int = 3

const retryOnStatusCodeFmt = "[1-9]?[0-9][0-9]|[1-9][x][x]"
const retryOnStatusCodeErrorMsg = "length of status code must be 3, with leading digit and last 2 digits being 0-9 or x, i.e. 5xx, 500, 502, 503, 504"

var retryOnStatusCodeFmtRegexp = regexp.MustCompile("^" + retryOnStatusCodeFmt + "$")

func (r *RetryPolicyWebhook) validateRetryConfig(path *field.Path, config *gwpav1alpha2.RetryConfig) field.ErrorList {
	var errs field.ErrorList

	if len(config.RetryOn) > 0 {
		for _, code := range config.RetryOn {
			if len(code) > retryOnMaxLength {
				errs = append(errs, field.TooLongMaxLength(path.Child("retryOn"), code, retryOnMaxLength))
			}
			if !retryOnStatusCodeFmtRegexp.MatchString(code) {
				errs = append(errs, field.Invalid(path.Child("retryOn"), code, retryOnStatusCodeErrorMsg))
			}
		}
	} else {
		errs = append(errs, field.Required(path.Child("retryOn"), "retryOn cannot be empty"))
	}

	return errs
}
