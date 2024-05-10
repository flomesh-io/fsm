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
	"net/http"

	"github.com/pkg/errors"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

type register struct {
	*webhook.RegisterConfig
	nsigClient nsigClientset.Interface
}

// NewRegister creates a new register for the namespacedingress resources
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
		nsigClient:     nsigClientset.NewForConfigOrDie(cfg.KubeConfig),
	}
}

// GetWebhooks returns the webhooks for the namespacedingress resources
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"networking.flomesh.io"},
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

// GetHandlers returns the handlers for the namespacedingress resources
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.NamespacedIngressMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Configurator, r.MeshName, r.FSMVersion)),
		constants.NamespacedIngressValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient, r.nsigClient)),
	}
}

type defaulter struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
	meshName   string
	fsmVersion string
}

func newDefaulter(kubeClient kubernetes.Interface, cfg configurator.Configurator, meshName, fsmVersion string) *defaulter {
	return &defaulter{
		kubeClient: kubeClient,
		cfg:        cfg,
		meshName:   meshName,
		fsmVersion: fsmVersion,
	}
}

// RuntimeObject returns the runtime object for the defaulter
func (w *defaulter) RuntimeObject() runtime.Object {
	return &nsigv1alpha1.NamespacedIngress{}
}

// SetDefaults sets the default values for the namespacedingress resource
func (w *defaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", c.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", c.Spec)

	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}
	if len(c.Labels) == 0 {
		c.Labels = map[string]string{}
	}
	c.Labels[constants.FSMAppNameLabelKey] = constants.FSMAppNameLabelValue
	c.Labels[constants.FSMAppInstanceLabelKey] = w.meshName
	c.Labels[constants.FSMAppVersionLabelKey] = w.fsmVersion
	c.Labels[constants.AppLabel] = constants.FSMIngressName

	if c.Spec.ServiceAccountName == "" {
		c.Spec.ServiceAccountName = "fsm-namespaced-ingress"
	}

	if c.Spec.LogLevel == nil {
		c.Spec.LogLevel = pointer.String("info")
	}

	if c.Spec.Replicas == nil {
		c.Spec.Replicas = pointer.Int32(1)
	}

	if c.Spec.HTTP.Port.Name == "" {
		c.Spec.HTTP.Port.Name = "http"
	}

	if c.Spec.HTTP.Port.Protocol == "" {
		c.Spec.HTTP.Port.Protocol = corev1.ProtocolTCP
	}

	if c.Spec.HTTP.Port.Port == 0 {
		c.Spec.HTTP.Port.Port = 80
	}

	if c.Spec.HTTP.Port.TargetPort == 0 {
		c.Spec.HTTP.Port.TargetPort = 8000
	}

	if c.Spec.TLS.Port.Name == "" {
		c.Spec.TLS.Port.Name = "https"
	}

	if c.Spec.TLS.Port.Protocol == "" {
		c.Spec.TLS.Port.Protocol = corev1.ProtocolTCP
	}

	if c.Spec.TLS.Port.Port == 0 {
		c.Spec.TLS.Port.Port = 443
	}

	if c.Spec.TLS.Port.TargetPort == 0 {
		c.Spec.TLS.Port.TargetPort = 8443
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

	log.Debug().Msgf("After setting default values, spec=%v", c.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
	nsigClient nsigClientset.Interface
}

// RuntimeObject returns the runtime object for the validator
func (w *validator) RuntimeObject() runtime.Object {
	return &nsigv1alpha1.NamespacedIngress{}
}

// ValidateCreate validates the creation of the namespacedingress resource
func (w *validator) ValidateCreate(obj interface{}) error {
	namespacedingress, ok := obj.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return nil
	}

	list, err := w.nsigClient.NetworkingV1alpha1().
		NamespacedIngresses(namespacedingress.Namespace).
		List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return err
	}

	// There's already an NamespacedIngress in this namespace, return error
	if len(list.Items) > 0 {
		return errors.Errorf(
			"there's already %d NamespacedIngress(s) in namespace %q. Each namespace can have ONLY ONE NamespacedIngress",
			len(list.Items),
			namespacedingress.Namespace,
		)
	}

	return doValidation(namespacedingress)
}

// ValidateUpdate validates the update of the namespacedingress resource
func (w *validator) ValidateUpdate(_, obj interface{}) error {
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

// ValidateDelete validates the deletion of the namespacedingress resource
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface, nsigClient nsigClientset.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
		nsigClient: nsigClient,
	}
}

func doValidation(_ interface{}) error {
	return nil
}
