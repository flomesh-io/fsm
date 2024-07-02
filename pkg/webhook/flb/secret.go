package flb

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type SecretWebhook struct {
	webhook.DefaultWebhook
}

func NewSecretWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &SecretWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&corev1.Secret{}).
		WithWebhookServiceName(cfg.WebhookSvcName).
		WithWebhookServiceNamespace(cfg.WebhookSvcNs).
		WithCABundle(cfg.CaBundle).
		WithCategory("flb").
		WithObjectSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.FLBConfigSecretLabel: "true",
			},
		}).
		Complete(); err != nil {
		return nil
	} else {
		r.CfgBuilder = blder
	}

	return r
}

func (r *SecretWebhook) Default(_ context.Context, obj runtime.Object) error {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	mc := r.Configurator
	if secret.Name != mc.GetFLBSecretName() {
		return nil
	}

	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	if len(secret.Data[constants.FLBSecretKeyDefaultAlgo]) == 0 {
		secret.Data[constants.FLBSecretKeyDefaultAlgo] = []byte("rr")
	}

	return nil
}

func (r *SecretWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *SecretWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *SecretWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	mc := r.Configurator
	if secret.Name != mc.GetFLBSecretName() {
		return nil, nil
	}

	if mc.IsFLBStrictModeEnabled() {
		for _, key := range []string{
			constants.FLBSecretKeyBaseURL,
			constants.FLBSecretKeyUsername,
			constants.FLBSecretKeyPassword,
			constants.FLBSecretKeyDefaultAddressPool,
			constants.FLBSecretKeyDefaultAlgo,
		} {
			value, ok := secret.Data[key]
			if !ok {
				return nil, fmt.Errorf("%q is required", key)
			}

			if len(value) == 0 {
				return nil, fmt.Errorf("%q has an empty value", key)
			}
		}
	}

	return nil, nil
}
