package v1

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"k8s.io/apimachinery/pkg/util/validation/field"

	gwv1validation "github.com/flomesh-io/fsm/pkg/apis/gateway/v1/validation"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/version"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"
)

type HTTPRouteWebhook struct {
	webhook.DefaultWebhook
}

func NewHTTPRouteWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &HTTPRouteWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1.HTTPRoute{}).
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

func (r *HTTPRouteWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *HTTPRouteWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *HTTPRouteWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	route, ok := obj.(*gwv1.HTTPRoute)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	var errorList field.ErrorList
	if !version.IsCELValidationEnabled(r.KubeClient) {
		errorList = append(errorList, gwv1validation.ValidateHTTPRoute(route)...)
	}
	errorList = append(errorList, webhook.ValidateParentRefs(route.Spec.ParentRefs)...)
	if r.Configurator.GetFeatureFlags().EnableValidateHTTPRouteHostnames {
		errorList = append(errorList, webhook.ValidateRouteHostnames(route.Spec.Hostnames)...)
	}
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}
