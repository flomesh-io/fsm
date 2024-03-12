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

package webhook

import (
	"context"
	"fmt"
	"os"

	"github.com/flomesh-io/fsm/pkg/webhook/referencegrant"

	"github.com/flomesh-io/fsm/pkg/webhook/udproute"

	"github.com/flomesh-io/fsm/pkg/webhook/retry"

	"github.com/flomesh-io/fsm/pkg/webhook/upstreamtls"

	"github.com/flomesh-io/fsm/pkg/webhook/faultinjection"

	"github.com/flomesh-io/fsm/pkg/webhook/healthcheck"

	"github.com/flomesh-io/fsm/pkg/webhook/accesscontrol"

	"github.com/flomesh-io/fsm/pkg/version"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/webhook"
	"github.com/flomesh-io/fsm/pkg/webhook/circuitbreaking"
	"github.com/flomesh-io/fsm/pkg/webhook/cluster"
	flbsecret "github.com/flomesh-io/fsm/pkg/webhook/flb/secret"
	flbservice "github.com/flomesh-io/fsm/pkg/webhook/flb/service"
	flbtls "github.com/flomesh-io/fsm/pkg/webhook/flb/tls"
	"github.com/flomesh-io/fsm/pkg/webhook/gateway"
	"github.com/flomesh-io/fsm/pkg/webhook/gatewayclass"
	"github.com/flomesh-io/fsm/pkg/webhook/globaltrafficpolicy"
	"github.com/flomesh-io/fsm/pkg/webhook/grpcroute"
	"github.com/flomesh-io/fsm/pkg/webhook/httproute"
	"github.com/flomesh-io/fsm/pkg/webhook/ingress"
	"github.com/flomesh-io/fsm/pkg/webhook/loadbalancer"
	"github.com/flomesh-io/fsm/pkg/webhook/namespacedingress"
	"github.com/flomesh-io/fsm/pkg/webhook/ratelimit"
	"github.com/flomesh-io/fsm/pkg/webhook/serviceexport"
	"github.com/flomesh-io/fsm/pkg/webhook/serviceimport"
	"github.com/flomesh-io/fsm/pkg/webhook/sessionsticky"
	"github.com/flomesh-io/fsm/pkg/webhook/tcproute"
	"github.com/flomesh-io/fsm/pkg/webhook/tlsroute"
)

// RegisterWebHooks registers all webhooks based on the configuration
func RegisterWebHooks(ctx *fctx.ControllerContext) error {
	log.Info().Msgf("[MGR] Registering Webhooks ...")

	registers, err := webhookRegisters(ctx)

	if err != nil {
		return err
	}

	if err := createWebhookConfigurations(ctx, registers); err != nil {
		return err
	}

	registerWebhookHandlers(ctx, registers)

	return nil
}

func webhookRegisters(ctx *fctx.ControllerContext) ([]webhook.Register, error) {
	mc := ctx.Config

	cert, err := issueCertForWebhook(ctx.CertificateManager, mc)
	if err != nil {
		return nil, err
	}

	cfg := registerConfig(ctx, mc, cert)

	return getRegisters(cfg, mc), nil
}

func createWebhookConfigurations(ctx *fctx.ControllerContext, registers []webhook.Register) error {
	mutatingWebhooks, validatingWebhooks := allWebhooks(registers)

	// Mutating
	if mwc := flomeshadmission.NewMutatingWebhookConfiguration(mutatingWebhooks, ctx.MeshName, ctx.FSMVersion); mwc != nil {
		mutating := ctx.KubeClient.
			AdmissionregistrationV1().
			MutatingWebhookConfigurations()
		if _, err := mutating.Create(context.Background(), mwc, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				existingMwc, err := mutating.Get(context.Background(), mwc.Name, metav1.GetOptions{})
				if err != nil {
					log.Error().Msgf("Unable to get MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
					return err
				}

				existingMwc.Webhooks = mwc.Webhooks
				_, err = mutating.Update(context.Background(), existingMwc, metav1.UpdateOptions{})
				if err != nil {
					// Should be not conflict for a leader-election manager, error is error
					log.Error().Msgf("Unable to update MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
					return err
				}
			} else {
				log.Error().Msgf("Unable to create MutatingWebhookConfigurations %q, %s", mwc.Name, err.Error())
				return err
			}
		}
	}

	// Validating
	if vwc := flomeshadmission.NewValidatingWebhookConfiguration(validatingWebhooks, ctx.MeshName, ctx.FSMVersion); vwc != nil {
		validating := ctx.KubeClient.
			AdmissionregistrationV1().
			ValidatingWebhookConfigurations()
		if _, err := validating.Create(context.Background(), vwc, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				existingVmc, err := validating.Get(context.Background(), vwc.Name, metav1.GetOptions{})
				if err != nil {
					log.Error().Msgf("Unable to get ValidatingWebhookConfigurations %q, %s", vwc.Name, err.Error())
					return err
				}

				existingVmc.Webhooks = vwc.Webhooks
				_, err = validating.Update(context.Background(), existingVmc, metav1.UpdateOptions{})
				if err != nil {
					log.Error().Msgf("Unable to update ValidatingWebhookConfigurations %q, %s", vwc.Name, err.Error())
					return err
				}
			} else {
				log.Error().Msgf("Unable to create ValidatingWebhookConfigurations %q, %s", vwc.Name, err.Error())
				return err
			}
		}
	}

	return nil
}

func issueCertForWebhook(certMgr *certificate.Manager, mc configurator.Configurator) (*certificate.Certificate, error) {
	//cert, err := certMgr.IssueCertificate(
	//	mc.Webhook.ServiceName,
	//	constants.DefaultCAValidityPeriod,
	//	[]string{
	//		mc.Webhook.ServiceName,
	//		fmt.Sprintf("%s.%s.svc", mc.Webhook.ServiceName, mc.GetFSMNamespace()),
	//		fmt.Sprintf("%s.%s.svc.cluster.local", mc.Webhook.ServiceName, mc.GetFSMNamespace()),
	//	},
	//)

	cert, err := certMgr.IssueCertificate(
		fmt.Sprintf("%s.%s.svc", constants.FSMControllerName, mc.GetFSMNamespace()),
		certificate.Internal,
		certificate.FullCNProvided())
	if err != nil {
		return nil, err
	}
	if err != nil {
		log.Error().Msgf("Error issuing certificate: %s ", err)
		return nil, err
	}

	// write ca.crt, tls.crt & tls.key to file
	servingCertsPath := constants.WebhookServerServingCertsPath
	if err := os.MkdirAll(servingCertsPath, 0750); err != nil {
		log.Error().Msgf("error creating dir %q, %s", servingCertsPath, err.Error())
		return nil, err
	}

	certFiles := map[string][]byte{
		constants.RootCACertName:    cert.GetIssuingCA(),
		constants.TLSCertName:       cert.GetCertificateChain(),
		constants.TLSPrivateKeyName: cert.GetPrivateKey(),
	}

	for file, data := range certFiles {
		fileName := fmt.Sprintf("%s/%s", servingCertsPath, file)
		if err := os.WriteFile(
			fileName,
			data,
			0600); err != nil {
			log.Error().Msgf("error writing file %q, %s", fileName, err.Error())
			return nil, err
		}
	}

	return cert, nil
}

func allWebhooks(registers []webhook.Register) (mutating []admissionregv1.MutatingWebhook, validating []admissionregv1.ValidatingWebhook) {
	for _, r := range registers {
		m, v := r.GetWebhooks()

		if len(m) > 0 {
			mutating = append(mutating, m...)
		}

		if len(v) > 0 {
			validating = append(validating, v...)
		}
	}

	return mutating, validating
}

func registerWebhookHandlers(ctx *fctx.ControllerContext, registers []webhook.Register) {
	hookServer := ctx.Manager.GetWebhookServer()

	for _, r := range registers {
		for path, handler := range r.GetHandlers() {
			hookServer.Register(path, handler)
		}
	}
}

func getRegisters(regCfg *webhook.RegisterConfig, mc configurator.Configurator) []webhook.Register {
	result := make([]webhook.Register, 0)

	//result = append(result, injector.NewRegister(regCfg))

	result = append(result, cluster.NewRegister(regCfg))
	//result = append(result, cm.NewRegister(regCfg))
	//result = append(result, proxyprofile.NewRegister(regCfg))
	result = append(result, serviceexport.NewRegister(regCfg))
	result = append(result, serviceimport.NewRegister(regCfg))
	result = append(result, globaltrafficpolicy.NewRegister(regCfg))

	if mc.IsIngressEnabled() {
		result = append(result, ingress.NewRegister(regCfg))
		if mc.IsNamespacedIngressEnabled() {
			result = append(result, namespacedingress.NewRegister(regCfg))
		}
	}

	if mc.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(regCfg.KubeClient) {
		result = append(result, gateway.NewRegister(regCfg))
		result = append(result, gatewayclass.NewRegister(regCfg))
		result = append(result, httproute.NewRegister(regCfg))
		result = append(result, grpcroute.NewRegister(regCfg))
		result = append(result, tcproute.NewRegister(regCfg))
		result = append(result, tlsroute.NewRegister(regCfg))
		result = append(result, udproute.NewRegister(regCfg))
		result = append(result, referencegrant.NewRegister(regCfg))
		result = append(result, ratelimit.NewRegister(regCfg))
		result = append(result, sessionsticky.NewRegister(regCfg))
		result = append(result, loadbalancer.NewRegister(regCfg))
		result = append(result, circuitbreaking.NewRegister(regCfg))
		result = append(result, accesscontrol.NewRegister(regCfg))
		result = append(result, healthcheck.NewRegister(regCfg))
		result = append(result, faultinjection.NewRegister(regCfg))
		result = append(result, upstreamtls.NewRegister(regCfg))
		result = append(result, retry.NewRegister(regCfg))
	}

	if mc.IsFLBEnabled() {
		result = append(result, flbsecret.NewRegister(regCfg))
		result = append(result, flbservice.NewRegister(regCfg))
		result = append(result, flbtls.NewRegister(regCfg))
	}

	return result
}

func registerConfig(ctx *fctx.ControllerContext, mc configurator.Configurator, cert *certificate.Certificate) *webhook.RegisterConfig {
	return &webhook.RegisterConfig{
		ControllerContext: ctx,
		WebhookSvcNs:      mc.GetFSMNamespace(),
		WebhookSvcName:    constants.FSMControllerName,
		CaBundle:          cert.GetIssuingCA(),
	}
}
