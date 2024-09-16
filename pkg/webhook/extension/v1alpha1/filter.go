package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type FilterWebhook struct {
	webhook.DefaultWebhook
}

func NewFilterWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &FilterWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.Filter{}).
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

func (r *FilterWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.Filter)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *FilterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *FilterWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *FilterWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	filter, ok := obj.(*extv1alpha1.Filter)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errs := r.validateSpec(ctx, filter.Spec, field.NewPath("spec"))

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *FilterWebhook) validateSpec(ctx context.Context, spec extv1alpha1.FilterSpec, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	errs = append(errs, r.validateDefinitionRef(ctx, spec.DefinitionRef, path.Child("definitionRef"))...)
	errs = append(errs, r.validateConfigRef(ctx, spec.ConfigRef, path.Child("configRef"))...)

	return errs
}

func (r *FilterWebhook) validateDefinitionRef(ctx context.Context, definitionRef *gwv1.LocalObjectReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	if definitionRef != nil {
		if definitionRef.Group != extv1alpha1.GroupName {
			errs = append(errs, field.Invalid(path.Child("group"), definitionRef.Group, fmt.Sprintf("group must be %s", extv1alpha1.GroupName)))
		}

		if definitionRef.Kind != constants.GatewayAPIExtensionFilterDefinitionKind {
			errs = append(errs, field.Invalid(path.Child("kind"), definitionRef.Kind, fmt.Sprintf("kind must be %s", constants.GatewayAPIExtensionFilterDefinitionKind)))
		}
	}

	return errs
}

func (r *FilterWebhook) validateConfigRef(ctx context.Context, configRef *gwv1.LocalObjectReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	if configRef != nil && configRef.Group != extv1alpha1.GroupName {
		errs = append(errs, field.Invalid(path.Child("group"), configRef.Group, fmt.Sprintf("group must be %s", extv1alpha1.GroupName)))
	}

	return errs
}
