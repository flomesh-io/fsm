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

// Package tls contains webhook logic for the FLB TLS secret resource
package tls

import (
	"net/http"

	"github.com/flomesh-io/fsm/pkg/flb"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new FLB Secret webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the list of webhooks of the FLB Secret resource
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{""},
		[]string{"v1"},
		[]string{"secrets"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mflbtlssecret.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.FLBTLSSecretMutatingWebhookPath,
			r.CaBundle,
			nil,
			&metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.FLBTLSSecretLabel: "true",
				},
			},
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vflbtlssecret.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.FLBTLSSecretValidatingWebhookPath,
			r.CaBundle,
			nil,
			&metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.FLBTLSSecretLabel: "true",
				},
			},
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the list of handlers of the FLB Secret resource
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.FLBTLSSecretMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Configurator)),
		constants.FLBTLSSecretValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient, r.Configurator)),
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

// RuntimeObject returns the runtime object of the webhook
func (w *defaulter) RuntimeObject() runtime.Object {
	return &corev1.Secret{}
}

// SetDefaults sets the default values of the FLB Secret resource
func (w *defaulter) SetDefaults(obj interface{}) {
	//secret, ok := obj.(*corev1.Secret)
	//if !ok {
	//	return
	//}
}

type validator struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
}

// RuntimeObject returns the runtime object of the webhook
func (w *validator) RuntimeObject() runtime.Object {
	return &corev1.Secret{}
}

// ValidateCreate validates the creation of the FLB Secret resource
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of the FLB Secret resource
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the FLB Secret resource
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
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	if _, err := flb.IsValidTLSSecret(secret); err != nil {
		return err
	}

	return nil
}
