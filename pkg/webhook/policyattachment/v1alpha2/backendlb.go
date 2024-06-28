package v1alpha2

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/flomesh-io/fsm/pkg/webhook"
)

type BackendLBPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewBackendLBPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &BackendLBPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1alpha2.BackendLBPolicy{}).
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
