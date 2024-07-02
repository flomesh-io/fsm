package ingress

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/utils"
	"github.com/flomesh-io/fsm/pkg/utils"

	"github.com/flomesh-io/fsm/pkg/webhook"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type K8sIngressWebhook struct {
	webhook.DefaultWebhook
}

func NewK8sIngressWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &K8sIngressWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&networkingv1.Ingress{}).
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

func (r *K8sIngressWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *K8sIngressWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *K8sIngressWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	if !ingresspipy.IsValidPipyIngress(ing) {
		return nil, nil
	}

	upstreamSSLSecret := ing.Annotations[constants.PipyIngressAnnotationUpstreamSSLSecret]
	if upstreamSSLSecret != "" {
		if err := r.secretExists(ctx, upstreamSSLSecret, ing); err != nil {
			return nil, fmt.Errorf("secert %q doesn't exist: %s, please check annotation 'pipy.ingress.kubernetes.io/upstream-ssl-secret' of Ingress %s/%s", upstreamSSLSecret, err, ing.Namespace, ing.Name)
		}
	}

	trustedCASecret := ing.Annotations[constants.PipyIngressAnnotationTLSTrustedCASecret]
	if trustedCASecret != "" {
		if err := r.secretExists(ctx, trustedCASecret, ing); err != nil {
			return nil, fmt.Errorf("secert %q doesn't exist: %s, please check annotation 'pipy.ingress.kubernetes.io/tls-trusted-ca-secret' of Ingress %s/%s", trustedCASecret, err, ing.Namespace, ing.Name)
		}
	}

	for _, tls := range ing.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		if err := r.secretExists(ctx, tls.SecretName, ing); err != nil {
			return nil, fmt.Errorf("TLS secret %q of Ingress %s/%s doesn't exist, please check spec.tls section of Ingress", tls.SecretName, ing.Namespace, ing.Name)
		}
	}

	return nil, nil
}

func (r *K8sIngressWebhook) secretExists(ctx context.Context, secretName string, ing *networkingv1.Ingress) error {
	ns, name, err := utils.SecretNamespaceAndName(secretName, ing)
	if err != nil {
		return err
	}

	if name == "" {
		return fmt.Errorf("secret name of Ingress %s/%s is empty or invalid", ing.Namespace, ing.Name)
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, secret); err != nil {
		return err
	}

	return nil
}
