package flb

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/flb"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type ServiceWebhook struct {
	webhook.DefaultWebhook
}

func NewServiceWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &ServiceWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&corev1.Service{}).
		WithWebhookServiceName(cfg.WebhookSvcName).
		WithWebhookServiceNamespace(cfg.WebhookSvcNs).
		WithCABundle(cfg.CaBundle).
		WithCategory("flb").
		Complete(); err != nil {
		return nil
	} else {
		r.CfgBuilder = blder
	}

	return r
}

func (r *ServiceWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *ServiceWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *ServiceWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	if !flb.IsFLBEnabled(service, r.KubeClient) {
		return nil, nil
	}

	mc := r.Configurator
	if mc.IsFLBStrictModeEnabled() {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: service.Namespace, Name: mc.GetFLBSecretName()}, secret); err != nil {
			return nil, err
		}
	}

	if flb.IsTLSEnabled(service) {
		if _, err := flb.IsValidTLSPort(service); err != nil {
			return nil, err
		}

		if _, err := flb.IsServiceRefToValidTLSSecret(service, r.KubeClient); err != nil {
			return nil, err
		}
	}

	return nil, nil
}
