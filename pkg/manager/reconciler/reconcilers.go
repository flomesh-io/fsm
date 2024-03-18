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
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	clusterv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/cluster/v1alpha1"
	ctv1 "github.com/flomesh-io/fsm/pkg/controllers/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/controllers/flb"
	gatewayv1alpha2 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1alpha2"
	gatewayv1beta1 "github.com/flomesh-io/fsm/pkg/controllers/gateway/v1beta1"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/mcs/v1alpha1"
	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/namespacedingress/v1alpha1"
	pav1alpha1 "github.com/flomesh-io/fsm/pkg/controllers/policyattachment/v1alpha1"
	svclb "github.com/flomesh-io/fsm/pkg/controllers/servicelb"
	"github.com/flomesh-io/fsm/pkg/version"
)

// RegisterReconcilers registers all reconcilers based on the configuration
func RegisterReconcilers(ctx *fctx.ControllerContext) error {
	log.Info().Msgf("[MGR] Registering Reconcilers ...")

	mc := ctx.Config

	reconcilers := make(map[string]controllers.Reconciler)

	//reconcilers["ProxyProfile"] = proxyprofilev1alpha1.NewReconciler(ctx)
	reconcilers["MCS(Cluster)"] = clusterv1alpha1.NewReconciler(ctx)
	reconcilers["MCS(ServiceExport)"] = mcsv1alpha1.NewServiceExportReconciler(ctx)

	//if mc.ShouldCreateServiceAndEndpointSlicesForMCS() && version.IsEndpointSliceEnabled(ctx.K8sAPI) {
	//	reconcilers["MCS(ServiceImport)"] = mcsv1alpha1.NewServiceImportReconciler(ctx)
	//	reconcilers["MCS(Service)"] = mcsv1alpha1.NewServiceReconciler(ctx)
	//	reconcilers["MCS(EndpointSlice)"] = mcsv1alpha1.NewEndpointSliceReconciler(ctx)
	//}

	if mc.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(ctx.KubeClient) {
		reconcilers["GatewayAPI(GatewayClass)"] = gatewayv1beta1.NewGatewayClassReconciler(ctx)
		reconcilers["GatewayAPI(Gateway)"] = gatewayv1beta1.NewGatewayReconciler(ctx)
		reconcilers["GatewayAPI(HTTPRoute)"] = gatewayv1beta1.NewHTTPRouteReconciler(ctx)
		reconcilers["GatewayAPI(GRPCRoute)"] = gatewayv1alpha2.NewGRPCRouteReconciler(ctx)
		reconcilers["GatewayAPI(TCPRoute)"] = gatewayv1alpha2.NewTCPRouteReconciler(ctx)
		reconcilers["GatewayAPI(TLSRoute)"] = gatewayv1alpha2.NewTLSRouteReconciler(ctx)
		reconcilers["GatewayAPI(UDPRoute)"] = gatewayv1alpha2.NewUDPRouteReconciler(ctx)
		reconcilers["PolicyAttachment(RateLimit)"] = pav1alpha1.NewRateLimitPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(SessionSticky)"] = pav1alpha1.NewSessionStickyPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(LoadBalancer)"] = pav1alpha1.NewLoadBalancerPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(CircuitBreaking)"] = pav1alpha1.NewCircuitBreakingPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(AccessControl)"] = pav1alpha1.NewAccessControlPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(HealthCheck)"] = pav1alpha1.NewHealthCheckPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(FaultInjection)"] = pav1alpha1.NewFaultInjectionPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(UpstreamTLS)"] = pav1alpha1.NewUpstreamTLSPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(Retry)"] = pav1alpha1.NewRetryPolicyReconciler(ctx)
		reconcilers["PolicyAttachment(GatewayTLS)"] = pav1alpha1.NewGatewayTLSPolicyReconciler(ctx)
	}

	reconcilers["Connector(ConsulConnector)"] = ctv1.NewConsulConnectorReconciler(ctx)
	reconcilers["Connector(EurekaConnector)"] = ctv1.NewEurekaConnectorReconciler(ctx)
	reconcilers["Connector(NacosConnector)"] = ctv1.NewNacosConnectorReconciler(ctx)
	reconcilers["Connector(MachineConnector)"] = ctv1.NewMachineConnectorReconciler(ctx)
	reconcilers["Connector(GatewayConnector)"] = ctv1.NewGatewayConnectorReconciler(ctx)

	if mc.IsNamespacedIngressEnabled() {
		reconcilers["NamespacedIngress"] = nsigv1alpha1.NewReconciler(ctx)
	}

	if mc.IsServiceLBEnabled() {
		reconcilers["ServiceLB(Service)"] = svclb.NewServiceReconciler(ctx)
		reconcilers["ServiceLB(Node)"] = svclb.NewNodeReconciler(ctx)
	}

	if mc.IsFLBEnabled() {
		reconcilers["FLB"] = flb.NewReconciler(ctx)
	}

	for name, r := range reconcilers {
		if err := r.SetupWithManager(ctx.Manager); err != nil {
			log.Error().Msgf("Failed to setup reconciler %s: %s", name, err)
			return err
		}
	}

	return nil
}
