package v1alpha2

import (
	"context"
	"fmt"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/flomesh-io/fsm/pkg/webhook"
)

type BackendLBPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewBackendLBPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &BackendLBPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwpav1alpha2.BackendLBPolicy{}).
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

func (r *BackendLBPolicyWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *BackendLBPolicyWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *BackendLBPolicyWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	policy, ok := obj.(*gwpav1alpha2.BackendLBPolicy)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errorList := r.validateTargetRefs(policy.Spec.TargetRefs)
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	if !version.IsCELValidationEnabled(r.KubeClient) {
		errorList = append(errorList, validateBackendLB(policy)...)
	}

	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}

func (r *BackendLBPolicyWebhook) validateTargetRefs(refs []gwv1alpha2.LocalPolicyTargetReference) field.ErrorList {
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

func validateBackendLB(policy *gwpav1alpha2.BackendLBPolicy) field.ErrorList {
	return validateBackendLBSpec(&policy.Spec, field.NewPath("spec"))
}

func validateBackendLBSpec(spec *gwpav1alpha2.BackendLBPolicySpec, path *field.Path) field.ErrorList {
	if spec.SessionPersistence == nil {
		return nil
	}

	if (spec.SessionPersistence.Type == nil || *spec.SessionPersistence.Type == gwv1.CookieBasedSessionPersistence) && spec.SessionPersistence.CookieConfig != nil {
		if spec.SessionPersistence.CookieConfig.LifetimeType == nil {
			return nil
		}

		switch *spec.SessionPersistence.CookieConfig.LifetimeType {
		case gwv1.SessionCookieLifetimeType:
			return nil
		case gwv1.PermanentCookieLifetimeType:
			if spec.SessionPersistence.AbsoluteTimeout == nil {
				return field.ErrorList{field.Invalid(path.Child("sessionPersistence", "absoluteTimeout"), spec.SessionPersistence.AbsoluteTimeout, "AbsoluteTimeout must be specified when cookie lifetimeType is Permanent")}
			}
		}
	}

	return nil
}
