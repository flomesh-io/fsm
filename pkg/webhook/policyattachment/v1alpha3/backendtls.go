package v1alpha3

import (
	"github.com/flomesh-io/fsm/pkg/webhook"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type BackendTLSPolicyWebhook struct {
	webhook.DefaultWebhook
}

func NewBackendTLSPolicyWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &BackendTLSPolicyWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1alpha3.BackendTLSPolicy{}).
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
