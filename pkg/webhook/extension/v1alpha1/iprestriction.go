package v1alpha1

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type IPRestrictionWebhook struct {
	webhook.DefaultWebhook
}

func NewIPRestrictionWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &IPRestrictionWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.IPRestriction{}).
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

func (r *IPRestrictionWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.IPRestriction)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *IPRestrictionWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *IPRestrictionWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *IPRestrictionWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	ipRestriction, ok := obj.(*extv1alpha1.IPRestriction)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errs := r.validateSpec(ctx, ipRestriction.Spec, field.NewPath("spec"))

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *IPRestrictionWebhook) validateSpec(ctx context.Context, spec extv1alpha1.IPRestrictionSpec, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	if len(spec.Allowed) == 0 && len(spec.Forbidden) == 0 {
		errs = append(errs, field.Invalid(path, spec, "either allowed or forbidden must be set"))
	}

	for i, ip := range spec.Allowed {
		if _, err := netip.ParseAddr(ip); err == nil {
			continue
		}

		if _, _, err := net.ParseCIDR(ip); err == nil {
			continue
		}

		errs = append(errs, field.Invalid(path.Child("allowed").Index(i), ip, "invalid IP address or CIDR"))
	}

	for i, ip := range spec.Forbidden {
		if _, err := netip.ParseAddr(ip); err == nil {
			continue
		}

		if _, _, err := net.ParseCIDR(ip); err == nil {
			continue
		}

		errs = append(errs, field.Invalid(path.Child("forbidden").Index(i), ip, "invalid IP address or CIDR"))
	}

	return errs
}
