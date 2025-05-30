package v1alpha1

import (
	"context"
	"fmt"
	"net/netip"

	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type DNSModifierWebhook struct {
	webhook.DefaultWebhook
}

func NewDNSModifierWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &DNSModifierWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&extv1alpha1.DNSModifier{}).
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

func (r *DNSModifierWebhook) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*extv1alpha1.DNSModifier)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	return nil
}

func (r *DNSModifierWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *DNSModifierWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *DNSModifierWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	dnsModifier, ok := obj.(*extv1alpha1.DNSModifier)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	errs := r.validateSpec(ctx, dnsModifier.Spec, field.NewPath("spec"))

	if len(errs) > 0 {
		return warnings, utils.ErrorListToError(errs)
	}

	return nil, nil
}

func (r *DNSModifierWebhook) validateSpec(ctx context.Context, spec extv1alpha1.DNSModifierSpec, path *field.Path) field.ErrorList {
	var errs field.ErrorList

	for z, cfg := range spec.Zones {
		p := path.Child("zones").Key(z)

		if len(cfg.Domains) == 0 {
			errs = append(errs, field.Invalid(p.Child("domains"), cfg.Domains, "domains must not be empty"))
		}

		if len(cfg.Domains) > 0 {
			for i, domain := range cfg.Domains {
				p := p.Child("domains").Index(i)

				for j, answer := range domain.Answer {
					p := p.Child("answer").Index(j)

					rType := answer.RType
					if rType == nil {
						rType = ptr.To("A")
					}

					switch *rType {
					case "A":
						if ip, err := netip.ParseAddr(answer.RData); err != nil {
							errs = append(errs, field.Invalid(p.Child("rdata"), answer.RData, "invalid IP address"))
						} else {
							if !ip.Is4() {
								errs = append(errs, field.Invalid(p.Child("rdata"), answer.RData, "invalid IPv4 address for type A"))
							}
						}
					case "AAAA":
						if ip, err := netip.ParseAddr(answer.RData); err != nil {
							errs = append(errs, field.Invalid(p.Child("rdata"), answer.RData, "invalid IP address"))
						} else {
							if !ip.Is6() {
								errs = append(errs, field.Invalid(p.Child("rdata"), answer.RData, "invalid IPv6 address for type AAAA"))
							}
						}
					default:
						errs = append(errs, field.Invalid(p.Child("type"), rType, "invalid RType"))
					}
				}
			}
		}
	}

	return errs
}
