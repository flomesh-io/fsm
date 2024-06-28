package v1

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/flomesh-io/fsm/pkg/webhook"

	"github.com/flomesh-io/fsm/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	gwv1validation "github.com/flomesh-io/fsm/pkg/apis/gateway/v1/validation"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/version"
	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/types"

	"github.com/rs/zerolog/log"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"
)

type GatewayWebhook struct {
	webhook.DefaultWebhook
}

func NewGatewayWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &GatewayWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&gwv1.Gateway{}).
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

func (r *GatewayWebhook) Default(ctx context.Context, obj runtime.Object) error {
	gateway, ok := obj.(*gwv1.Gateway)
	if !ok {
		return fmt.Errorf("unexpected type: %T", obj)
	}

	log.Debug().Msgf("Default Webhook, name=%s", gateway.Name)
	log.Debug().Msgf("Before setting default values: %v", gateway)

	gatewayClass := &gwv1.GatewayClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		log.Error().Msgf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
		return err
	}

	if gatewayClass.Spec.ControllerName != constants.GatewayController {
		log.Warn().Msgf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, constants.GatewayController)
		return nil
	}

	// if it's a valid gateway, set default values
	if len(gateway.Labels) == 0 {
		gateway.Labels = map[string]string{}
	}
	gateway.Labels[constants.FSMAppNameLabelKey] = constants.FSMAppNameLabelValue
	gateway.Labels[constants.FSMAppInstanceLabelKey] = r.MeshName
	gateway.Labels[constants.FSMAppVersionLabelKey] = r.FSMVersion
	gateway.Labels[constants.AppLabel] = constants.FSMGatewayName

	return nil
}

func (r *GatewayWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *GatewayWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *GatewayWebhook) doValidation(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	gateway, ok := obj.(*gwv1.Gateway)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	gatewayClass := &gwv1.GatewayClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		return nil, fmt.Errorf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
	}

	if gatewayClass.Spec.ControllerName != constants.GatewayController {
		log.Warn().Msgf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, constants.GatewayController)
		return nil, nil
	}

	var errorList field.ErrorList
	if !version.IsCELValidationEnabled(r.KubeClient) {
		errorList = append(errorList, gwv1validation.ValidateGateway(gateway)...)
	}
	errorList = append(errorList, r.validateListenerPort(gateway)...)
	errorList = append(errorList, r.validateCertificateSecret(ctx, gateway)...)
	if r.Configurator.GetFeatureFlags().EnableValidateGatewayListenerHostname {
		errorList = append(errorList, r.validateListenerHostname(gateway)...)
	}
	if len(errorList) > 0 {
		return nil, utils.ErrorListToError(errorList)
	}

	return nil, nil
}

func (r *GatewayWebhook) validateCertificateSecret(ctx context.Context, gateway *gwv1.Gateway) field.ErrorList {
	var errs field.ErrorList

	// TODO: validate frontend CA
	for i, c := range gateway.Spec.Listeners {
		switch c.Protocol {
		case gwv1.HTTPSProtocolType:
			if c.TLS != nil && c.TLS.Mode != nil {
				switch *c.TLS.Mode {
				case gwv1.TLSModeTerminate:
					errs = append(errs, r.validateSecretsExistence(ctx, gateway, c, i)...)
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
					errs = append(errs, r.validateSecretsExistence(ctx, gateway, c, i)...)
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

func (r *GatewayWebhook) validateSecretsExistence(ctx context.Context, gateway *gwv1.Gateway, c gwv1.Listener, i int) field.ErrorList {
	var errs field.ErrorList

	for j, ref := range c.TLS.CertificateRefs {
		if string(*ref.Kind) == "Secret" && string(*ref.Group) == "" {
			ns := gwutils.NamespaceDerefOr(ref.Namespace, gateway.Namespace)
			name := string(ref.Name)

			path := field.NewPath("spec").
				Child("listeners").Index(i).
				Child("tls").
				Child("certificateRefs").Index(j)
			secret := &corev1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, secret); err != nil {
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

func (r *GatewayWebhook) validateListenerHostname(gateway *gwv1.Gateway) field.ErrorList {
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

func (r *GatewayWebhook) validateListenerPort(gateway *gwv1.Gateway) field.ErrorList {
	var errs field.ErrorList
	for i, listener := range gateway.Spec.Listeners {
		//if listener.Port > reservedPortRangeStart {
		if constants.ReservedGatewayPorts.Has(int32(listener.Port)) {
			path := field.NewPath("spec").
				Child("listeners").Index(i).
				Child("port")

			errs = append(errs, field.Invalid(path, listener.Port, fmt.Sprintf("port %d is reserved, please use other port instead", listener.Port)))
		}
	}
	return errs
}
