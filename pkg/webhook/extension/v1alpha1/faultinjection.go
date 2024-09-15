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

type FaultInjectionWebhook struct {
	webhook.DefaultWebhook
}

func NewFaultInjectionWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &FaultInjectionWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.FaultInjection{}).
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

func (r *FaultInjectionWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.FaultInjection)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *FaultInjectionWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *FaultInjectionWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *FaultInjectionWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	faultInjection, ok := obj.(*extv1alpha1.FaultInjection)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errs := r.validateSpec(ctx, faultInjection.Spec, field.NewPath("spec"))

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *FaultInjectionWebhook) validateSpec(ctx context.Context, spec extv1alpha1.FaultInjectionSpec, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	if spec.Delay == nil && spec.Abort == nil {
		errs = append(errs, field.Invalid(path, spec, "either delay or abort must be set"))
	}

	if spec.Delay != nil && spec.Abort != nil {
		errs = append(errs, field.Invalid(path, spec, "only one of delay or abort can be set"))
	}

	if spec.Delay != nil && spec.Delay.Min != nil && spec.Delay.Max != nil {
		if spec.Delay.Min.Nanoseconds() > spec.Delay.Max.Nanoseconds() {
			errs = append(errs, field.Invalid(path.Child("delay.min"), spec.Delay.Min, "min delay must be less than or equals max delay"))
		}
	}

	return errs
}
