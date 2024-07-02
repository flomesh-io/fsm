package ingress

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/flomesh-io/fsm/pkg/webhook"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceImportWebhook struct {
	webhook.DefaultWebhook
}

func NewServiceImportWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &ServiceImportWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&mcsv1alpha1.ServiceImport{}).
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

func (r *ServiceImportWebhook) Default(_ context.Context, obj runtime.Object) error {
	serviceImport, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	if serviceImport.Spec.Type == "" {
		// ONLY set the value, there's no any logic to handle the type yet
		serviceImport.Spec.Type = mcsv1alpha1.ClusterSetIP
	}

	return nil
}
