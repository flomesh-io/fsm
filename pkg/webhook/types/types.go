package types

import (
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	fctx "github.com/flomesh-io/fsm/pkg/context"
)

// RegisterConfig is the configuration for webhook registers
type RegisterConfig struct {
	*fctx.ControllerContext
	WebhookSvcNs   string
	WebhookSvcName string
	CaBundle       []byte
}

// WebhookConfigurationProvider interface for webhook configurations
type WebhookConfigurationProvider interface {
	GetWebhookConfigurations() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook)
}

type CategoryProvider interface {
	GetCategory() string
}

// Register is the interface for webhook registers
type Register interface {
	admission.CustomDefaulter
	admission.CustomValidator
	WebhookConfigurationProvider
	CategoryProvider
}
