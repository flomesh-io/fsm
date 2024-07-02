package v1alpha2

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	gwv1alpha2validation "github.com/flomesh-io/fsm/pkg/apis/gateway/v1alpha2/validation"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/version"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type TLSRouteWebhook struct {
	webhook.DefaultWebhook
}

func NewTLSRouteWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &TLSRouteWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1alpha2.TLSRoute{}).
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

func (r *TLSRouteWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *TLSRouteWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *TLSRouteWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	route, ok := obj.(*gwv1alpha2.TLSRoute)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	var errorList field.ErrorList
	if !version.IsCELValidationEnabled(r.KubeClient) {
		errorList = append(errorList, gwv1alpha2validation.ValidateTLSRoute(route)...)
	}
	errorList = append(errorList, webhook.ValidateParentRefs(route.Spec.ParentRefs)...)
	if r.Configurator.GetFeatureFlags().EnableValidateTLSRouteHostnames {
		errorList = append(errorList, webhook.ValidateRouteHostnames(route.Spec.Hostnames)...)
	}
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}
