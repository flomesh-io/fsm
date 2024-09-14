package v1alpha1

import (
	"context"
	"fmt"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type FilterDefinitionWebhook struct {
	webhook.DefaultWebhook
}

func NewFilterDefinitionWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &FilterDefinitionWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.FilterDefinition{}).
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

func (r *FilterDefinitionWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.FilterDefinition)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *FilterDefinitionWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *FilterDefinitionWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *FilterDefinitionWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	FilterDefinition, ok := obj.(*extv1alpha1.FilterDefinition)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	warnings, errs := r.validateDuplicateFilterDefinitionType(ctx, FilterDefinition)

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *FilterDefinitionWebhook) validateDuplicateFilterDefinitionType(ctx context.Context, FilterDefinition *extv1alpha1.FilterDefinition) (admission.Warnings, field.ErrorList) {
	var errs field.ErrorList

	list := &extv1alpha1.FilterDefinitionList{}
	if err := r.Manager.GetCache().List(ctx, list); err != nil {
		return nil, nil
	}

	for _, f := range list.Items {
		if f.Name == FilterDefinition.Name {
			continue
		}

		if f.Spec.Type == FilterDefinition.Spec.Type {
			path := field.NewPath("spec").Child("type")
			errs = append(errs, field.Invalid(path, FilterDefinition.Spec.Type, "FilterDefinition type must be unique within the cluster"))
			break
		}
	}

	return nil, errs
}
