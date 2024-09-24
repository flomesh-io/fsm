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

package reconciler

import (
	"context"
	"fmt"
	"os"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/namespacedingress/v1alpha1"

	svclb "github.com/flomesh-io/fsm/pkg/controllers/servicelb"

	ctv1 "github.com/flomesh-io/fsm/pkg/controllers/connector/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/controllers/flb"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/mcs/v1alpha1"

	pav1alpha3 "github.com/flomesh-io/fsm/pkg/controllers/policyattachment/v1alpha3"

	pav1alpha2 "github.com/flomesh-io/fsm/pkg/controllers/policyattachment/v1alpha2"

	gatewayv1alpha2 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1alpha2"
	gatewayv1beta1 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1beta1"

	clusterv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/cluster/v1alpha1"
	extensionv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/extension/v1alpha1"
	gatewayv1 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1"

	"github.com/flomesh-io/fsm/pkg/controllers"

	"github.com/flomesh-io/fsm/pkg/webhook"

	"github.com/flomesh-io/fsm/pkg/version"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	extwhv1alpha1 "github.com/flomesh-io/fsm/pkg/webhook/extension/v1alpha1"
	flbwh "github.com/flomesh-io/fsm/pkg/webhook/flb"
	gwwhv1 "github.com/flomesh-io/fsm/pkg/webhook/gatewayapi/v1"
	gwwhv1alpha2 "github.com/flomesh-io/fsm/pkg/webhook/gatewayapi/v1alpha2"
	ingresswh "github.com/flomesh-io/fsm/pkg/webhook/ingress"
	mcswhv1alpha1 "github.com/flomesh-io/fsm/pkg/webhook/mcs/v1alpha1"
	pawhv1alpha2 "github.com/flomesh-io/fsm/pkg/webhook/policyattachment/v1alpha2"
	pawhv1alpha3 "github.com/flomesh-io/fsm/pkg/webhook/policyattachment/v1alpha3"
)

// RegisterWebhooksAndReconcilers registers all webhooks based on the configuration
func RegisterWebhooksAndReconcilers(ctx context.Context) error {
	log.Info().Msgf("[MGR] Registering Webhooks ...")

	cctx, err := fctx.ToControllerContext(ctx)
	if err != nil {
		return err
	}

	whs, recons, err := registers(cctx)

	if err != nil {
		return err
	}

	if err := createWebhookConfigurations(cctx, whs); err != nil {
		return err
	}

	if err := registerReconcilers(cctx, recons); err != nil {
		return err
	}

	return nil
}

func registers(ctx *fctx.ControllerContext) (map[ResourceType]whtypes.Register, map[ResourceType]controllers.Reconciler, error) {
	mc := ctx.Configurator

	cert, err := issueCertForWebhook(ctx.CertManager, mc)
	if err != nil {
		return nil, nil, err
	}

	cfg := registerConfig(ctx, mc, cert)

	whs, recons := getRegisters(cfg, mc)

	return whs, recons, nil
}

func createWebhookConfigurations(ctx *fctx.ControllerContext, registers map[ResourceType]whtypes.Register) error {
	mutatingWebhooks, validatingWebhooks := allWebhooks(registers)

	// Mutating
	if mwc := webhook.NewMutatingWebhookConfiguration(mutatingWebhooks, ctx.MeshName, ctx.FSMVersion); mwc != nil {
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
	if vwc := webhook.NewValidatingWebhookConfiguration(validatingWebhooks, ctx.MeshName, ctx.FSMVersion); vwc != nil {
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
	cert, err := certMgr.IssueCertificate(
		fmt.Sprintf("%s.%s.svc", constants.FSMControllerName, mc.GetFSMNamespace()),
		certificate.Internal,
		certificate.FullCNProvided())
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

func allWebhooks(registers map[ResourceType]whtypes.Register) (mutating []admissionregv1.MutatingWebhook, validating []admissionregv1.ValidatingWebhook) {
	for _, r := range registers {
		m, v := r.GetWebhookConfigurations()

		if len(m) > 0 {
			mutating = append(mutating, m...)
		}

		if len(v) > 0 {
			validating = append(validating, v...)
		}
	}

	return mutating, validating
}

func getRegisters(regCfg *whtypes.RegisterConfig, mc configurator.Configurator) (map[ResourceType]whtypes.Register, map[ResourceType]controllers.Reconciler) {
	ctx := regCfg.ControllerContext
	webhooks := map[ResourceType]whtypes.Register{}
	reconcilers := map[ResourceType]controllers.Reconciler{}

	webhooks[MCSCluster] = mcswhv1alpha1.NewClusterWebhook(regCfg)
	reconcilers[MCSCluster] = clusterv1alpha1.NewReconciler(ctx, webhooks[MCSCluster])

	webhooks[MCSServiceExport] = mcswhv1alpha1.NewServiceExportWebhook(regCfg)
	reconcilers[MCSServiceExport] = mcsv1alpha1.NewServiceExportReconciler(ctx, webhooks[MCSServiceExport])

	// no reconcilers
	webhooks[MCSServiceImport] = mcswhv1alpha1.NewServiceImportWebhook(regCfg)
	webhooks[MCSGlobalTrafficPolicy] = mcswhv1alpha1.NewGlobalTrafficPolicyWebhook(regCfg)

	// Connectors, no webhooks
	reconcilers[ConnectorConsulConnector] = ctv1.NewConsulConnectorReconciler(ctx)
	reconcilers[ConnectorEurekaConnector] = ctv1.NewEurekaConnectorReconciler(ctx)
	reconcilers[ConnectorNacosConnector] = ctv1.NewNacosConnectorReconciler(ctx)
	reconcilers[ConnectorMachineConnector] = ctv1.NewMachineConnectorReconciler(ctx)
	reconcilers[ConnectorGatewayConnector] = ctv1.NewGatewayConnectorReconciler(ctx)

	if mc.IsIngressEnabled() {
		// no reconciler
		webhooks[K8sIngress] = ingresswh.NewK8sIngressWebhook(regCfg)
	}

	if mc.IsNamespacedIngressEnabled() {
		webhooks[NamespacedIngress] = ingresswh.NewNamespacedIngressWebhook(regCfg)
		reconcilers[NamespacedIngress] = nsigv1alpha1.NewReconciler(ctx, webhooks[NamespacedIngress])
	}

	if mc.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(regCfg.KubeClient) {
		webhooks[GatewayAPIGateway] = gwwhv1.NewGatewayWebhook(regCfg)
		reconcilers[GatewayAPIGateway] = gatewayv1.NewGatewayReconciler(ctx, webhooks[GatewayAPIGateway])

		webhooks[GatewayAPIGatewayClass] = gwwhv1.NewGatewayClassWebhook(regCfg)
		reconcilers[GatewayAPIGatewayClass] = gatewayv1.NewGatewayClassReconciler(ctx, webhooks[GatewayAPIGatewayClass])

		webhooks[GatewayAPIHTTPRoute] = gwwhv1.NewHTTPRouteWebhook(regCfg)
		reconcilers[GatewayAPIHTTPRoute] = gatewayv1.NewHTTPRouteReconciler(ctx, webhooks[GatewayAPIHTTPRoute])

		webhooks[GatewayAPIGRPCRoute] = gwwhv1.NewGRPCRouteWebhook(regCfg)
		reconcilers[GatewayAPIGRPCRoute] = gatewayv1.NewGRPCRouteReconciler(ctx, webhooks[GatewayAPIGRPCRoute])

		webhooks[GatewayAPITCPRoute] = gwwhv1alpha2.NewTCPRouteWebhook(regCfg)
		reconcilers[GatewayAPITCPRoute] = gatewayv1alpha2.NewTCPRouteReconciler(ctx, webhooks[GatewayAPITCPRoute])

		webhooks[GatewayAPIUDPRoute] = gwwhv1alpha2.NewUDPRouteWebhook(regCfg)
		reconcilers[GatewayAPIUDPRoute] = gatewayv1alpha2.NewUDPRouteReconciler(ctx, webhooks[GatewayAPIUDPRoute])

		webhooks[GatewayAPITLSRoute] = gwwhv1alpha2.NewTLSRouteWebhook(regCfg)
		reconcilers[GatewayAPITLSRoute] = gatewayv1alpha2.NewTLSRouteReconciler(ctx, webhooks[GatewayAPITLSRoute])

		reconcilers[GatewayAPIReferenceGrant] = gatewayv1beta1.NewReferenceGrantReconciler(ctx)

		webhooks[PolicyAttachmentHealthCheck] = pawhv1alpha2.NewHealthCheckPolicyWebhook(regCfg)
		reconcilers[PolicyAttachmentHealthCheck] = pav1alpha2.NewHealthCheckPolicyReconciler(ctx, webhooks[PolicyAttachmentHealthCheck])

		webhooks[PolicyAttachmentRetry] = pawhv1alpha2.NewRetryPolicyWebhook(regCfg)
		reconcilers[PolicyAttachmentRetry] = pav1alpha2.NewRetryPolicyReconciler(ctx, webhooks[PolicyAttachmentRetry])

		webhooks[PolicyAttachmentBackendLB] = pawhv1alpha2.NewBackendLBPolicyWebhook(regCfg)
		reconcilers[PolicyAttachmentBackendLB] = pav1alpha2.NewBackendLBPolicyReconciler(ctx, webhooks[PolicyAttachmentBackendLB])

		webhooks[PolicyAttachmentBackendTLS] = pawhv1alpha3.NewBackendTLSPolicyWebhook(regCfg)
		reconcilers[PolicyAttachmentBackendTLS] = pav1alpha3.NewBackendTLSPolicyReconciler(ctx, webhooks[PolicyAttachmentBackendTLS])

		webhooks[GatewayAPIExtensionFilter] = extwhv1alpha1.NewFilterWebhook(regCfg)
		reconcilers[GatewayAPIExtensionFilter] = extensionv1alpha1.NewFilterReconciler(ctx, webhooks[GatewayAPIExtensionFilter])

		webhooks[GatewayAPIExtensionListenerFilter] = extwhv1alpha1.NewListenerFilterWebhook(regCfg)
		reconcilers[GatewayAPIExtensionListenerFilter] = extensionv1alpha1.NewListenerFilterReconciler(ctx, webhooks[GatewayAPIExtensionListenerFilter])

		webhooks[GatewayAPIExtensionFilterDefinition] = extwhv1alpha1.NewFilterDefinitionWebhook(regCfg)
		reconcilers[GatewayAPIExtensionFilterDefinition] = extensionv1alpha1.NewFilterDefinitionReconciler(ctx, webhooks[GatewayAPIExtensionFilterDefinition])

		webhooks[GatewayAPIExtensionFilterConfig] = extwhv1alpha1.NewFilterConfigWebhook(regCfg)
		reconcilers[GatewayAPIExtensionFilterConfig] = extensionv1alpha1.NewFilterConfigReconciler(ctx, webhooks[GatewayAPIExtensionFilterConfig])

		reconcilers[GatewayAPIExtensionCircuitBreaker] = extensionv1alpha1.NewCircuitBreakerReconciler(ctx)

		reconcilers[GatewayAPIExtensionRateLimit] = extensionv1alpha1.NewRateLimitReconciler(ctx)

		reconcilers[GatewayAPIExtensionHTTPLog] = extensionv1alpha1.NewHTTPLogReconciler(ctx)

		reconcilers[GatewayAPIExtensionMetrics] = extensionv1alpha1.NewMetricsReconciler(ctx)

		reconcilers[GatewayAPIExtensionZipkin] = extensionv1alpha1.NewZipkinReconciler(ctx)

		webhooks[GatewayAPIExtensionFaultInjection] = extwhv1alpha1.NewFaultInjectionWebhook(regCfg)
		reconcilers[GatewayAPIExtensionFaultInjection] = extensionv1alpha1.NewFaultInjectionReconciler(ctx, webhooks[GatewayAPIExtensionFaultInjection])
	}

	if mc.IsServiceLBEnabled() {
		// no webhooks
		reconcilers[ServiceLBService] = svclb.NewServiceReconciler(ctx)
		reconcilers[ServiceLBNode] = svclb.NewNodeReconciler(ctx)
	}

	if mc.IsFLBEnabled() {
		settingManager := flb.NewSettingManager(ctx)

		webhooks[FLBService] = flbwh.NewServiceWebhook(regCfg)
		reconcilers[FLBService] = flb.NewServiceReconciler(ctx, webhooks[FLBService], settingManager)

		// no reconciler
		webhooks[FLBSecret] = flbwh.NewSecretWebhook(regCfg)

		webhooks[FLBTLSSecret] = flbwh.NewTLSSecretWebhook(regCfg)
		reconcilers[FLBTLSSecret] = flb.NewSecretReconciler(ctx, webhooks[FLBTLSSecret], settingManager)
	}

	return webhooks, reconcilers
}

func registerConfig(ctx *fctx.ControllerContext, mc configurator.Configurator, cert *certificate.Certificate) *whtypes.RegisterConfig {
	return &whtypes.RegisterConfig{
		ControllerContext: ctx,
		WebhookSvcNs:      mc.GetFSMNamespace(),
		WebhookSvcName:    constants.FSMControllerName,
		CaBundle:          cert.GetIssuingCA(),
	}
}

// registerReconcilers registers all reconcilers based on the configuration
func registerReconcilers(ctx context.Context, reconcilers map[ResourceType]controllers.Reconciler) error {
	log.Info().Msgf("[MGR] Registering Reconcilers ...")

	cctx, err := fctx.ToControllerContext(ctx)
	if err != nil {
		return err
	}

	//reconcilers := make(map[ResourceType]controllers.Reconciler)
	//for t := range getResourceTypeFunctionMapping(cctx) {
	//	if r := newReconciler(t, cctx, registers); r != nil {
	//		reconcilers[t] = r
	//	}
	//}

	for t, r := range reconcilers {
		if err := r.SetupWithManager(cctx.Manager); err != nil {
			log.Error().Msgf("Failed to setup reconciler %s: %s", t, err)
			return err
		}
	}

	return nil
}
