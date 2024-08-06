package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"

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

	_, errs := r.validateDuplicateFilterType(ctx, filter)

	scope := ptr.Deref(filter.Spec.Scope, extv1alpha1.FilterScopeRoute)
	switch scope {
	case extv1alpha1.FilterScopeListener:
		errs = append(errs, r.validateListenerFilter(ctx, filter)...)
	case extv1alpha1.FilterScopeRoute:
		errs = append(errs, r.validateRouteFilter(ctx, filter)...)
	}

	if len(errs) > 0 {
		return nil, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *FilterWebhook) validateDuplicateFilterType(ctx context.Context, filter *extv1alpha1.Filter) (admission.Warnings, field.ErrorList) {
	var errs field.ErrorList

	list := &extv1alpha1.FilterList{}
	if err := r.Manager.GetCache().List(ctx, list, client.InNamespace(filter.Namespace)); err != nil {
		return nil, nil
	}

	for _, f := range list.Items {
		if f.Name == filter.Name {
			continue
		}

		if f.Spec.Type == filter.Spec.Type {
			path := field.NewPath("spec").Child("type")
			errs = append(errs, field.Invalid(path, filter.Spec.Type, "filter type must be unique within the namespace"))
			break
		}
	}

	return nil, errs
}

func (r *FilterWebhook) validateListenerFilter(_ context.Context, filter *extv1alpha1.Filter) field.ErrorList {
	var errs field.ErrorList

	if len(filter.Spec.TargetRefs) == 0 {
		path := field.NewPath("spec").Child("targetRefs")
		errs = append(errs, field.Required(path, "targetRefs must be specified for filter with listener scope"))
	}

	return errs
}

func (r *FilterWebhook) validateRouteFilter(_ context.Context, filter *extv1alpha1.Filter) field.ErrorList {
	var errs field.ErrorList

	if len(filter.Spec.TargetRefs) > 0 {
		path := field.NewPath("spec").Child("targetRefs")
		errs = append(errs, field.Invalid(path, filter.Spec.TargetRefs, "targetRefs must not be specified for filter with route scope"))
	}

	return errs
}
