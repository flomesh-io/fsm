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

package gateway

import (
	"context"
	"fmt"
	gwv1validation "github.com/flomesh-io/fsm/pkg/apis/gateway/v1/validation"
	"github.com/flomesh-io/fsm/pkg/version"
	"net/http"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

type register struct {
	*webhook.RegisterConfig
	gatewayAPIClient gatewayApiClientset.Interface
}

const (
	reservedPortRangeStart = 60000
)

// NewRegister creates a new gateway webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig:   cfg,
		gatewayAPIClient: gatewayApiClientset.NewForConfigOrDie(cfg.KubeConfig),
	}
}

// GetWebhooks returns the webhooks to be registered of gateway
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{constants.GatewayAPIGroup},
		[]string{"v1"},
		[]string{"gateways"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mgateway.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GatewayMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vgateway.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.GatewayValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Ignore,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the handlers to be registered of gateway
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.GatewayMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.gatewayAPIClient, r.Configurator, r.MeshName, r.FSMVersion)),
		constants.GatewayValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient, r.Configurator)),
	}
}

type defaulter struct {
	kubeClient       kubernetes.Interface
	gatewayAPIClient gatewayApiClientset.Interface
	cfg              configurator.Configurator
	meshName         string
	fsmVersion       string
}

func newDefaulter(kubeClient kubernetes.Interface, gatewayAPIClient gatewayApiClientset.Interface, cfg configurator.Configurator, meshName, fsmVersion string) *defaulter {
	return &defaulter{
		kubeClient:       kubeClient,
		gatewayAPIClient: gatewayAPIClient,
		cfg:              cfg,
		meshName:         meshName,
		fsmVersion:       fsmVersion,
	}
}

// RuntimeObject returns the runtime object of gateway
func (w *defaulter) RuntimeObject() runtime.Object {
	return &gwv1.Gateway{}
}

// SetDefaults sets the default values of gateway
func (w *defaulter) SetDefaults(obj interface{}) {
	gateway, ok := obj.(*gwv1.Gateway)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", gateway.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", gateway.Spec)

	gatewayClass, err := w.gatewayAPIClient.
		GatewayV1().
		GatewayClasses().
		Get(context.TODO(), string(gateway.Spec.GatewayClassName), metav1.GetOptions{})
	if err != nil {
		log.Error().Msgf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
		return
	}

	if gatewayClass.Spec.ControllerName != constants.GatewayController {
		log.Warn().Msgf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, constants.GatewayController)
		return
	}

	// if it's a valid gateway, set default values
	if len(gateway.Labels) == 0 {
		gateway.Labels = map[string]string{}
	}
	gateway.Labels[constants.FSMAppNameLabelKey] = constants.FSMAppNameLabelValue
	gateway.Labels[constants.FSMAppInstanceLabelKey] = w.meshName
	gateway.Labels[constants.FSMAppVersionLabelKey] = w.fsmVersion
	gateway.Labels[constants.AppLabel] = constants.FSMGatewayName

	log.Debug().Msgf("After setting default values, spec=%v", gateway.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
}

// RuntimeObject returns the runtime object of gateway
func (w *validator) RuntimeObject() runtime.Object {
	return &gwv1.Gateway{}
}

// ValidateCreate validates the creation of gateway
func (w *validator) ValidateCreate(obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateUpdate validates the update of gateway
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	return w.doValidation(obj)
}

// ValidateDelete validates the deletion of gateway
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
	gateway, ok := obj.(*gwv1.Gateway)
	if !ok {
		return nil
	}

	var errorList field.ErrorList
	if !version.IsCELValidationEnabled(w.kubeClient) {
		errorList = append(errorList, gwv1validation.ValidateGateway(gateway)...)
	}
	errorList = append(errorList, w.validateListenerPort(gateway)...)
	errorList = append(errorList, w.validateCertificateSecret(gateway)...)
	if w.cfg.GetFeatureFlags().EnableValidateGatewayListenerHostname {
		errorList = append(errorList, w.validateListenerHostname(gateway)...)
	}
	if len(errorList) > 0 {
		return utils.ErrorListToError(errorList)
	}

	return nil
}

func (w *validator) validateCertificateSecret(gateway *gwv1.Gateway) field.ErrorList {
	var errs field.ErrorList

	for i, c := range gateway.Spec.Listeners {
		switch c.Protocol {
		case gwv1.HTTPSProtocolType:
			if c.TLS != nil && c.TLS.Mode != nil {
				switch *c.TLS.Mode {
				case gwv1.TLSModeTerminate:
					errs = append(errs, w.validateSecretsExistence(gateway, c, i)...)
				case gwv1.TLSModePassthrough:
					path := field.NewPath("spec").
						Child("listeners").Index(i).
						Child("tls").
						Child("mode")
					errs = append(errs, field.Forbidden(path, fmt.Sprintf("TLSModeType %s is not supported when Protocol is %s, please use Protocol %s", gwv1.TLSModePassthrough, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType)))
				}
			}
		case gwv1.TLSProtocolType:
			if c.TLS != nil && c.TLS.Mode != nil {
				switch *c.TLS.Mode {
				case gwv1.TLSModeTerminate:
					errs = append(errs, w.validateSecretsExistence(gateway, c, i)...)
				case gwv1.TLSModePassthrough:
					if len(c.TLS.CertificateRefs) > 0 {
						path := field.NewPath("spec").
							Child("listeners").Index(i).
							Child("tls").
							Child("certificateRefs")
						errs = append(errs, field.Forbidden(path, fmt.Sprintf("No need to provide certificates when Protocol is %s and TLSModeType is %s", gwv1.TLSProtocolType, gwv1.TLSModePassthrough)))
					}
				}
			}
		}
	}

	return errs
}

func (w *validator) validateSecretsExistence(gateway *gwv1.Gateway, c gwv1.Listener, i int) field.ErrorList {
	var errs field.ErrorList

	for j, ref := range c.TLS.CertificateRefs {
		if string(*ref.Kind) == "Secret" && string(*ref.Group) == "" {
			ns := gwutils.Namespace(ref.Namespace, gateway.Namespace)
			name := string(ref.Name)

			path := field.NewPath("spec").
				Child("listeners").Index(i).
				Child("tls").
				Child("certificateRefs").Index(j)
			secret, err := w.kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				errs = append(errs, field.NotFound(path, fmt.Sprintf("Failed to get Secret %s/%s: %s", ns, name, err)))
				continue
			}

			v, ok := secret.Data[corev1.TLSCertKey]
			if ok {
				if string(v) == "" {
					errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s is empty", ns, name, corev1.TLSCertKey)))
				}
			} else {
				errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s doesn't have required data by key %s", ns, name, corev1.TLSCertKey)))
			}

			v, ok = secret.Data[corev1.TLSPrivateKeyKey]
			if ok {
				if string(v) == "" {
					errs = append(errs, field.Invalid(path, string(v), fmt.Sprintf("The content of Secret %s/%s by key %s is empty", ns, name, corev1.TLSPrivateKeyKey)))
				}
			} else {
				errs = append(errs, field.NotFound(path, fmt.Sprintf("Secret %s/%s doesn't have required data by key %s", ns, name, corev1.TLSPrivateKeyKey)))
			}
		}
	}

	return errs
}

func (w *validator) validateListenerHostname(gateway *gwv1.Gateway) field.ErrorList {
	var errs field.ErrorList

	for i, listener := range gateway.Spec.Listeners {
		if listener.Hostname != nil {
			hostname := string(*listener.Hostname)
			if err := webhook.IsValidHostname(hostname); err != nil {
				path := field.NewPath("spec").
					Child("listeners").Index(i).
					Child("hostname")

				errs = append(errs, field.Invalid(path, hostname, fmt.Sprintf("%s", err)))
			}
		}
	}

	return errs
}

func (w *validator) validateListenerPort(gateway *gwv1.Gateway) field.ErrorList {
	var errs field.ErrorList
	for i, listener := range gateway.Spec.Listeners {
		if listener.Port > reservedPortRangeStart {
			path := field.NewPath("spec").
				Child("listeners").Index(i).
				Child("port")

			errs = append(errs, field.Invalid(path, listener.Port, fmt.Sprintf("port must be less than or equals %d", reservedPortRangeStart)))
		}
	}
	return errs
}
