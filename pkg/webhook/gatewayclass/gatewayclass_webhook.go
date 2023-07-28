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

package gatewayclass

import (
	"fmt"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/webhook"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"net/http"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gwv1beta1validation "sigs.k8s.io/gateway-api/apis/v1beta1/validation"
)

type register struct {
	*webhook.RegisterConfig
}

func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"gateway.networking.k8s.io"},
		[]string{"v1beta1"},
		[]string{"gatewayclasses"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mgatewayclass.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GatewayClassMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vgatewayclass.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GatewayClassValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.GatewayClassMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.GatewayClassValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
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

func (w *defaulter) RuntimeObject() runtime.Object {
	return &gwv1beta1.GatewayClass{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		return
	}

	log.Info().Msgf("Default Webhook, name=%s", gatewayClass.Name)
	log.Info().Msgf("Before setting default values, spec=%v", gatewayClass.Spec)

	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}

	log.Info().Msgf("After setting default values, spec=%v", gatewayClass.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
}

func (w *validator) RuntimeObject() runtime.Object {
	return &gwv1beta1.GatewayClass{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	oldGatewayClass, ok := oldObj.(*gwv1beta1.GatewayClass)
	if !ok {
		return nil
	}

	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		return nil
	}

	if oldGatewayClass.Spec.ControllerName != gatewayClass.Spec.ControllerName {
		if err := doValidation(obj); err != nil {
			return err
		}
	}

	errorList := gwv1beta1validation.ValidateGatewayClassUpdate(oldGatewayClass, gatewayClass)
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	return nil
}

func (w *validator) ValidateDelete(obj interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		log.Warn().Msgf("unexpected object type: %T", obj)
		return nil
	}

	if gatewayClass.Spec.ControllerName != constants.GatewayController {
		return fmt.Errorf("unknown gateway controller: %s, ONLY %s is supported", gatewayClass.Spec.ControllerName, constants.GatewayController)
	}

	return nil
}
