package v1alpha1

import (
	"context"
	"fmt"

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

	var errs field.ErrorList

	list := &extv1alpha1.FilterList{}
	if err := r.Manager.GetCache().List(ctx, list, client.InNamespace(filter.Namespace)); err != nil {
		return nil, err
	}

	for _, f := range list.Items {
		if f.Name == filter.Name {
			continue
		}

		if f.Spec.Name == filter.Spec.Name {
			path := field.NewPath("spec").Child("name")
			errs = append(errs, field.Invalid(path, filter.Spec.Name, "filter name must be unique within the namespace"))
			break
		}
	}

	if len(errs) > 0 {
		return nil, utils.ErrorListToError(errs)
	}

	return nil, nil
}
