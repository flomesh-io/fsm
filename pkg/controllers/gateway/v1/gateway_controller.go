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
	"time"

	"github.com/flomesh-io/fsm/pkg/gateway/status/gw"

	ghodssyaml "github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/version"

	"sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
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
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"github.com/flomesh-io/fsm/pkg/helm"
	"github.com/flomesh-io/fsm/pkg/utils"
)

var (
	//go:embed chart.tgz
	chartSource []byte

	// namespace <-> active gateway
	activeGateways map[string]*gwv1.Gateway
)

type listener struct {
	Name     gwv1.SectionName  `json:"name"`
	Port     gwv1.PortNumber   `json:"port"`
	Protocol gwv1.ProtocolType `json:"protocol"`
}

type gatewayReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *gatewayReconciler) NeedLeaderElection() bool {
	return true
}

//func init() {
//	activeGateways = make(map[string]*gwv1.Gateway)
//}

// NewGatewayReconciler returns a new reconciler for Gateway resources
func NewGatewayReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &gatewayReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Gateway"),
		fctx:     ctx,
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

	//effectiveGatewayClass, err := gwutils.FindEffectiveGatewayClass(r.fctx.Manager.GetCache())gwutils.FindEffectiveGatewayClass(r.fctx.Manager.GetCache())
	effectiveGatewayClass, err := gwutils.FindGatewayClassByName(r.fctx.Manager.GetCache(), string(gateway.Spec.GatewayClassName))
	if err != nil {
		return ctrl.Result{}, err
	}

	if effectiveGatewayClass == nil {
		log.Warn().Msgf("No effective GatewayClass, ignore processing Gateway resource %s.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	//if string(gateway.Spec.GatewayClassName) != effectiveGatewayClass.Name {
	//	log.Warn().Msgf("Ignore Gateway %s/%s as it's GatewayClassName %q is not effective", gateway.Namespace, gateway.Name, gateway.Spec.GatewayClassName)
	//	return ctrl.Result{}, nil
	//}

	update := gw.NewGatewayStatusUpdate(
		gateway,
		&gateway.ObjectMeta,
		&gateway.TypeMeta,
		&gateway.Status,
	)

	if result, err := r.computeGatewayStatus(ctx, gateway, effectiveGatewayClass, update); err != nil || result.RequeueAfter > 0 || result.Requeue {
		return result, err
	}

	r.fctx.StatusUpdater.Send(status.Update{
		Resource:       &gwv1.Gateway{},
		NamespacedName: client.ObjectKeyFromObject(gateway),
		Mutator:        update,
	})

	r.fctx.GatewayEventHandler.OnAdd(gateway, false)

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) computeGatewayStatus(ctx context.Context, gateway *gwv1.Gateway, effectiveGatewayClass *gwv1.GatewayClass, update *gw.GatewayStatusUpdate) (ctrl.Result, error) {
	// 1. compute gateway effective status
	//effective, err := r.computeGatewayEffectiveCondition(ctx, gateway, effectiveGatewayClass, update)
	//if err != nil {
	//	return ctrl.Result{}, err
	//}

	// 2. stop processing other status if not effective
	//if !effective {
	//	return ctrl.Result{}, nil
	//}

	// 3. compute listener status & accepted status
	result, err := r.computeListenerStatus(ctx, gateway, update)
	if err != nil {
		return result, err
	}

	// 4. so far, it's accepted, just deploy it if not
	//if !isSameGateway(activeGateways[gateway.Namespace], gateway) {
	if result, err := r.applyGateway(gateway, update); err != nil {
		return result, err
	}
	//}

	// 5. compute gateway address and programmed status
	result, err = r.updateGatewayAddresses(ctx, gateway, update)
	if err != nil || result.RequeueAfter > 0 || result.Requeue {
		return result, err
	}

	if !update.ConditionExists(gwv1.GatewayConditionAccepted) {
		r.recorder.Eventf(gateway, corev1.EventTypeNormal, "Accepted", "Gateway is accepted")

		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionTrue,
			gwv1.GatewayReasonAccepted,
			"Gateway is accepted",
		)
	}

	if !update.ConditionExists(gwv1.GatewayConditionProgrammed) {
		r.recorder.Eventf(gateway, corev1.EventTypeNormal, "Programmed", "Gateway is programmed")

		update.AddCondition(
			gwv1.GatewayConditionProgrammed,
			metav1.ConditionTrue,
			gwv1.GatewayReasonProgrammed,
			"Gateway is programmed",
		)
	}

	return ctrl.Result{}, nil
}

//func (r *gatewayReconciler) computeGatewayEffectiveCondition(ctx context.Context, gateway *gwv1.Gateway, effectiveGatewayClass *gwv1.GatewayClass, update *gw.GatewayStatusUpdate) (bool, error) {
//	// 1. List all Gateways in the namespace whose GatewayClass is current effective class
//	c := r.fctx.Manager.GetCache()
//	gatewayList := &gwv1.GatewayList{}
//	if err := c.List(ctx, gatewayList, &client.ListOptions{
//		FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, effectiveGatewayClass.Name),
//		Namespace:     gateway.Namespace,
//	}); err != nil {
//		log.Error().Msgf("error listing gateways: %s", err)
//		return false, err
//	}
//
//	if len(gatewayList.Items) == 0 {
//		return false, fmt.Errorf("no gateway found in namespace %s with GatewayClass %s", gateway.Namespace, effectiveGatewayClass.Name)
//	}
//
//	// 2. Sort the gateways by CreationTimestamp
//	validGateways := gwutils.SortResources(gwutils.ToSlicePtr(gatewayList.Items))
//
//	// 3. Current Gateway is not the oldest, it should be ineffective
//	if validGateways[0].Name != gateway.Name {
//		update.AddCondition(
//			v1.GatewayConditionEffective,
//			metav1.ConditionFalse,
//			v1.GatewayReasonNotOldest,
//			fmt.Sprintf("Gateway is not effective as it's not the oldest in namespace %q.", gateway.Namespace),
//		)
//
//		return false, nil
//	}
//
//	// 4. Current Gateway is the oldest, it's effective
//	update.AddCondition(
//		v1.GatewayConditionEffective,
//		metav1.ConditionTrue,
//		v1.GatewayReasonEffective,
//		"Gateway is effective.",
//	)
//
//	return true, nil
//}

func (r *gatewayReconciler) computeListenerStatus(_ context.Context, gateway *gwv1.Gateway, update *gw.GatewayStatusUpdate) (ctrl.Result, error) {
	invalidListeners := invalidateListeners(gateway.Spec.Listeners)
	for name, cond := range invalidListeners {
		update.AddListenerCondition(
			string(name),
			gwv1.ListenerConditionType(cond.Type),
			cond.Status,
			gwv1.ListenerConditionReason(cond.Reason),
			cond.Message,
		)
	}

	addInvalidListenerCondition := func(name gwv1.SectionName, msg string) {
		update.AddListenerCondition(
			string(name),
			gwv1.ListenerConditionProgrammed,
			metav1.ConditionFalse,
			gwv1.ListenerReasonInvalid,
			msg,
		)
	}

	for _, listener := range gateway.Spec.Listeners {
		groupKinds := supportedRouteGroupKinds(gateway, listener, update)
		update.SetListenerSupportedKinds(string(listener.Name), groupKinds)

		if listener.AllowedRoutes != nil && listener.AllowedRoutes.Namespaces != nil &&
			listener.AllowedRoutes.Namespaces.From != nil && *listener.AllowedRoutes.Namespaces.From == gwv1.NamespacesFromSelector {
			if listener.AllowedRoutes.Namespaces.Selector == nil {
				addInvalidListenerCondition(listener.Name, "Listener.AllowedRoutes.Namespaces.Selector is required when Listener.AllowedRoutes.Namespaces.From is set to \"Selector\".")
				continue
			}

			if len(listener.AllowedRoutes.Namespaces.Selector.MatchExpressions)+len(listener.AllowedRoutes.Namespaces.Selector.MatchLabels) == 0 {
				addInvalidListenerCondition(listener.Name, "Listener.AllowedRoutes.Namespaces.Selector must specify at least one MatchLabel or MatchExpression.")
				continue
			}

			var err error
			_, err = metav1.LabelSelectorAsSelector(listener.AllowedRoutes.Namespaces.Selector)
			if err != nil {
				addInvalidListenerCondition(listener.Name, fmt.Sprintf("Error parsing Listener.AllowedRoutes.Namespaces.Selector: %v.", err))
				continue
			}
		}

		if _, ok := invalidListeners[listener.Name]; ok {
			continue
		}

		listenerStatus := update.GetListenerStatus(string(listener.Name))

		if listenerStatus == nil || len(listenerStatus.Conditions) == 0 {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionProgrammed,
				metav1.ConditionTrue,
				gwv1.ListenerReasonProgrammed,
				"Valid listener",
			)
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionAccepted,
				metav1.ConditionTrue,
				gwv1.ListenerReasonAccepted,
				"Listener accepted",
			)
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionTrue,
				gwv1.ListenerReasonResolvedRefs,
				"Listener references resolved",
			)
		} else {
			if metautil.FindStatusCondition(listenerStatus.Conditions, string(gwv1.ListenerConditionProgrammed)) == nil {
				addInvalidListenerCondition(listener.Name, "Invalid listener, see other listener conditions for details")
			}

			if metautil.FindStatusCondition(listenerStatus.Conditions, string(gwv1.ListenerConditionAccepted)) == nil {
				update.AddListenerCondition(
					string(listener.Name),
					gwv1.ListenerConditionAccepted,
					metav1.ConditionTrue,
					gwv1.ListenerReasonAccepted,
					"Listener accepted",
				)
			}

			if metautil.FindStatusCondition(listenerStatus.Conditions, string(gwv1.ListenerConditionResolvedRefs)) == nil {
				update.AddListenerCondition(
					string(listener.Name),
					gwv1.ListenerConditionResolvedRefs,
					metav1.ConditionTrue,
					gwv1.ListenerReasonResolvedRefs,
					"Listener references resolved",
				)
			}
		}
	}

	return ctrl.Result{}, nil
}

func invalidateListeners(listeners []gwv1.Listener) map[gwv1.SectionName]metav1.Condition {
	conflictCondition := func(msg string) metav1.Condition {
		return metav1.Condition{
			Type:    string(gwv1.ListenerConditionConflicted),
			Status:  metav1.ConditionTrue,
			Reason:  string(gwv1.ListenerReasonHostnameConflict),
			Message: msg,
		}
	}

	invalidListenerConditions := map[gwv1.SectionName]metav1.Condition{}

	for i, listener := range listeners {
		// Check for a valid hostname.
		if hostname := ptr.Deref(listener.Hostname, ""); len(hostname) > 0 {
			if err := isValidHostname(string(hostname)); err != nil {
				invalidListenerConditions[listener.Name] = metav1.Condition{
					Type:    string(gwv1.ListenerConditionProgrammed),
					Status:  metav1.ConditionFalse,
					Reason:  string(gwv1.ListenerReasonInvalid),
					Message: fmt.Sprintf("Invalid hostname %q: %v", hostname, err),
				}
				continue
			}
		}

		// Check for a supported protocol.
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType, gwv1.UDPProtocolType:
		default:
			invalidListenerConditions[listener.Name] = conflictCondition(fmt.Sprintf("Listener protocol %q is unsupported, must be one of HTTP, HTTPS, TLS, TCP or UDP", listener.Protocol))
			continue
		}

		if listener.Port > 60000 && listener.Port <= 65535 {
			invalidListenerConditions[listener.Name] = conflictCondition(fmt.Sprintf("Listener port %d is invalid, must be in the range 1-60000", listener.Port))
			continue
		}

		func() {
			for j := range i {
				otherListener := listeners[j]

				if listener.Port != otherListener.Port {
					continue
				}

				if listener.Protocol != otherListener.Protocol {
					// same port, different protocol, not allowed
					invalidListenerConditions[listener.Name] = metav1.Condition{
						Type:    string(gwv1.ListenerConditionConflicted),
						Status:  metav1.ConditionTrue,
						Reason:  string(gwv1.ListenerReasonProtocolConflict),
						Message: "All Listener protocols for a given port must be the same",
					}

					return
				}

				switch listener.Protocol {
				case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
					// Hostname conflict
					if ptr.Deref(listener.Hostname, "") == ptr.Deref(otherListener.Hostname, "") {
						invalidListenerConditions[listener.Name] = metav1.Condition{
							Type:    string(gwv1.ListenerConditionConflicted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gwv1.ListenerReasonHostnameConflict),
							Message: "All Listener hostnames for a given port must be unique",
						}
						return
					}
				}
			}
		}()
	}

	return invalidListenerConditions
}

func supportedRouteGroupKinds(_ *gwv1.Gateway, listener gwv1.Listener, update *gw.GatewayStatusUpdate) []gwv1.RouteGroupKind {
	if len(listener.AllowedRoutes.Kinds) == 0 {
		switch listener.Protocol {
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
	}

	kinds := make([]gwv1.RouteGroupKind, 0)

	for _, routeKind := range listener.AllowedRoutes.Kinds {
		if routeKind.Group != nil && *routeKind.Group != constants.GatewayAPIGroup {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("Group %q is not supported, group must be %q", *routeKind.Group, gwv1.GroupName),
			)

			continue
		}

		if routeKind.Kind != constants.GatewayAPIHTTPRouteKind &&
			routeKind.Kind != constants.GatewayAPITLSRouteKind &&
			routeKind.Kind != constants.GatewayAPIGRPCRouteKind &&
			routeKind.Kind != constants.GatewayAPITCPRouteKind &&
			routeKind.Kind != constants.GatewayAPIUDPRouteKind {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("Kind %q is not supported, kind must be %q, %q, %q, %q or %q", routeKind.Kind, constants.GatewayAPIHTTPRouteKind, constants.GatewayAPIGRPCRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPITCPRouteKind, constants.GatewayAPIUDPRouteKind),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPIHTTPRouteKind && listener.Protocol != gwv1.HTTPProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("HTTPRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPIGRPCRouteKind && listener.Protocol != gwv1.HTTPProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("GRPCRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPITLSRouteKind && listener.Protocol != gwv1.TLSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("TLSRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPITCPRouteKind && listener.Protocol != gwv1.TCPProtocolType && listener.Protocol != gwv1.TLSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("TCPRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPIUDPRouteKind && listener.Protocol != gwv1.UDPProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("UDPRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		kinds = append(kinds, gwv1.RouteGroupKind{
			Group: routeKind.Group,
			Kind:  routeKind.Kind,
		})
	}

	return kinds
}

func (r *gatewayReconciler) computeGatewayProgrammedCondition(ctx context.Context, gw *gwv1.Gateway, addresses []gwv1.GatewayStatusAddress, update *gw.GatewayStatusUpdate) bool {
	if len(addresses) == 0 {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Addresses", "No addresses have been assigned to the Gateway")

		update.AddCondition(
			gwv1.GatewayConditionProgrammed,
			metav1.ConditionFalse,
			gwv1.GatewayReasonAddressNotAssigned,
			"No addresses have been assigned to the Gateway",
		)
		return false
	}

	//isSpecAddressAssigned := func(specAddresses []gwv1.GatewayAddress, statusAddresses []gwv1.GatewayStatusAddress) bool {
	//	if len(specAddresses) == 0 {
	//		return true
	//	}
	//
	//	for _, specAddress := range specAddresses {
	//		for _, statusAddress := range statusAddresses {
	//			// Types must match
	//			if ptr.Deref(specAddress.Type, gwv1.IPAddressType) != ptr.Deref(statusAddress.Type, gwv1.IPAddressType) {
	//				continue
	//			}
	//
	//			// Values must match
	//			if specAddress.Value != statusAddress.Value {
	//				continue
	//			}
	//
	//			return true
	//		}
	//	}
	//
	//	return false
	//}
	//if !isSpecAddressAssigned(gw.Spec.Addresses, addresses) {
	//	defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Addresses", "None of the addresses in Spec.Addresses have been assigned to the Gateway")
	//
	//	return gatewayAddressNotAssignedCondition(gw, "None of the addresses in Spec.Addresses have been assigned to the Gateway"), false
	//}

	deployment := r.gatewayDeployment(ctx, gw)
	if deployment == nil || deployment.Status.AvailableReplicas == 0 {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Unavailable", "Gateway Deployment replicas unavailable")

		update.AddCondition(
			gwv1.GatewayConditionProgrammed,
			metav1.ConditionFalse,
			gwv1.GatewayReasonNoResources,
			"Deployment replicas unavailable",
		)

		return false
	}

	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Programmed", "Gateway is programmed")

	update.AddCondition(
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionTrue,
		gwv1.GatewayReasonProgrammed,
		fmt.Sprintf("Address assigned to the Gateway, %d/%d Deployment replicas available", deployment.Status.AvailableReplicas, deployment.Status.Replicas),
	)

	return true
}

func (r *gatewayReconciler) updateGatewayAddresses(ctx context.Context, gateway *gwv1.Gateway, update *gw.GatewayStatusUpdate) (ctrl.Result, error) {
	addresses := r.gatewayAddresses(ctx, gateway)
	programmed := r.computeGatewayProgrammedCondition(ctx, gateway, addresses, update)

	if !programmed {
		log.Debug().Msgf("[GW] Requeue gateway %s/%s after 5 second", gateway.Namespace, gateway.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	update.SetAddresses(addresses)

	allListenersProgrammed := func(gw *gwv1.Gateway) bool {
		for _, listener := range gw.Status.Listeners {
			if !gwutils.IsListenerProgrammed(listener) {
				return false
			}
		}

		return true
	}

	if !allListenersProgrammed(gateway) {
		defer r.recorder.Eventf(gateway, corev1.EventTypeWarning, "Listeners", "Not All listeners are programmed")

		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionFalse,
			gwv1.GatewayReasonListenersNotValid,
			"Not all listeners are programmed",
		)
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) gatewayService(ctx context.Context, gateway *gwv1.Gateway) (*corev1.Service, error) {
	gatewayServiceName := func(activeGateway *gwv1.Gateway) string {
		if hasTCP(activeGateway) {
			return fmt.Sprintf("fsm-gateway-%s-%s-tcp", activeGateway.Namespace, activeGateway.Name)
		}

		if hasUDP(activeGateway) {
			return fmt.Sprintf("fsm-gateway-%s-%s-udp", activeGateway.Namespace, activeGateway.Name)
		}

		return ""
	}

	serviceName := gatewayServiceName(gateway)
	if serviceName == "" {
		log.Warn().Msgf("[GW] No supported service protocols for Gateway %s/%s, only TCP and UDP are supported now.", gateway.Namespace, gateway.Name)
		return nil, fmt.Errorf("no supported service protocols for Gateway %s/%s, only TCP and UDP are supported", gateway.Namespace, gateway.Name)
	}

	svc := &corev1.Service{}
	key := client.ObjectKey{
		Namespace: gateway.Namespace,
		Name:      serviceName,
	}
	if err := r.fctx.Get(ctx, key, svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func (r *gatewayReconciler) gatewayAddresses(ctx context.Context, gw *gwv1.Gateway) []gwv1.GatewayStatusAddress {
	gwSvc, err := r.gatewayService(ctx, gw)
	if err != nil {
		log.Error().Msgf("Failed to get gateway service: %s", err)
		return nil
	}

	var addresses, hostnames []string

	switch gwSvc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		for i := range gwSvc.Status.LoadBalancer.Ingress {
			switch {
			case len(gwSvc.Status.LoadBalancer.Ingress[i].IP) > 0:
				addresses = append(addresses, gwSvc.Status.LoadBalancer.Ingress[i].IP)
			case len(gwSvc.Status.LoadBalancer.Ingress[i].Hostname) > 0:
				if gwSvc.Status.LoadBalancer.Ingress[i].Hostname == "localhost" {
					addresses = append(addresses, "127.0.0.1")
				}
				hostnames = append(hostnames, gwSvc.Status.LoadBalancer.Ingress[i].Hostname)
			}
		}
	case corev1.ServiceTypeNodePort:
		addresses = append(addresses, r.getNodeIPs(ctx, gwSvc)...)
	default:
		return nil
	}

	var gwAddresses []gwv1.GatewayStatusAddress
	for i := range addresses {
		addr := gwv1.GatewayStatusAddress{
			Type:  ptr.To(gwv1.IPAddressType),
			Value: addresses[i],
		}
		gwAddresses = append(gwAddresses, addr)
	}

	for i := range hostnames {
		addr := gwv1.GatewayStatusAddress{
			Type:  ptr.To(gwv1.HostnameAddressType),
			Value: hostnames[i],
		}
		gwAddresses = append(gwAddresses, addr)
	}

	return gwAddresses
}

func (r *gatewayReconciler) getNodeIPs(ctx context.Context, svc *corev1.Service) []string {
	pods := &corev1.PodList{}
	if err := r.fctx.List(
		ctx,
		pods,
		client.InNamespace(svc.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(svc.Spec.Selector),
		},
	); err != nil {
		log.Error().Msgf("Failed to get pods: %s", err)
		return nil
	}

	extIPs := sets.New[string]()
	intIPs := sets.New[string]()

	for _, pod := range pods.Items {
		if pod.Spec.NodeName == "" || pod.Status.PodIP == "" {
			continue
		}

		if !utils.IsPodStatusConditionTrue(pod.Status.Conditions, corev1.PodReady) {
			continue
		}

		node := &corev1.Node{}
		if err := r.fctx.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, node); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			log.Error().Msgf("Failed to get node %q: %s", pod.Spec.NodeName, err)
			return nil
		}

		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeExternalIP:
				extIPs.Insert(addr.Address)
			case corev1.NodeInternalIP:
				intIPs.Insert(addr.Address)
			default:
				continue
			}
		}
	}

	var nodeIPs []string
	if len(extIPs) > 0 {
		nodeIPs = extIPs.UnsortedList()
	} else {
		nodeIPs = intIPs.UnsortedList()
	}

	if version.IsDualStackEnabled(r.fctx.KubeClient) {
		ips, err := utils.FilterByIPFamily(nodeIPs, svc)
		if err != nil {
			return nil
		}

		nodeIPs = ips
	}

	return nodeIPs
}

func (r *gatewayReconciler) gatewayDeployment(ctx context.Context, gw *gwv1.Gateway) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{
		Namespace: gw.Namespace,
		Name:      fmt.Sprintf("fsm-gateway-%s-%s", gw.Namespace, gw.Name),
	}

	if err := r.fctx.Get(ctx, key, deployment); err != nil {
		if errors.IsNotFound(err) {
			log.Warn().Msgf("Deployment %s not found", key.String())
			return nil
		}

		log.Error().Msgf("Failed to get deployment %s: %s", key.String(), err)
		return nil
	}

	return deployment
}

//func isSameGateway(oldGateway, newGateway *gwv1.Gateway) bool {
//	return equality.Semantic.DeepEqual(oldGateway, newGateway)
//}

func (r *gatewayReconciler) applyGateway(gateway *gwv1.Gateway, update *gw.GatewayStatusUpdate) (ctrl.Result, error) {
	if len(gateway.Spec.Addresses) > 0 {
		update.AddCondition(
			gwv1.GatewayConditionProgrammed,
			metav1.ConditionFalse,
			gwv1.GatewayReasonAddressNotAssigned,
			".spec.addresses is not supported yet.",
		)

		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionFalse,
			gwv1.GatewayReasonUnsupportedAddress,
			".spec.addresses is not supported yet.",
		)

		r.recorder.Eventf(gateway, corev1.EventTypeWarning, "Address", ".spec.addresses is not supported yet.")

		return ctrl.Result{}, nil
	}

	mc := r.fctx.Configurator

	result, err := r.deriveCodebases(gateway, mc)
	if err != nil {
		return result, err
	}

	result, err = r.updateConfig(gateway, mc)
	if err != nil {
		return result, err
	}

	result, err = r.deployGateway(gateway, mc, update)
	if err != nil {
		return result, err
	}

	//activeGateways[gateway.Namespace] = gateway

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) deriveCodebases(gw *gwv1.Gateway, _ configurator.Configurator) (ctrl.Result, error) {
	gwPath := utils.GatewayCodebasePath(gw.Namespace, gw.Name)
	parentPath := utils.GetDefaultGatewaysPath()
	if err := r.fctx.RepoClient.DeriveCodebase(gwPath, parentPath); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Codebase", "Failed to derive codebase of gateway: %s", err)

		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateConfig(gw *gwv1.Gateway, _ configurator.Configurator) (ctrl.Result, error) {
	// TODO: update pipy repo
	// defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Repo", "Failed to update repo config of gateway: %s", err)
	//defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Repo", "Update repo config of gateway successfully")
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) deployGateway(gw *gwv1.Gateway, mc configurator.Configurator, update *gw.GatewayStatusUpdate) (ctrl.Result, error) {
	actionConfig := helm.ActionConfig(gw.Namespace, log.Debug().Msgf)

	resolveValues := func(object metav1.Object, mc configurator.Configurator) (map[string]interface{}, error) {
		gatewayValues, err := r.resolveGatewayValues(object, mc, update)
		if err != nil {
			return nil, err
		}

		parameterValues, err := r.resolveParameterValues(gw, update)
		if err != nil {
			log.Error().Msgf("Failed to resolve parameter values from ParametersRef: %s, it doesn't take effect", err)
			return gatewayValues, nil
		}

		if parameterValues == nil {
			return gatewayValues, nil
		}

		// gateway values take precedence over parameter values, means the values from MeshConfig override the values from ParametersRef
		// see the overrides variables for a complete list of values
		return chartutil.CoalesceTables(parameterValues, gatewayValues), nil
	}

	templateClient := helm.TemplateClient(
		actionConfig,
		fmt.Sprintf("fsm-gateway-%s-%s", gw.Namespace, gw.Name),
		gw.Namespace,
		r.kubeVersionForTemplate(),
	)
	if ctrlResult, err := helm.RenderChart(templateClient, gw, chartSource, mc, r.fctx.Client, r.fctx.Scheme, resolveValues); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Deploy", "Failed to deploy gateway: %s", err)
		return ctrlResult, err
	}

	//defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Deploy", "Deploy gateway successfully")

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) kubeVersionForTemplate() *chartutil.KubeVersion {
	if version.IsEndpointSliceEnabled(r.fctx.KubeClient) {
		return constants.KubeVersion121
	}

	return constants.KubeVersion119
}

func (r *gatewayReconciler) resolveGatewayValues(object metav1.Object, mc configurator.Configurator, update *gw.GatewayStatusUpdate) (map[string]interface{}, error) {
	gateway, ok := object.(*gwv1.Gateway)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of *gwv1.Gateway", object)
	}

	log.Debug().Msgf("[GW] Resolving Values ...")

	// these values are from MeshConfig and Gateway resource, it will not be overridden by values from ParametersRef
	gwBytes, err := ghodssyaml.Marshal(map[string]interface{}{
		"fsm": map[string]interface{}{
			"fsmNamespace": mc.GetFSMNamespace(),
			"meshName":     r.fctx.MeshName,
			"gateway": map[string]interface{}{
				"namespace":      gateway.Namespace,
				"name":           gateway.Name,
				"listeners":      r.listenersForTemplate(gateway, update),
				"infrastructure": infraForTemplate(gateway),
				"logLevel":       mc.GetFSMGatewayLogLevel(),
			},
			"image": map[string]interface{}{
				"registry":   mc.GetImageRegistry(),
				"tag":        mc.GetImageTag(),
				"pullPolicy": mc.GetImagePullPolicy(),
			},
		},
		"hasTCP": hasTCP(gateway),
		"hasUDP": hasUDP(gateway),
	})
	if err != nil {
		return nil, fmt.Errorf("convert values map to yaml, err = %v", err)
	}

	log.Debug().Msgf("\n\nGATEWAY VALUES YAML:\n\n\n%s\n\n", string(gwBytes))

	gwValues, err := chartutil.ReadValues(gwBytes)
	if err != nil {
		return nil, err
	}

	return gwValues.AsMap(), nil
}

func infraForTemplate(gateway *gwv1.Gateway) map[string]map[gwv1.AnnotationKey]gwv1.AnnotationValue {
	infra := map[string]map[gwv1.AnnotationKey]gwv1.AnnotationValue{
		"annotations": {},
		"labels":      {},
	}

	if gateway.Spec.Infrastructure != nil {
		if len(gateway.Spec.Infrastructure.Annotations) > 0 {
			infra["annotations"] = gateway.Spec.Infrastructure.Annotations
		}
		if len(gateway.Spec.Infrastructure.Labels) > 0 {
			infra["labels"] = gateway.Spec.Infrastructure.Labels
		}
	}

	return infra
}

func (r *gatewayReconciler) listenersForTemplate(gateway *gwv1.Gateway, update *gw.GatewayStatusUpdate) []listener {
	listeners := make([]listener, 0)
	for _, l := range gateway.Spec.Listeners {
		s := update.GetListenerStatus(string(l.Name))

		if s == nil {
			continue
		}

		if !gwutils.IsListenerValid(*s) {
			continue
		}

		listeners = append(listeners, listener{
			Name:     l.Name,
			Port:     l.Port,
			Protocol: l.Protocol,
		})
	}

	return listeners
}

func (r *gatewayReconciler) resolveParameterValues(gateway *gwv1.Gateway, update *gw.GatewayStatusUpdate) (map[string]interface{}, error) {
	if gateway.Spec.Infrastructure == nil {
		return nil, nil
	}

	if gateway.Spec.Infrastructure.ParametersRef == nil {
		return nil, nil
	}

	paramRef := gateway.Spec.Infrastructure.ParametersRef
	if paramRef.Group != corev1.GroupName {
		return nil, nil
	}

	if paramRef.Kind != constants.KubernetesConfigMapKind {
		return nil, nil
	}

	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{
		Namespace: gateway.Namespace,
		Name:      paramRef.Name,
	}

	if err := r.fctx.Get(context.TODO(), key, cm); err != nil {
		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionFalse,
			gwv1.GatewayReasonInvalidParameters,
			fmt.Sprintf("Failed to get ConfigMap %s: %s", key, err),
		)
		return nil, fmt.Errorf("failed to get Configmap %s: %s", key, err)
	}

	if len(cm.Data) == 0 {
		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionFalse,
			gwv1.GatewayReasonInvalidParameters,
			fmt.Sprintf("Configmap %q has no data", key),
		)
		return nil, fmt.Errorf("configmap %q has no data", key)
	}

	valuesYaml, ok := cm.Data["values.yaml"]
	if !ok {
		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionFalse,
			gwv1.GatewayReasonInvalidParameters,
			fmt.Sprintf("Configmap %q doesn't have required values.yaml", key),
		)
		return nil, fmt.Errorf("configmap %q has no values.yaml", key)
	}

	log.Debug().Msgf("[GW] values.yaml from ConfigMap %s: \n%s\n", key.String(), valuesYaml)

	paramsMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(valuesYaml), &paramsMap); err != nil {
		update.AddCondition(
			gwv1.GatewayConditionAccepted,
			metav1.ConditionFalse,
			gwv1.GatewayReasonInvalidParameters,
			fmt.Sprintf("Failed to unmarshal values.yaml of Configmap %s: %s", key, err),
		)
		return nil, fmt.Errorf("failed to unmarshal values.yaml of Configmap %s: %s", key, err)
	}

	log.Debug().Msgf("[GW] values parsed from values.yaml: %v", paramsMap)

	return paramsMap, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.Gateway{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			gateway, ok := obj.(*gwv1.Gateway)
			if !ok {
				log.Error().Msgf("unexpected object type %T", obj)
				return false
			}

			gatewayClass := &gwv1.GatewayClass{}
			key := types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}
			if err := r.fctx.Get(context.TODO(), key, gatewayClass); err != nil {
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
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.configMapToGateways),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.secretToGateways),
		).
		Complete(r); err != nil {
		return err
	}

	return addGatewayIndexers(context.TODO(), mgr)
}

func (r *gatewayReconciler) gatewayClassToGateways(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayClass, ok := obj.(*gwv1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	if gwutils.IsAcceptedGatewayClass(gatewayClass) {
		c := r.fctx.Manager.GetCache()
		gateways := &gwv1.GatewayList{}
		if err := c.List(ctx, gateways, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, gatewayClass.Name),
		}); err != nil {
			log.Error().Msgf("error listing gateways: %s", err)
			return nil
		}

		var reconciles []reconcile.Request
		for _, gwy := range gateways.Items {
			gwy := gwy
			if gwutils.IsActiveGateway(&gwy) {
				reconciles = append(reconciles, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: gwy.Namespace,
						Name:      gwy.Name,
					},
				})
			}
		}

		return reconciles
	}

	return nil
}

func (r *gatewayReconciler) configMapToGateways(ctx context.Context, object client.Object) []reconcile.Request {
	cm, ok := object.(*corev1.ConfigMap)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", object)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	gateways := &gwv1.GatewayList{}
	if err := c.List(ctx, gateways, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigMapGatewayIndex, client.ObjectKeyFromObject(cm).String()),
		Namespace:     cm.Namespace,
	}); err != nil {
		log.Error().Msgf("error listing gateways: %s", err)
		return nil
	}

	if len(gateways.Items) == 0 {
		return nil
	}

	reconciles := make([]reconcile.Request, 0)
	for _, gw := range gateways.Items {
		gw := gw
		if gwutils.IsActiveGateway(&gw) {
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

func (r *gatewayReconciler) secretToGateways(ctx context.Context, object client.Object) []reconcile.Request {
	secret, ok := object.(*corev1.Secret)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", object)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	gateways := &gwv1.GatewayList{}
	if err := c.List(ctx, gateways, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.SecretGatewayIndex, client.ObjectKeyFromObject(secret).String()),
	}); err != nil {
		log.Error().Msgf("error listing gateways: %s", err)
		return nil
	}

	if len(gateways.Items) == 0 {
		return nil
	}

	reconciles := make([]reconcile.Request, 0)
	for _, gwy := range gateways.Items {
		gwy := gwy
		if gwutils.IsActiveGateway(&gwy) {
			reconciles = append(reconciles, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gwy.Namespace,
					Name:      gwy.Name,
				},
			})
		}
	}

	return reconciles
}

func addGatewayIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.Gateway{}, constants.SecretGatewayIndex, secretGatewayIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.Gateway{}, constants.ConfigMapGatewayIndex, configMapGatewayIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.Gateway{}, constants.ClassGatewayIndex, func(obj client.Object) []string {
		gateway := obj.(*gwv1.Gateway)
		return []string{string(gateway.Spec.GatewayClassName)}
	}); err != nil {
		return err
	}

	return nil
}

func secretGatewayIndexFunc(obj client.Object) []string {
	gateway := obj.(*gwv1.Gateway)
	var secretReferences []string
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol != gwv1.TLSProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			continue
		}

		if listener.TLS == nil || *listener.TLS.Mode != gwv1.TLSModeTerminate {
			continue
		}

		for _, cert := range listener.TLS.CertificateRefs {
			if *cert.Kind == constants.KubernetesSecretKind {
				secretReferences = append(secretReferences,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(cert.Namespace, gateway.Namespace),
						Name:      string(cert.Name),
					}.String(),
				)
			}
		}

		if listener.TLS.FrontendValidation != nil {
			for _, ca := range listener.TLS.FrontendValidation.CACertificateRefs {
				if ca.Kind == constants.KubernetesSecretKind {
					secretReferences = append(secretReferences,
						types.NamespacedName{
							Namespace: gwutils.NamespaceDerefOr(ca.Namespace, gateway.Namespace),
							Name:      string(ca.Name),
						}.String(),
					)
				}
			}
		}
	}

	return secretReferences
}

func configMapGatewayIndexFunc(obj client.Object) []string {
	gateway := obj.(*gwv1.Gateway)
	var cmRefs []string

	// check against listeners
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol != gwv1.TLSProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			continue
		}

		if listener.TLS == nil || *listener.TLS.Mode != gwv1.TLSModeTerminate {
			continue
		}

		if listener.TLS.FrontendValidation == nil {
			continue
		}

		for _, ca := range listener.TLS.FrontendValidation.CACertificateRefs {
			if ca.Kind == constants.KubernetesConfigMapKind {
				cmRefs = append(cmRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(ca.Namespace, gateway.Namespace),
						Name:      string(ca.Name),
					}.String(),
				)
			}
		}
	}

	// check against infrastructure ParametersRef
	if gateway.Spec.Infrastructure != nil && gateway.Spec.Infrastructure.ParametersRef != nil {
		parametersRef := gateway.Spec.Infrastructure.ParametersRef
		if parametersRef.Kind == constants.KubernetesConfigMapKind {
			cmRefs = append(cmRefs,
				types.NamespacedName{
					Namespace: gateway.Namespace,
					Name:      string(parametersRef.Name),
				}.String(),
			)
		}
	}

	return cmRefs
}
