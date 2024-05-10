/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package grpcroute

import (
	"net/http"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a GRPCRoute webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for GRPCRoute
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.GatewayAPIGroup},
		[]string{"v1alpha2"},
		[]string{"grpcroutes"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mgrpcroute.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GRPCRouteMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vgrpcroute.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GRPCRouteValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for GRPCRoute
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.GRPCRouteMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Configurator)),
		constants.GRPCRouteValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient, r.Configurator)),
	}
}

type defaulter struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
}

func newDefaulter(kubeClient kubernetes.Interface, cfg configurator.Configurator) *defaulter {
	return &defaulter{
		kubeClient: kubeClient,
		cfg:        cfg,
	}
}

// RuntimeObject returns the runtime object for the webhook
func (w *defaulter) RuntimeObject() runtime.Object {
	return &gwv1.GRPCRoute{}
}

// SetDefaults sets the default values for the GRPCRoute
func (w *defaulter) SetDefaults(obj interface{}) {
	route, ok := obj.(*gwv1.GRPCRoute)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", route.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", route.Spec)

	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}

	log.Debug().Msgf("After setting default values, spec=%v", route.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &gwv1.GRPCRoute{}
}

// ValidateCreate validates the creation of the GRPCRoute
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of the GRPCRoute
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the GRPCRoute
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface, cfg configurator.Configurator) *validator {
	return &validator{
		kubeClient: kubeClient,
		cfg:        cfg,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	route, ok := obj.(*gwv1.GRPCRoute)
	if !ok {
		return nil
	}

	//errorList := gwv1alpha2validation.ValidateGRPCRoute(route)
	var errorList field.ErrorList
	errorList = append(errorList, webhook.ValidateParentRefs(route.Spec.ParentRefs)...)
	if w.cfg.GetFeatureFlags().EnableValidateGRPCRouteHostnames {
		errorList = append(errorList, webhook.ValidateRouteHostnames(route.Spec.Hostnames)...)
	}
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	return nil
}
