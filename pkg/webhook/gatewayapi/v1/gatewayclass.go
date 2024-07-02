package v1

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	gwv1validation "github.com/flomesh-io/fsm/pkg/apis/gateway/v1/validation"
	"github.com/flomesh-io/fsm/pkg/utils"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type GatewayClassWebhook struct {
	webhook.DefaultWebhook
}

func NewGatewayClassWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &GatewayClassWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1.GatewayClass{}).
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

func (r *GatewayClassWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	oldGatewayClass, ok := oldObj.(*gwv1.GatewayClass)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", oldObj)
	}

	gatewayClass, ok := newObj.(*gwv1.GatewayClass)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", newObj)
	}

	if errorList := gwv1validation.ValidateGatewayClassUpdate(oldGatewayClass, gatewayClass); len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}
