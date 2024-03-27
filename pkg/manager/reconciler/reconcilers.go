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

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	clusterv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/cluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/controllers/flb"
	gatewayv1 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1"
	gatewayv1alpha2 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1alpha2"
	gatewayv1beta1 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1beta1"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/mcs/v1alpha1"
	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/namespacedingress/v1alpha1"
	pav1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/policyattachment/v1alpha1"
	svclb "github.com/flomesh-io/fsm/pkg/controllers/servicelb"
	"github.com/flomesh-io/fsm/pkg/version"
)

// RegisterReconcilers registers all reconcilers based on the configuration
func RegisterReconcilers(ctx context.Context) error {
	log.Info().Msgf("[MGR] Registering Reconcilers ...")

	cctx, err := fctx.ToControllerContext(ctx)
	if err != nil {
		return err
	}

	mc := cctx.Configurator

	reconcilers := make(map[string]controllers.Reconciler)

	//reconcilers["ProxyProfile"] = proxyprofilev1alpha1.NewReconciler(cctx)
	reconcilers["MCS(Cluster)"] = clusterv1alpha1.NewReconciler(cctx)
	reconcilers["MCS(ServiceExport)"] = mcsv1alpha1.NewServiceExportReconciler(cctx)

	//if mc.ShouldCreateServiceAndEndpointSlicesForMCS() && version.IsEndpointSliceEnabled(ctx.K8sAPI) {
	//	reconcilers["MCS(ServiceImport)"] = mcsv1alpha1.NewServiceImportReconciler(cctx)
	//	reconcilers["MCS(Service)"] = mcsv1alpha1.NewServiceReconciler(cctx)
	//	reconcilers["MCS(EndpointSlice)"] = mcsv1alpha1.NewEndpointSliceReconciler(cctx)
	//}

	if mc.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(cctx.KubeClient) {
		reconcilers["GatewayAPI(GatewayClass)"] = gatewayv1.NewGatewayClassReconciler(cctx)
		reconcilers["GatewayAPI(Gateway)"] = gatewayv1.NewGatewayReconciler(cctx)
		reconcilers["GatewayAPI(HTTPRoute)"] = gatewayv1.NewHTTPRouteReconciler(cctx)
		reconcilers["GatewayAPI(GRPCRoute)"] = gatewayv1alpha2.NewGRPCRouteReconciler(cctx)
		reconcilers["GatewayAPI(TCPRoute)"] = gatewayv1alpha2.NewTCPRouteReconciler(cctx)
		reconcilers["GatewayAPI(TLSRoute)"] = gatewayv1alpha2.NewTLSRouteReconciler(cctx)
		reconcilers["GatewayAPI(UDPRoute)"] = gatewayv1alpha2.NewUDPRouteReconciler(cctx)
		reconcilers["GatewayAPI(ReferenceGrant)"] = gatewayv1beta1.NewReferenceGrantReconciler(cctx)
		reconcilers["PolicyAttachment(RateLimit)"] = pav1alpha1.NewRateLimitPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(SessionSticky)"] = pav1alpha1.NewSessionStickyPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(LoadBalancer)"] = pav1alpha1.NewLoadBalancerPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(CircuitBreaking)"] = pav1alpha1.NewCircuitBreakingPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(AccessControl)"] = pav1alpha1.NewAccessControlPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(HealthCheck)"] = pav1alpha1.NewHealthCheckPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(FaultInjection)"] = pav1alpha1.NewFaultInjectionPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(UpstreamTLS)"] = pav1alpha1.NewUpstreamTLSPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(Retry)"] = pav1alpha1.NewRetryPolicyReconciler(cctx)
		reconcilers["PolicyAttachment(GatewayTLS)"] = pav1alpha1.NewGatewayTLSPolicyReconciler(cctx)
	}

	if mc.IsNamespacedIngressEnabled() {
		reconcilers["NamespacedIngress"] = nsigv1alpha1.NewReconciler(cctx)
	}

	if mc.IsServiceLBEnabled() {
		reconcilers["ServiceLB(Service)"] = svclb.NewServiceReconciler(cctx)
		reconcilers["ServiceLB(Node)"] = svclb.NewNodeReconciler(cctx)
	}

	if mc.IsFLBEnabled() {
		settingManager := flb.NewSettingManager(cctx)
		reconcilers["FLB(Service)"] = flb.NewServiceReconciler(cctx, settingManager)
		reconcilers["FLB(TLSSecret)"] = flb.NewSecretReconciler(cctx, settingManager)
	}

	for name, r := range reconcilers {
		if err := r.SetupWithManager(cctx.Manager); err != nil {
			log.Error().Msgf("Failed to setup reconciler %s: %s", name, err)
			return err
		}
	}

	return nil
}
