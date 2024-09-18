package v1alpha1

import (
	"context"
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type FilterConfigWebhook struct {
	webhook.DefaultWebhook
}

func NewFilterConfigWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &FilterConfigWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.FilterConfig{}).
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

func (r *FilterConfigWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.FilterConfig)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *FilterConfigWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *FilterConfigWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *FilterConfigWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	filterConfig, ok := obj.(*extv1alpha1.FilterConfig)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	warnings, errs := r.validateSpec(ctx, filterConfig.Spec, field.NewPath("spec"))

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *FilterConfigWebhook) validateSpec(ctx context.Context, spec extv1alpha1.FilterConfigSpec, path *field.Path) (admission.Warnings, field.ErrorList) {
	var errs field.ErrorList

	m := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(spec.Config), &m); err != nil {
		errs = append(errs, field.Invalid(path.Child("config"), spec.Config, fmt.Sprintf("config is not in valid YAML format: %s", err)))
	}

	return nil, errs
}
