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

package secret

import (
	"fmt"
	"net/http"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/webhook"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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
			"mflbsecret.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.FLBSecretMutatingWebhookPath,
			r.CaBundle,
			nil,
			&metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.FlbSecretLabel: "true",
				},
			},
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vflbsecret.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.FLBSecretValidatingWebhookPath,
			r.CaBundle,
			nil,
			&metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.FlbSecretLabel: "true",
				},
			},
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the list of handlers of the FLB Secret resource
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.FLBSecretMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.FLBSecretValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient, r.Config)),
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
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}

	mc := w.cfg
	if secret.Name != mc.GetFLBSecretName() {
		return
	}

	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	if len(secret.Data[constants.FLBSecretKeyDefaultAlgo]) == 0 {
		secret.Data[constants.FLBSecretKeyDefaultAlgo] = []byte("rr")
	}
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
func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of the FLB Secret resource
func (w *validator) ValidateDelete(obj interface{}) error {
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

	mc := w.cfg
	if secret.Name != mc.GetFLBSecretName() {
		return nil
	}

	if mc.IsFLBStrictModeEnabled() {
		for _, key := range []string{
			constants.FLBSecretKeyBaseURL,
			constants.FLBSecretKeyUsername,
			constants.FLBSecretKeyPassword,
			constants.FLBSecretKeyDefaultCluster,
			constants.FLBSecretKeyDefaultAddressPool,
			constants.FLBSecretKeyDefaultAlgo,
		} {
			value, ok := secret.Data[key]
			if !ok {
				return fmt.Errorf("%q is required", key)
			}

			if len(value) == 0 {
				return fmt.Errorf("%q has an empty value", key)
			}
		}
	}

	return nil
}
