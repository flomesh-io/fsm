package ingress

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/constants"

	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/webhook"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type NamespacedIngressWebhook struct {
	webhook.DefaultWebhook
}

func NewNamespacedIngressWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &NamespacedIngressWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&nsigv1alpha1.NamespacedIngress{}).
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

func (r *NamespacedIngressWebhook) Default(_ context.Context, obj runtime.Object) error {
	c, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	if len(c.Labels) == 0 {
		c.Labels = map[string]string{}
	}
	c.Labels[constants.FSMAppNameLabelKey] = constants.FSMAppNameLabelValue
	c.Labels[constants.FSMAppInstanceLabelKey] = r.MeshName
	c.Labels[constants.FSMAppVersionLabelKey] = r.FSMVersion
	c.Labels[constants.AppLabel] = constants.FSMIngressName

	if c.Spec.ServiceAccountName == "" {
		c.Spec.ServiceAccountName = "fsm-namespaced-ingress"
	}

	if c.Spec.LogLevel == nil {
		c.Spec.LogLevel = pointer.String("info")
	}

	if c.Spec.Replicas == nil {
		c.Spec.Replicas = pointer.Int32(1)
	}

	if c.Spec.HTTP.Port.Name == "" {
		c.Spec.HTTP.Port.Name = "http"
	}

	if c.Spec.HTTP.Port.Protocol == "" {
		c.Spec.HTTP.Port.Protocol = corev1.ProtocolTCP
	}

	if c.Spec.HTTP.Port.Port == 0 {
		c.Spec.HTTP.Port.Port = 80
	}

	if c.Spec.HTTP.Port.TargetPort == 0 {
		c.Spec.HTTP.Port.TargetPort = 8000
	}

	if c.Spec.TLS.Port.Name == "" {
		c.Spec.TLS.Port.Name = "https"
	}

	if c.Spec.TLS.Port.Protocol == "" {
		c.Spec.TLS.Port.Protocol = corev1.ProtocolTCP
	}

	if c.Spec.TLS.Port.Port == 0 {
		c.Spec.TLS.Port.Port = 443
	}

	if c.Spec.TLS.Port.TargetPort == 0 {
		c.Spec.TLS.Port.TargetPort = 8443
	}

	if c.Spec.TLS.SSLPassthrough.UpstreamPort == nil {
		c.Spec.TLS.SSLPassthrough.UpstreamPort = pointer.Int32(443)
	}

	if c.Spec.PodSecurityContext == nil {
		c.Spec.PodSecurityContext = &corev1.PodSecurityContext{
			RunAsNonRoot: pointer.Bool(true),
			RunAsUser:    pointer.Int64(65532),
			RunAsGroup:   pointer.Int64(65532),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	}

	if c.Spec.SecurityContext == nil {
		c.Spec.SecurityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: pointer.Bool(false),
		}
	}

	return nil
}

func (r *NamespacedIngressWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	namespacedingress, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	list := &nsigv1alpha1.NamespacedIngressList{}
	if err := r.Client.List(ctx, list, client.InNamespace(namespacedingress.Namespace)); err != nil {
		return nil, err
	}

	// There's already an NamespacedIngress in this namespace, return error
	if len(list.Items) > 0 {
		return nil, errors.Errorf(
			"there's already %d NamespacedIngress(s) in namespace %q. Each namespace can have ONLY ONE NamespacedIngress",
			len(list.Items),
			namespacedingress.Namespace,
		)
	}

	return nil, nil
}
