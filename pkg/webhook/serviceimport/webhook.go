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

package serviceimport

import (
	"net/http"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new ServiceImport webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks to be registered for ServiceImport
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"serviceimports"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mserviceimport.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.ServiceImportMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vserviceimport.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.ServiceImportValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered for ServiceImport
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.ServiceImportMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.ServiceImportValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
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
	return &mcsv1alpha1.ServiceImport{}
}

// SetDefaults sets the default values for the webhook
func (w *defaulter) SetDefaults(obj interface{}) {
	serviceImport, ok := obj.(*mcsv1alpha1.ServiceImport)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", serviceImport.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", serviceImport.Spec)

	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}

	if serviceImport.Spec.Type == "" {
		// ONLY set the value, there's no any logic to handle the type yet
		serviceImport.Spec.Type = mcsv1alpha1.ClusterSetIP
	}

	log.Debug().Msgf("After setting default values, spec=%v", serviceImport.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object for the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &mcsv1alpha1.ServiceImport{}
}

// ValidateCreate validates the create request for the webhook
func (w *validator) ValidateCreate(obj interface{}) error {
	return doValidation(obj)
}

// ValidateUpdate validates the update request for the webhook
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return doValidation(obj)
}

// ValidateDelete validates the delete request for the webhook
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(_ interface{}) error {
	//serviceImport, ok := obj.(*mcsv1alpha1.ServiceImport)
	//if !ok {
	//    return nil
	//}

	return nil
}
