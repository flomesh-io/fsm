package ingress

import (
	"github.com/flomesh-io/fsm/pkg/webhook"
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type ServiceExportWebhook struct {
	webhook.DefaultWebhook
}

func NewServiceExportWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &ServiceExportWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&mcsv1alpha1.ServiceExport{}).
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
