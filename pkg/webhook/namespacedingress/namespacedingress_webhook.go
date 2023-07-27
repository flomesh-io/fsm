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

package namespacedingress

import (
	"context"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/pkg/errors"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
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
		[]string{"flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"namespacedingresses"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mnamespacedingress.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.NamespacedIngressMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vnamespacedingress.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.NamespacedIngressValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.NamespacedIngressMutatingWebhookPath:   webhook.DefaultingWebhookFor(newDefaulter(r.KubeClient, r.Config)),
		constants.NamespacedIngressValidatingWebhookPath: webhook.ValidatingWebhookFor(newValidator(r.KubeClient, r.NsigClient)),
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
	return &nsigv1alpha1.NamespacedIngress{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return
	}

	log.Info().Msgf("Default Webhook, name=%s", c.Name)
	log.Info().Msgf("Before setting default values, spec=%v", c.Spec)

	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}

	if c.Spec.ServiceAccountName == "" {
		c.Spec.ServiceAccountName = "fsm-namespaced-ingress"
	}

	if c.Spec.LogLevel == nil {
		c.Spec.LogLevel = pointer.String("info")
	}

	if c.Spec.Replicas == nil {
		c.Spec.Replicas = pointer.Int32(1)
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

	log.Info().Msgf("After setting default values, spec=%v", c.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
	nsigClient nsigClientset.Interface
}

func (w *validator) RuntimeObject() runtime.Object {
	return &nsigv1alpha1.NamespacedIngress{}
}

func (w *validator) ValidateCreate(obj interface{}) error {
	namespacedingress, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return nil
	}

	list, err := w.nsigClient.FlomeshV1alpha1().
		NamespacedIngresses(namespacedingress.Namespace).
		List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return err
	}

	// There's already an NamespacedIngress in this namespace, return error
	if len(list.Items) > 0 {
		return errors.Errorf(
			"There's already %d NamespacedIngress(s) in namespace %q. Each namespace can have ONLY ONE NamespacedIngress.",
			len(list.Items),
			namespacedingress.Namespace,
		)
	}

	return doValidation(namespacedingress)
}

func (w *validator) ValidateUpdate(oldObj, obj interface{}) error {
	//oldNamespacedIngress, ok := oldObj.(*nsigv1alpha1.NamespacedIngress)
	//if !ok {
	//	return nil
	//}
	//
	//namespacedingress, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	//if !ok {
	//	return nil
	//}
	//
	//if oldNamespacedIngress.Namespace != namespacedingress.Namespace {
	//    return errors.Errorf("")
	//}

	return doValidation(obj)
}

func (w *validator) ValidateDelete(obj interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface, nsigClient nsigClientset.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
		nsigClient: nsigClient,
	}
}

func doValidation(obj interface{}) error {
	return nil
}
