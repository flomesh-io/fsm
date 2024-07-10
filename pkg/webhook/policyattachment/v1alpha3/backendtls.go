package v1alpha3

import (
	"context"
	"fmt"

	"github.com/flomesh-io/fsm/pkg/version"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/webhook"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type BackendTLSPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewBackendTLSPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &BackendTLSPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1alpha3.BackendTLSPolicy{}).
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

func (r *BackendTLSPolicyWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *BackendTLSPolicyWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *BackendTLSPolicyWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	policy, ok := obj.(*gwv1alpha3.BackendTLSPolicy)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errorList := r.validateTargetRefs(policy.Spec.TargetRefs)
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	if !version.IsCELValidationEnabled(r.KubeClient) {
		errorList = append(errorList, validateBackendTLS(policy)...)
	}
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}
func (r *BackendTLSPolicyWebhook) validateTargetRefs(refs []gwv1alpha2.LocalPolicyTargetReferenceWithSectionName) field.ErrorList {
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

func validateBackendTLS(policy *gwv1alpha3.BackendTLSPolicy) field.ErrorList {
	return validateBackendTLSSpec(&policy.Spec, field.NewPath("spec"))
}

func validateBackendTLSSpec(spec *gwv1alpha3.BackendTLSPolicySpec, path *field.Path) field.ErrorList {
	if len(spec.Validation.CACertificateRefs) == 0 &&
		(spec.Validation.WellKnownCACertificates == nil || *spec.Validation.WellKnownCACertificates == "") {
		return field.ErrorList{field.Invalid(path.Child("validation"), spec.Validation, "Either validation.wellKnownCACertificates or validation.caCertificateRefs must be set")}
	}

	if len(spec.Validation.CACertificateRefs) != 0 && spec.Validation.WellKnownCACertificates != nil && *spec.Validation.WellKnownCACertificates != "" {
		return field.ErrorList{field.Invalid(path.Child("validation"), spec.Validation, "Only one of validation.caCertificateRefs or validation.wellKnownCACertificates may be specified, not both")}
	}

	return nil
}
