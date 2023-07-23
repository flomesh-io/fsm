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

package v1beta1

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/helm"
	"github.com/flomesh-io/fsm/pkg/utils"
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
	"sigs.k8s.io/controller-runtime/pkg/source"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sort"
	"strings"
	"time"
)

var (
	//go:embed chart.tgz
	chartSource []byte

	// namespace <-> active gateway
	activeGateways map[string]*gwv1beta1.Gateway
)

type gatewayValues struct {
	Gateway   *gwv1beta1.Gateway `json:"gwy,omitempty"`
	Listeners []gwpkg.Listener   `json:"listeners,omitempty"`
}

type gatewayReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func init() {
	activeGateways = make(map[string]*gwv1beta1.Gateway)
}

func NewGatewayReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &gatewayReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Gateway"),
		fctx:     ctx,
	}
}

func (r *gatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := &gwv1beta1.Gateway{}
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
			r.fctx.EventHandler.OnDelete(&gwv1beta1.Gateway{
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
		r.fctx.EventHandler.OnDelete(gateway)
		return ctrl.Result{}, nil
	}

	var gatewayClasses gwv1beta1.GatewayClassList
	if err := r.fctx.List(ctx, &gatewayClasses); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list gateway classes: %s", err)
	}

	var effectiveGatewayClass *gwv1beta1.GatewayClass
	for idx, cls := range gatewayClasses.Items {
		if gwutils.IsEffectiveGatewayClass(&cls) {
			effectiveGatewayClass = &gatewayClasses.Items[idx]
			break
		}
	}

	if effectiveGatewayClass == nil {
		log.Warn().Msgf("No effective GatewayClass, ignore processing Gateway resource %s.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// 1. List all Gateways in the namespace whose GatewayClass is current effective class
	gatewayList := &gwv1beta1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(gateway.Namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", gateway.Namespace, err)
		return ctrl.Result{}, err
	}

	// 2. Find the oldest Gateway in the namespace, if CreateTimestamp is equal, then sort by alphabet order asc.
	// If spec.GatewayClassName equals effectiveGatewayClass then it's a valid gateway
	// Otherwise, it's invalid
	validGateways := make([]*gwv1beta1.Gateway, 0)
	invalidGateways := make([]*gwv1beta1.Gateway, 0)

	for _, gw := range gatewayList.Items {
		if string(gw.Spec.GatewayClassName) == effectiveGatewayClass.Name {
			validGateways = append(validGateways, &gw)
		} else {
			invalidGateways = append(invalidGateways, &gw)
		}
	}

	sort.Slice(validGateways, func(i, j int) bool {
		if validGateways[i].CreationTimestamp.Time.Equal(validGateways[j].CreationTimestamp.Time) {
			return validGateways[i].Name < validGateways[j].Name
		} else {
			return validGateways[i].CreationTimestamp.Time.Before(validGateways[j].CreationTimestamp.Time)
		}
	})

	// 3. Set the oldest as Accepted and the rest are unaccepted
	statusChangedGateways := make([]*gwv1beta1.Gateway, 0)
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

	// 5. update listener status of this gateway no matter it's accepted or not
	result, err := r.updateListenerStatus(ctx, gateway)
	if err != nil {
		return result, err
	}

	// 6. after all status of gateways in the namespace have been updated successfully
	//   list all gateways in the namespace and deploy/redeploy the effective one
	activeGateway, result, err := r.findActiveGatewayByNamespace(ctx, gateway.Namespace)
	if err != nil {
		return result, err
	}

	if activeGateway != nil && !isSameGateway(activeGateways[gateway.Namespace], activeGateway) {
		result, err = r.applyGateway(activeGateway)
		if err != nil {
			return result, err
		}

		activeGateways[gateway.Namespace] = activeGateway
	}

	// 7. update addresses of Gateway status if any IP is allocated
	if activeGateway != nil {
		lbSvc := &corev1.Service{}
		key := client.ObjectKey{
			Namespace: activeGateway.Namespace,
			Name:      fmt.Sprintf("fsm-gateway-%s", activeGateway.Namespace),
		}
		if err := r.fctx.Get(ctx, key, lbSvc); err != nil {
			return ctrl.Result{}, err
		}

		if lbSvc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			if len(lbSvc.Status.LoadBalancer.Ingress) > 0 {
				addresses := gatewayAddresses(activeGateway, lbSvc)
				if len(addresses) > 0 {
					activeGateway.Status.Addresses = addresses
					if err := r.fctx.Status().Update(ctx, activeGateway); err != nil {
						//defer r.recorder.Eventf(activeGateway, corev1.EventTypeWarning, "UpdateAddresses", "Failed to update addresses of gateway: %s", err)

						return ctrl.Result{}, err
					}
				}

				defer r.recorder.Eventf(activeGateway, corev1.EventTypeNormal, "UpdateAddresses", "Addresses of gateway is updated: %s", strings.Join(addressesToStrings(addresses), ","))
			} else {
				if len(activeGateway.Status.Addresses) == 0 {
					defer r.recorder.Eventf(activeGateway, corev1.EventTypeNormal, "UpdateAddresses", "Addresses of gateway has not been assigned yet")

					return ctrl.Result{Requeue: true}, nil
				}
			}
		}

		// if there's any previous active gateways and has been assigned addresses, clean it up
		gatewayList := &gwv1beta1.GatewayList{}
		if err := r.fctx.List(ctx, gatewayList, client.InNamespace(activeGateway.Namespace)); err != nil {
			log.Error().Msgf("Failed to list all gateways in namespace %s: %s", activeGateway.Namespace, err)
			return ctrl.Result{}, err
		}

		for _, gw := range gatewayList.Items {
			if gw.Name != activeGateway.Name && len(gw.Status.Addresses) > 0 {
				gw.Status.Addresses = nil
				if err := r.fctx.Status().Update(ctx, &gw); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	r.fctx.EventHandler.OnAdd(gateway)

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateListenerStatus(ctx context.Context, gateway *gwv1beta1.Gateway) (ctrl.Result, error) {
	if len(gateway.Annotations) == 0 {
		gateway.Annotations = make(map[string]string)
	}

	oldHash := gateway.Annotations["gateway.flomesh.io/listeners-hash"]
	hash := utils.SimpleHash(gateway.Spec.Listeners)

	if oldHash != hash {
		gateway.Annotations["gateway.flomesh.io/listeners-hash"] = hash
		if err := r.fctx.Update(ctx, gateway); err != nil {
			return ctrl.Result{}, err
		}

		existingListenerStatus := make(map[gwv1beta1.SectionName]gwv1beta1.ListenerStatus)
		for _, status := range gateway.Status.Listeners {
			existingListenerStatus[status.Name] = status
		}

		gateway.Status.Listeners = nil
		listenerStatus := make([]gwv1beta1.ListenerStatus, 0)
		for _, listener := range gateway.Spec.Listeners {
			status, ok := existingListenerStatus[listener.Name]
			if ok {
				// update existing status
				programmedConditionExists := false
				acceptedConditionExists := false
				for _, cond := range status.Conditions {
					if cond.Type == string(gwv1beta1.ListenerConditionProgrammed) {
						programmedConditionExists = true
					}
					if cond.Type == string(gwv1beta1.ListenerConditionAccepted) {
						acceptedConditionExists = true
					}
				}

				if !programmedConditionExists {
					metautil.SetStatusCondition(&status.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.ListenerConditionProgrammed),
						Status:             metav1.ConditionFalse,
						ObservedGeneration: gateway.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.ListenerReasonInvalid),
						Message:            fmt.Sprintf("Invalid listener %q[:%d]", listener.Name, listener.Port),
					})
				}

				if !acceptedConditionExists {
					metautil.SetStatusCondition(&status.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.ListenerConditionAccepted),
						Status:             metav1.ConditionTrue,
						ObservedGeneration: gateway.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.ListenerReasonAccepted),
						Message:            fmt.Sprintf("listener %q[:%d] is accepted.", listener.Name, listener.Port),
					})
				}
			} else {
				// create new status
				status = gwv1beta1.ListenerStatus{
					Name:           listener.Name,
					SupportedKinds: supportedRouteGroupKindsByProtocol(listener.Protocol),
					Conditions: []metav1.Condition{
						{
							Type:               string(gwv1beta1.ListenerConditionAccepted),
							Status:             metav1.ConditionTrue,
							ObservedGeneration: gateway.Generation,
							LastTransitionTime: metav1.Time{Time: time.Now()},
							Reason:             string(gwv1beta1.ListenerReasonAccepted),
							Message:            fmt.Sprintf("listener %q[:%d] is accepted.", listener.Name, listener.Port),
						},
						{
							Type:               string(gwv1beta1.ListenerConditionProgrammed),
							Status:             metav1.ConditionTrue,
							ObservedGeneration: gateway.Generation,
							LastTransitionTime: metav1.Time{Time: time.Now()},
							Reason:             string(gwv1beta1.ListenerReasonProgrammed),
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

func supportedRouteGroupKindsByProtocol(protocol gwv1beta1.ProtocolType) []gwv1beta1.RouteGroupKind {
	switch protocol {
	case gwv1beta1.HTTPProtocolType, gwv1beta1.HTTPSProtocolType:
		return []gwv1beta1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer("gateway.networking.k8s.io"),
				Kind:  "HTTPRoute",
			},
			{
				Group: gwutils.GroupPointer("gateway.networking.k8s.io"),
				Kind:  "GRPCRoute",
			},
		}
	case gwv1beta1.TLSProtocolType:
		return []gwv1beta1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer("gateway.networking.k8s.io"),
				Kind:  "TLSRoute",
			},
			{
				Group: gwutils.GroupPointer("gateway.networking.k8s.io"),
				Kind:  "TCPRoute",
			},
		}
	case gwv1beta1.TCPProtocolType:
		return []gwv1beta1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer("gateway.networking.k8s.io"),
				Kind:  "TCPRoute",
			},
		}
	case gwv1beta1.UDPProtocolType:
		return []gwv1beta1.RouteGroupKind{
			{
				Group: gwutils.GroupPointer("gateway.networking.k8s.io"),
				Kind:  "UDPRoute",
			},
		}
	}

	return nil
}

func gatewayAddresses(activeGateway *gwv1beta1.Gateway, lbSvc *corev1.Service) []gwv1beta1.GatewayAddress {
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

	addresses := make([]gwv1beta1.GatewayAddress, 0)
	if ipChanged {
		for _, ip := range expectedIPs {
			addresses = append(addresses, gwv1beta1.GatewayAddress{
				Type:  addressTypePointer(gwv1beta1.IPAddressType),
				Value: ip,
			})
		}
	}
	if hostnameChanged {
		for _, hostname := range expectedHostnames {
			addresses = append(addresses, gwv1beta1.GatewayAddress{
				Type:  addressTypePointer(gwv1beta1.HostnameAddressType),
				Value: hostname,
			})
		}
	}

	return addresses
}

func (r *gatewayReconciler) updateStatus(ctx context.Context, gw *gwv1beta1.Gateway) (ctrl.Result, error) {
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

func gatewayIPs(gateway *gwv1beta1.Gateway) []string {
	var ips []string

	for _, addr := range gateway.Status.Addresses {
		if addr.Type == addressTypePointer(gwv1beta1.IPAddressType) && addr.Value != "" {
			ips = append(ips, addr.Value)
		}
	}

	return ips
}

func gatewayHostnames(gateway *gwv1beta1.Gateway) []string {
	var hostnames []string

	for _, addr := range gateway.Status.Addresses {
		if addr.Type == addressTypePointer(gwv1beta1.HostnameAddressType) && addr.Value != "" {
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

func addressTypePointer(addrType gwv1beta1.AddressType) *gwv1beta1.AddressType {
	return &addrType
}

func addressesToStrings(addresses []gwv1beta1.GatewayAddress) []string {
	result := make([]string, 0)
	for _, addr := range addresses {
		result = append(result, addr.Value)
	}

	return result
}

func (r *gatewayReconciler) findActiveGatewayByNamespace(ctx context.Context, namespace string) (*gwv1beta1.Gateway, ctrl.Result, error) {
	gatewayList := &gwv1beta1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", namespace, err)
		return nil, ctrl.Result{}, err
	}

	for _, gw := range gatewayList.Items {
		if gwutils.IsActiveGateway(&gw) {
			return &gw, ctrl.Result{}, nil
		}
	}

	return nil, ctrl.Result{}, nil
}

func isSameGateway(oldGateway, newGateway *gwv1beta1.Gateway) bool {
	return equality.Semantic.DeepEqual(oldGateway, newGateway)
}

func (r *gatewayReconciler) applyGateway(gateway *gwv1beta1.Gateway) (ctrl.Result, error) {
	mc := r.fctx.Config

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

func (r *gatewayReconciler) deriveCodebases(gw *gwv1beta1.Gateway, mc configurator.Configurator) (ctrl.Result, error) {
	gwPath := utils.GatewayCodebasePath(gw.Namespace)
	parentPath := utils.GetDefaultGatewaysPath()
	if _, err := r.fctx.RepoClient.DeriveCodebase(gwPath, parentPath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateConfig(gw *gwv1beta1.Gateway, mc configurator.Configurator) (ctrl.Result, error) {
	// TODO: update pipy repo
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) deployGateway(gw *gwv1beta1.Gateway, mc configurator.Configurator) (ctrl.Result, error) {
	releaseName := fmt.Sprintf("fsm-gateway-%s", gw.Namespace)
	kubeVersion := &chartutil.KubeVersion{
		Version: fmt.Sprintf("v%s.%s.0", "1", "21"),
		Major:   "1",
		Minor:   "21",
	}
	if ctrlResult, err := helm.RenderChart(releaseName, gw, chartSource, mc, r.fctx.Client, r.fctx.Scheme, kubeVersion, resolveValues); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Deploy", "Failed to deploy gateway: %s", err)
		return ctrlResult, err
	}
	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Deploy", "Deploy gateway successfully")

	return ctrl.Result{}, nil
}

func resolveValues(object metav1.Object, mc configurator.Configurator) (map[string]interface{}, error) {
	gateway, ok := object.(*gwv1beta1.Gateway)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of *gwv1beta1.Gateway", object)
	}

	log.Info().Msgf("[GW] Resolving Values ...")

	gwBytes, err := ghodssyaml.Marshal(&gatewayValues{
		Gateway:   gateway,
		Listeners: gwutils.GetValidListenersFromGateway(gateway),
	})
	if err != nil {
		return nil, fmt.Errorf("convert Gateway to yaml, err = %v", err)
	}
	log.Info().Msgf("\n\nGATEWAY VALUES YAML:\n\n\n%s\n\n", string(gwBytes))
	gwValues, err := chartutil.ReadValues(gwBytes)
	if err != nil {
		return nil, err
	}

	finalValues := gwValues.AsMap()

	overrides := []string{
		fmt.Sprintf("fsm.image.registry=%s", mc.GetImageRegistry()),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetFSMNamespace()),
		fmt.Sprintf("fsm.fsmGateway.logLevel=%s", mc.GetGatewayApiLogLevel()),
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func (r *gatewayReconciler) setAccepted(gateway *gwv1beta1.Gateway) {
	metautil.SetStatusCondition(&gateway.Status.Conditions, metav1.Condition{
		Type:               string(gwv1beta1.GatewayConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1beta1.GatewayReasonAccepted),
		Message:            fmt.Sprintf("Gateway %s/%s is accepted.", gateway.Namespace, gateway.Name),
	})
}

func (r *gatewayReconciler) setUnaccepted(gateway *gwv1beta1.Gateway) {
	metautil.SetStatusCondition(&gateway.Status.Conditions, metav1.Condition{
		Type:               string(gwv1beta1.GatewayConditionAccepted),
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
		For(&gwv1beta1.Gateway{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			gateway, ok := obj.(*gwv1beta1.Gateway)
			if !ok {
				log.Error().Msgf("unexpected object type %T", obj)
				return false
			}

			gatewayClass, err := r.fctx.GatewayAPIClient.
				GatewayV1beta1().
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
			&source.Kind{Type: &gwv1beta1.GatewayClass{}},
			handler.EnqueueRequestsFromMapFunc(r.gatewayClassToGateways),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
				if !ok {
					log.Error().Msgf("unexpected object type: %T", obj)
					return false
				}

				return gatewayClass.Spec.ControllerName == constants.GatewayController
			})),
		).
		Complete(r)
}

func (r *gatewayReconciler) gatewayClassToGateways(obj client.Object) []reconcile.Request {
	gatewayClass, ok := obj.(*gwv1beta1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	if gwutils.IsEffectiveGatewayClass(gatewayClass) {
		var gateways gwv1beta1.GatewayList
		if err := r.fctx.List(context.TODO(), &gateways); err != nil {
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
