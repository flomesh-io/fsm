package webhook

import (
	"context"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type DefaultWebhook struct {
	whtypes.Register
	*whtypes.RegisterConfig
	client.Client
	CfgBuilder *builder.WebhookConfigurationBuilder
}

func (r *DefaultWebhook) GetCategory() string {
	if r.CfgBuilder == nil {
		return ""
	}

	return r.CfgBuilder.GetCategory()
}

func (r *DefaultWebhook) GetWebhookConfigurations() (mwebhooks []admissionregv1.MutatingWebhook, vwebhooks []admissionregv1.ValidatingWebhook) {
	if r.CfgBuilder == nil {
		return
	}

	if mwh := r.CfgBuilder.MutatingWebhook(); mwh != nil {
		mwebhooks = append(mwebhooks, *mwh)
	}

	if vwh := r.CfgBuilder.ValidatingWebhook(); vwh != nil {
		vwebhooks = append(vwebhooks, *vwh)
	}

	return
}

func (r *DefaultWebhook) Default(_ context.Context, _ runtime.Object) error {
	return nil
}

func (r *DefaultWebhook) ValidateCreate(_ context.Context, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (r *DefaultWebhook) ValidateUpdate(_ context.Context, _, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (r *DefaultWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}
