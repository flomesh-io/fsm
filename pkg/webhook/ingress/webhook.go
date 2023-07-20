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

package ingress

import (
	"context"
	"fmt"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/constants"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/webhook"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"net/http"
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
		[]string{"networking.k8s.io"},
		[]string{"v1"},
		[]string{"ingresses"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mingress.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.IngressMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vingress.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.IngressValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.IngressMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient)),
		constants.IngressValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient)),
	}
}

type defaulter struct {
	kubeClient kubernetes.Interface
}

func newDefaulter(kubeClient kubernetes.Interface) *defaulter {
	return &defaulter{
		kubeClient: kubeClient,
	}
}

func (w *defaulter) RuntimeObject() runtime.Object {
	return &networkingv1.Ingress{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return
	}

	if !ingresspipy.IsValidPipyIngress(ing) {
		return
	}

}

type validator struct {
	kubeClient kubernetes.Interface
}

func (w *validator) RuntimeObject() runtime.Object {
	return &networkingv1.Ingress{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	return w.doValidation(obj)
}

func (w *validator) ValidateDelete(obj interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func (w *validator) doValidation(obj interface{}) error {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}

	if !ingresspipy.IsValidPipyIngress(ing) {
		return nil
	}

	upstreamSSLSecret := ing.Annotations[ingresspipy.PipyIngressAnnotationUpstreamSSLSecret]
	if upstreamSSLSecret != "" {
		if err := w.secretExists(upstreamSSLSecret, ing); err != nil {
			return fmt.Errorf("secert %q doesn't exist: %s, please check annotation 'pipy.ingress.kubernetes.io/upstream-ssl-secret' of Ingress %s/%s", upstreamSSLSecret, err, ing.Namespace, ing.Name)
		}
	}

	trustedCASecret := ing.Annotations[ingresspipy.PipyIngressAnnotationTLSTrustedCASecret]
	if trustedCASecret != "" {
		if err := w.secretExists(trustedCASecret, ing); err != nil {
			return fmt.Errorf("secert %q doesn't exist: %s, please check annotation 'pipy.ingress.kubernetes.io/tls-trusted-ca-secret' of Ingress %s/%s", trustedCASecret, err, ing.Namespace, ing.Name)
		}
	}

	for _, tls := range ing.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		if err := w.secretExists(tls.SecretName, ing); err != nil {
			return fmt.Errorf("TLS secret %q of Ingress %s/%s doesn't exist, please check spec.tls section of Ingress", tls.SecretName, ing.Namespace, ing.Name)
		}
	}

	return nil
}

func (w *validator) secretExists(secretName string, ing *networkingv1.Ingress) error {
	ns, name, err := utils.SecretNamespaceAndName(secretName, ing)
	if err != nil {
		return err
	}

	if name == "" {
		return fmt.Errorf("secret name of Ingress %s/%s is empty or invalid", ing.Namespace, ing.Name)
	}

	if _, err := w.kubeClient.CoreV1().
		Secrets(ns).
		Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
		return err
	}

	return nil
}
