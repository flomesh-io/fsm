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

package v1

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	ghodssyaml "github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/helm"
	"github.com/flomesh-io/fsm/pkg/utils"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var (
	//go:embed chart.tgz
	chartSource []byte

	// namespace <-> active gateway
	activeGateways map[string]*gwv1.Gateway
)

type gatewayValues struct {
	Gateway   *gwv1.Gateway    `json:"gwy,omitempty"`
	Listeners []gwpkg.Listener `json:"listeners,omitempty"`
}

type gatewayReconciler struct {
	recorder         record.EventRecorder
	fctx             *fctx.ControllerContext
	gatewayAPIClient gwclient.Interface
}

func (r *gatewayReconciler) NeedLeaderElection() bool {
	return true
}

func init() {
	activeGateways = make(map[string]*gwv1.Gateway)
}

// NewGatewayReconciler returns a new reconciler for Gateway resources
func NewGatewayReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &gatewayReconciler{
		recorder:         ctx.Manager.GetEventRecorderFor("Gateway"),
		fctx:             ctx,
		gatewayAPIClient: gatewayApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reconciles a Gateway resource
func (r *gatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := &gwv1.Gateway{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		gateway,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("Gateway resource not found. Ignoring since object must be deleted")
			r.fctx.GatewayEventHandler.OnDelete(&gwv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: req.Namespace,
					Name:      req.Name,
				}})
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get Gateway, %v", err)
		return ctrl.Result{}, err
	}

	if gateway.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(gateway)
		return ctrl.Result{}, nil
	}

	effectiveGatewayClass, err := r.findEffectiveGatewayClass(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if effectiveGatewayClass == nil {
		log.Warn().Msgf("No effective GatewayClass, ignore processing Gateway resource %s.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	result, err := r.updateGatewayStatus(ctx, gateway, effectiveGatewayClass)
	if err != nil {
		return result, err
	}

	// 5. update listener status of this gateway no matter it's accepted or not
	result, err = r.updateListenerStatus(ctx, gateway)
	if err != nil {
		return result, err
	}

	result, err = r.updateGatewayAddresses(ctx, gateway)
	if err != nil || result.RequeueAfter > 0 || result.Requeue {
		return result, err
	}

	r.fctx.GatewayEventHandler.OnAdd(gateway, false)

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) findEffectiveGatewayClass(ctx context.Context) (*gwv1.GatewayClass, error) {
	var gatewayClasses gwv1.GatewayClassList
	if err := r.fctx.List(ctx, &gatewayClasses); err != nil {
		return nil, fmt.Errorf("failed to list gateway classes: %s", err)
	}

	var effectiveGatewayClass *gwv1.GatewayClass
	for idx, cls := range gatewayClasses.Items {
		cls := cls
		if gwutils.IsEffectiveGatewayClass(&cls) {
			effectiveGatewayClass = &gatewayClasses.Items[idx]
			break
		}
	}

	return effectiveGatewayClass, nil
}

func (r *gatewayReconciler) updateGatewayStatus(ctx context.Context, gateway *gwv1.Gateway, effectiveGatewayClass *gwv1.GatewayClass) (ctrl.Result, error) {
	// 1. List all Gateways in the namespace whose GatewayClass is current effective class
	gatewayList := &gwv1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(gateway.Namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", gateway.Namespace, err)
		return ctrl.Result{}, err
	}

	// 2. Find the oldest Gateway in the namespace, if CreateTimestamp is equal, then sort by alphabet order asc.
	// If spec.GatewayClassName equals effectiveGatewayClass then it's a valid gateway
	// Otherwise, it's invalid
	validGateways := make([]*gwv1.Gateway, 0)
	invalidGateways := make([]*gwv1.Gateway, 0)

	for _, gw := range gatewayList.Items {
		gw := gw // fix lint GO-LOOP-REF
		if string(gw.Spec.GatewayClassName) == effectiveGatewayClass.Name {
			validGateways = append(validGateways, &gw)
		} else {
			invalidGateways = append(invalidGateways, &gw)
		}
	}

	sort.Slice(validGateways, func(i, j int) bool {
		if validGateways[i].CreationTimestamp.Time.Equal(validGateways[j].CreationTimestamp.Time) {
			return client.ObjectKeyFromObject(validGateways[i]).String() < client.ObjectKeyFromObject(validGateways[j]).String()
		}

		return validGateways[i].CreationTimestamp.Time.Before(validGateways[j].CreationTimestamp.Time)
	})

	// 3. Set the oldest as Accepted and the rest are unaccepted
	statusChangedGateways := make([]*gwv1.Gateway, 0)
	for i := range validGateways {
		if i == 0 {
			if !gwutils.IsAcceptedGateway(validGateways[i]) {
				r.setAccepted(validGateways[i])
				statusChangedGateways = append(statusChangedGateways, validGateways[i])
			}
		} else {
			if gwutils.IsAcceptedGateway(validGateways[i]) {
				r.setUnaccepted(validGateways[i])
				statusChangedGateways = append(statusChangedGateways, validGateways[i])
			}
		}
	}

	// in case of effective GatewayClass changed or spec.GatewayClassName was changed
	for i := range invalidGateways {
		if gwutils.IsAcceptedGateway(invalidGateways[i]) {
			r.setUnaccepted(invalidGateways[i])
			statusChangedGateways = append(statusChangedGateways, invalidGateways[i])
		}
	}

	// 4. update status
	for _, gw := range statusChangedGateways {
		result, err := r.updateStatus(ctx, gw)
		if err != nil {
			return result, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateGatewayAddresses(ctx context.Context, gateway *gwv1.Gateway) (ctrl.Result, error) {
	// 6. after all status of gateways in the namespace have been updated successfully
	//   list all gateways in the namespace and deploy/redeploy the effective one
	activeGateway, err := r.findActiveGatewayByNamespace(ctx, gateway.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if activeGateway != nil && !isSameGateway(activeGateways[gateway.Namespace], activeGateway) {
		result, err := r.applyGateway(activeGateway)
		if err != nil {
			return result, err
		}

		activeGateways[gateway.Namespace] = activeGateway
	}

	if activeGateway == nil {
		return ctrl.Result{}, nil
	}

	// 7. update addresses of Gateway status if any IP is allocated
	serviceName := lbServiceName(activeGateway)
	if serviceName == "" {
		log.Warn().Msgf("[GW] No supported service protocols for Gateway %s/%s, only TCP and UDP are supported now.", activeGateway.Namespace, activeGateway.Name)
		return ctrl.Result{}, nil
	}

	lbSvc := &corev1.Service{}
	key := client.ObjectKey{
		Namespace: activeGateway.Namespace,
		Name:      serviceName,
	}
	if err := r.fctx.Get(ctx, key, lbSvc); err != nil {
		return ctrl.Result{}, err
	}

	if lbSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return ctrl.Result{}, nil
	}

	if len(lbSvc.Status.LoadBalancer.Ingress) == 0 {
		log.Debug().Msgf("[GW] No ingress IPs found for service %s/%s", lbSvc.Namespace, lbSvc.Name)
		if len(activeGateway.Status.Addresses) == 0 {
			log.Debug().Msgf("[GW] No addresses found for gateway %s/%s", activeGateway.Namespace, activeGateway.Name)
			defer r.recorder.Eventf(activeGateway, corev1.EventTypeNormal, "UpdateAddresses", "Addresses of gateway has not been assigned yet")
		}

		log.Debug().Msgf("[GW] Requeue gateway %s/%s after 3 second", activeGateway.Namespace, activeGateway.Name)
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	}

	addresses := gatewayAddresses(activeGateway, lbSvc)
	if len(addresses) > 0 {
		log.Debug().Msgf("[GW] Addresses of gateway %s/%s will be updated to: %s", activeGateway.Namespace, activeGateway.Name, strings.Join(addressesToStrings(addresses), ","))
		activeGateway.Status.Addresses = addresses
		if err := r.fctx.Status().Update(ctx, activeGateway); err != nil {
			//defer r.recorder.Eventf(activeGateway, corev1.EventTypeWarning, "UpdateAddresses", "Failed to update addresses of gateway: %s", err)

			return ctrl.Result{}, err
		}
	}

	defer r.recorder.Eventf(activeGateway, corev1.EventTypeNormal, "UpdateAddresses", "Addresses of gateway is updated: %s", strings.Join(addressesToStrings(addresses), ","))

	// if there's any previous active gateways and has been assigned addresses, clean it up
	gatewayList := &gwv1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(activeGateway.Namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", activeGateway.Namespace, err)
		return ctrl.Result{}, err
	}

	for _, gw := range gatewayList.Items {
		gw := gw // fix lint GO-LOOP-REF
		if gw.Name != activeGateway.Name && len(gw.Status.Addresses) > 0 {
			gw.Status.Addresses = nil
			if err := r.fctx.Status().Update(ctx, &gw); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func lbServiceName(activeGateway *gwv1.Gateway) string {
	if hasTCP(activeGateway) {
		return fmt.Sprintf("fsm-gateway-%s-tcp", activeGateway.Namespace)
	}

	if hasUDP(activeGateway) {
		return fmt.Sprintf("fsm-gateway-%s-udp", activeGateway.Namespace)
	}

	return ""
}

func (r *gatewayReconciler) updateListenerStatus(ctx context.Context, gateway *gwv1.Gateway) (ctrl.Result, error) {
	if len(gateway.Annotations) == 0 {
		gateway.Annotations = make(map[string]string)
	}

	oldHash := gateway.Annotations[constants.GatewayListenersHashAnnotation]
	hash := utils.SimpleHash(gateway.Spec.Listeners)

	if oldHash != hash {
		gateway.Annotations[constants.GatewayListenersHashAnnotation] = hash
		if err := r.fctx.Update(ctx, gateway); err != nil {
			return ctrl.Result{}, err
		}

		existingListenerStatus := make(map[gwv1.SectionName]gwv1.ListenerStatus)
		for _, status := range gateway.Status.Listeners {
			existingListenerStatus[status.Name] = status
		}

		gateway.Status.Listeners = nil
		listenerStatus := make([]gwv1.ListenerStatus, 0)
		for _, listener := range gateway.Spec.Listeners {
			status, ok := existingListenerStatus[listener.Name]
			if ok {
				// update existing status
				programmedConditionExists := false
				acceptedConditionExists := false
				for _, cond := range status.Conditions {
					if cond.Type == string(gwv1.ListenerConditionProgrammed) {
						programmedConditionExists = true
					}
					if cond.Type == string(gwv1.ListenerConditionAccepted) {
						acceptedConditionExists = true
					}
				}

				if !programmedConditionExists {
					metautil.SetStatusCondition(&status.Conditions, metav1.Condition{
						Type:               string(gwv1.ListenerConditionProgrammed),
						Status:             metav1.ConditionFalse,
						ObservedGeneration: gateway.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1.ListenerReasonInvalid),
						Message:            fmt.Sprintf("Invalid listener %q[:%d]", listener.Name, listener.Port),
					})
				}

				if !acceptedConditionExists {
					metautil.SetStatusCondition(&status.Conditions, metav1.Condition{
						Type:               string(gwv1.ListenerConditionAccepted),
						Status:             metav1.ConditionTrue,
						ObservedGeneration: gateway.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1.ListenerReasonAccepted),
						Message:            fmt.Sprintf("listener %q[:%d] is accepted.", listener.Name, listener.Port),
					})
				}
			} else {
				// create new status
				status = gwv1.ListenerStatus{
					Name:           listener.Name,
					SupportedKinds: supportedRouteGroupKindsByProtocol(listener.Protocol),
					Conditions: []metav1.Condition{
						{
							Type:               string(gwv1.ListenerConditionAccepted),
							Status:             metav1.ConditionTrue,
							ObservedGeneration: gateway.Generation,
							LastTransitionTime: metav1.Time{Time: time.Now()},
							Reason:             string(gwv1.ListenerReasonAccepted),
							Message:            fmt.Sprintf("listener %q[:%d] is accepted.", listener.Name, listener.Port),
						},
						{
							Type:               string(gwv1.ListenerConditionProgrammed),
							Status:             metav1.ConditionTrue,
							ObservedGeneration: gateway.Generation,
							LastTransitionTime: metav1.Time{Time: time.Now()},
							Reason:             string(gwv1.ListenerReasonProgrammed),
							Message:            fmt.Sprintf("Valid listener %q[:%d]", listener.Name, listener.Port),
						},
					},
				}
			}

			listenerStatus = append(listenerStatus, status)
		}

		if len(listenerStatus) > 0 {
			gateway.Status.Listeners = listenerStatus
			if err := r.fctx.Status().Update(ctx, gateway); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func supportedRouteGroupKindsByProtocol(protocol gwv1.ProtocolType) []gwv1.RouteGroupKind {
	switch protocol {
	case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType:
		return []gwv1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
				Kind:  constants.GatewayAPIHTTPRouteKind,
			},
			{
				Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
				Kind:  constants.GatewayAPIGRPCRouteKind,
			},
		}
	case gwv1.TLSProtocolType:
		return []gwv1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
				Kind:  constants.GatewayAPITLSRouteKind,
			},
			{
				Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
				Kind:  constants.GatewayAPITCPRouteKind,
			},
		}
	case gwv1.TCPProtocolType:
		return []gwv1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
				Kind:  constants.GatewayAPITCPRouteKind,
			},
		}
	case gwv1.UDPProtocolType:
		return []gwv1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
				Kind:  constants.GatewayAPIUDPRouteKind,
			},
		}
	}

	return nil
}

func gatewayAddresses(activeGateway *gwv1.Gateway, lbSvc *corev1.Service) []gwv1.GatewayStatusAddress {
	existingIPs := gatewayIPs(activeGateway)
	expectedIPs := lbIPs(lbSvc)
	existingHostnames := gatewayHostnames(activeGateway)
	expectedHostnames := lbHostnames(lbSvc)

	sort.Strings(expectedIPs)
	sort.Strings(existingIPs)
	sort.Strings(existingHostnames)
	sort.Strings(expectedHostnames)

	ipChanged := !utils.StringsEqual(expectedIPs, existingIPs)
	hostnameChanged := !utils.StringsEqual(expectedHostnames, existingHostnames)
	if !ipChanged && !hostnameChanged {
		return nil
	}

	addresses := make([]gwv1.GatewayStatusAddress, 0)
	if ipChanged {
		for _, ip := range expectedIPs {
			addresses = append(addresses, gwv1.GatewayStatusAddress{
				Type:  addressTypePointer(gwv1.IPAddressType),
				Value: ip,
			})
		}
	}
	if hostnameChanged {
		for _, hostname := range expectedHostnames {
			addresses = append(addresses, gwv1.GatewayStatusAddress{
				Type:  addressTypePointer(gwv1.HostnameAddressType),
				Value: hostname,
			})
		}
	}

	return addresses
}

func (r *gatewayReconciler) updateStatus(ctx context.Context, gw *gwv1.Gateway) (ctrl.Result, error) {
	if err := r.fctx.Status().Update(ctx, gw); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "UpdateStatus", "Failed to update status of gateway: %s", err)
		return ctrl.Result{}, err
	}

	if gwutils.IsAcceptedGateway(gw) {
		defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Accepted", "Gateway is accepted")
	} else {
		defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Rejected", "Gateway in unaccepted due to it's not the oldest in namespace %s or its gatewayClassName is incorrect", gw.Namespace)
	}

	return ctrl.Result{}, nil
}

func gatewayIPs(gateway *gwv1.Gateway) []string {
	var ips []string

	for _, addr := range gateway.Status.Addresses {
		if addr.Type == addressTypePointer(gwv1.IPAddressType) && addr.Value != "" {
			ips = append(ips, addr.Value)
		}
	}

	return ips
}

func gatewayHostnames(gateway *gwv1.Gateway) []string {
	var hostnames []string

	for _, addr := range gateway.Status.Addresses {
		if addr.Type == addressTypePointer(gwv1.HostnameAddressType) && addr.Value != "" {
			hostnames = append(hostnames, addr.Value)
		}
	}

	return hostnames
}

func lbIPs(svc *corev1.Service) []string {
	var ips []string

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		}
	}

	return ips
}

func lbHostnames(svc *corev1.Service) []string {
	var hostnames []string

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.Hostname != "" {
			hostnames = append(hostnames, ingress.Hostname)
		}
	}

	return hostnames
}

func addressTypePointer(addrType gwv1.AddressType) *gwv1.AddressType {
	return &addrType
}

func addressesToStrings(addresses []gwv1.GatewayStatusAddress) []string {
	result := make([]string, 0)
	for _, addr := range addresses {
		result = append(result, addr.Value)
	}

	return result
}

func (r *gatewayReconciler) findActiveGatewayByNamespace(ctx context.Context, namespace string) (*gwv1.Gateway, error) {
	gatewayList := &gwv1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", namespace, err)
		return nil, err
	}

	for _, gw := range gatewayList.Items {
		gw := gw // fix lint GO-LOOP-REF
		if gwutils.IsActiveGateway(&gw) {
			return &gw, nil
		}
	}

	return nil, nil
}

func isSameGateway(oldGateway, newGateway *gwv1.Gateway) bool {
	return equality.Semantic.DeepEqual(oldGateway, newGateway)
}

func (r *gatewayReconciler) applyGateway(gateway *gwv1.Gateway) (ctrl.Result, error) {
	mc := r.fctx.Configurator

	result, err := r.deriveCodebases(gateway, mc)
	if err != nil {
		defer r.recorder.Eventf(gateway, corev1.EventTypeWarning, "DeriveCodebase", "Failed to derive codebase of gateway: %s", err)

		return result, err
	}
	defer r.recorder.Eventf(gateway, corev1.EventTypeNormal, "DeriveCodebase", "Derive codebase of gateway successfully")

	result, err = r.updateConfig(gateway, mc)
	if err != nil {
		defer r.recorder.Eventf(gateway, corev1.EventTypeWarning, "UpdateRepo", "Failed to update repo config of gateway: %s", err)

		return result, err
	}
	defer r.recorder.Eventf(gateway, corev1.EventTypeNormal, "UpdateRepo", "Update repo config of gateway successfully")

	return r.deployGateway(gateway, mc)
}

func (r *gatewayReconciler) deriveCodebases(gw *gwv1.Gateway, _ configurator.Configurator) (ctrl.Result, error) {
	gwPath := utils.GatewayCodebasePath(gw.Namespace)
	parentPath := utils.GetDefaultGatewaysPath()
	if err := r.fctx.RepoClient.DeriveCodebase(gwPath, parentPath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateConfig(_ *gwv1.Gateway, _ configurator.Configurator) (ctrl.Result, error) {
	// TODO: update pipy repo
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) deployGateway(gw *gwv1.Gateway, mc configurator.Configurator) (ctrl.Result, error) {
	actionConfig := helm.ActionConfig(gw.Namespace, log.Debug().Msgf)
	templateClient := helm.TemplateClient(
		actionConfig,
		fmt.Sprintf("fsm-gateway-%s", gw.Namespace),
		gw.Namespace,
		constants.KubeVersion121,
	)
	if ctrlResult, err := helm.RenderChart(templateClient, gw, chartSource, mc, r.fctx.Client, r.fctx.Scheme, r.resolveValues); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Deploy", "Failed to deploy gateway: %s", err)
		return ctrlResult, err
	}
	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Deploy", "Deploy gateway successfully")

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) resolveValues(object metav1.Object, mc configurator.Configurator) (map[string]interface{}, error) {
	gateway, ok := object.(*gwv1.Gateway)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of *gwv1.Gateway", object)
	}

	log.Debug().Msgf("[GW] Resolving Values ...")

	gwBytes, err := ghodssyaml.Marshal(&gatewayValues{
		Gateway:   gateway,
		Listeners: gwutils.GetValidListenersFromGateway(gateway),
	})
	if err != nil {
		return nil, fmt.Errorf("convert Gateway to yaml, err = %v", err)
	}
	log.Debug().Msgf("\n\nGATEWAY VALUES YAML:\n\n\n%s\n\n", string(gwBytes))
	gwValues, err := chartutil.ReadValues(gwBytes)
	if err != nil {
		return nil, err
	}

	finalValues := gwValues.AsMap()

	overrides := []string{
		fmt.Sprintf("fsm.image.registry=%s", mc.GetImageRegistry()),
		fmt.Sprintf("fsm.image.tag=%s", mc.GetImageTag()),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.GetImagePullPolicy()),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetFSMNamespace()),
		fmt.Sprintf("fsm.fsmGateway.logLevel=%s", mc.GetFSMGatewayLogLevel()),
		fmt.Sprintf("fsm.meshName=%s", r.fctx.MeshName),
		fmt.Sprintf("fsm.curlImage=%s", mc.GetCurlImage()),
		fmt.Sprintf("hasTCP=%t", hasTCP(gateway)),
		fmt.Sprintf("hasUDP=%t", hasUDP(gateway)),
		fmt.Sprintf("fsm.fsmGateway.replicas=%d", replicas(gateway, constants.GatewayReplicasAnnotation, 1)),
		fmt.Sprintf("fsm.fsmGateway.resources.requests.cpu=%s", resources(gateway, constants.GatewayCPUAnnotation, resource.MustParse("0.5")).String()),
		fmt.Sprintf("fsm.fsmGateway.resources.requests.memory=%s", resources(gateway, constants.GatewayMemoryAnnotation, resource.MustParse("128M")).String()),
		fmt.Sprintf("fsm.fsmGateway.resources.limits.cpu=%s", resources(gateway, constants.GatewayCPULimitAnnotation, resource.MustParse("2")).String()),
		fmt.Sprintf("fsm.fsmGateway.resources.limits.memory=%s", resources(gateway, constants.GatewayMemoryLimitAnnotation, resource.MustParse("1G")).String()),
		fmt.Sprintf("fsm.fsmGateway.enablePodDisruptionBudget=%t", enabled(gateway, constants.GatewayPodDisruptionBudgetAnnotation, false)),
		fmt.Sprintf("fsm.fsmGateway.autoScale.enable=%t", enabled(gateway, constants.GatewayAutoScalingAnnotation, false)),
		fmt.Sprintf("fsm.fsmGateway.autoScale.minReplicas=%d", replicas(gateway, constants.GatewayAutoScalingMinReplicasAnnotation, 1)),
		fmt.Sprintf("fsm.fsmGateway.autoScale.maxReplicas=%d", replicas(gateway, constants.GatewayAutoScalingMaxReplicasAnnotation, 5)),
		fmt.Sprintf("fsm.fsmGateway.autoScale.cpu.targetAverageUtilization=%d", percentage(gateway, constants.GatewayAutoScalingTargetCPUUtilizationPercentageAnnotation, 80)),
		fmt.Sprintf("fsm.fsmGateway.autoScale.memory.targetAverageUtilization=%d", percentage(gateway, constants.GatewayAutoScalingTargetMemoryUtilizationPercentageAnnotation, 80)),
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func (r *gatewayReconciler) setAccepted(gateway *gwv1.Gateway) {
	metautil.SetStatusCondition(&gateway.Status.Conditions, metav1.Condition{
		Type:               string(gwv1.GatewayConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayReasonAccepted),
		Message:            fmt.Sprintf("Gateway %s/%s is accepted.", gateway.Namespace, gateway.Name),
	})
}

func (r *gatewayReconciler) setUnaccepted(gateway *gwv1.Gateway) {
	metautil.SetStatusCondition(&gateway.Status.Conditions, metav1.Condition{
		Type:               string(gwv1.GatewayConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Unaccepted",
		Message:            fmt.Sprintf("Gateway %s/%s is not accepted as it's not the oldest one in namespace %q.", gateway.Namespace, gateway.Name, gateway.Namespace),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.Gateway{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			gateway, ok := obj.(*gwv1.Gateway)
			if !ok {
				log.Error().Msgf("unexpected object type %T", obj)
				return false
			}

			gatewayClass, err := r.gatewayAPIClient.
				GatewayV1().
				GatewayClasses().
				Get(context.TODO(), string(gateway.Spec.GatewayClassName), metav1.GetOptions{})
			if err != nil {
				log.Error().Msgf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
				return false
			}

			if gatewayClass.Spec.ControllerName != constants.GatewayController {
				log.Warn().Msgf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, constants.GatewayController)
				return false
			}

			return true
		}))).
		Watches(
			&gwv1.GatewayClass{},
			handler.EnqueueRequestsFromMapFunc(r.gatewayClassToGateways),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				gatewayClass, ok := obj.(*gwv1.GatewayClass)
				if !ok {
					log.Error().Msgf("unexpected object type: %T", obj)
					return false
				}

				return gatewayClass.Spec.ControllerName == constants.GatewayController
			})),
		).
		Complete(r)
}

func (r *gatewayReconciler) gatewayClassToGateways(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayClass, ok := obj.(*gwv1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	if gwutils.IsEffectiveGatewayClass(gatewayClass) {
		var gateways gwv1.GatewayList
		if err := r.fctx.List(ctx, &gateways); err != nil {
			log.Error().Msgf("error listing gateways: %s", err)
			return nil
		}

		var reconciles []reconcile.Request
		for _, gw := range gateways.Items {
			if string(gw.Spec.GatewayClassName) == gatewayClass.GetName() {
				reconciles = append(reconciles, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: gw.Namespace,
						Name:      gw.Name,
					},
				})
			}
		}

		return reconciles
	}

	return nil
}
