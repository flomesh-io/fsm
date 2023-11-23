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

package status

import (
	"context"
	"fmt"
	"time"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// RouteStatusProcessor is responsible for computing the status of a Route
type RouteStatusProcessor struct {
	Informers *informers.InformerCollection
}

// ProcessRouteStatus computes the status of a Route
func (p *RouteStatusProcessor) ProcessRouteStatus(_ context.Context, route client.Object) ([]gwv1beta1.RouteParentStatus, error) {
	gatewayList := p.Informers.List(informers.InformerKeyGatewayAPIGateway)

	activeGateways := make([]*gwv1beta1.Gateway, 0)
	for _, gw := range gatewayList {
		gw := gw.(*gwv1beta1.Gateway)
		if gwutils.IsActiveGateway(gw) {
			activeGateways = append(activeGateways, gw)
		}
	}

	if len(activeGateways) > 0 {
		var params *computeParams
		switch route := route.(type) {
		case *gwv1beta1.HTTPRoute:
			params = &computeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  route.Spec.Hostnames,
				RouteNs:         route.GetNamespace(),
			}
		case *gwv1alpha2.GRPCRoute:
			params = &computeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  route.Spec.Hostnames,
				RouteNs:         route.GetNamespace(),
			}
		case *gwv1alpha2.TLSRoute:
			params = &computeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  route.Spec.Hostnames,
				RouteNs:         route.GetNamespace(),
			}
		case *gwv1alpha2.TCPRoute:
			params = &computeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  nil,
				RouteNs:         route.GetNamespace(),
			}
		case *gwv1alpha2.UDPRoute:
			params = &computeParams{
				ParentRefs:      route.Spec.ParentRefs,
				RouteGvk:        route.GroupVersionKind(),
				RouteGeneration: route.GetGeneration(),
				RouteHostnames:  nil,
				RouteNs:         route.GetNamespace(),
			}
		default:
			log.Warn().Msgf("Unsupported route type: %T", route)
			return nil, fmt.Errorf("unsupported route type: %T", route)
		}

		return p.computeRouteParentStatus(activeGateways, params), nil
	}

	return nil, nil
}

func (p *RouteStatusProcessor) computeRouteParentStatus(
	activeGateways []*gwv1beta1.Gateway,
	params *computeParams,
) []gwv1beta1.RouteParentStatus {
	status := make([]gwv1beta1.RouteParentStatus, 0)

	for _, gw := range activeGateways {
		validListeners := gwutils.GetValidListenersFromGateway(gw)

		for _, parentRef := range params.ParentRefs {
			if !gwutils.IsRefToGateway(parentRef, gwutils.ObjectKey(gw)) {
				continue
			}

			routeParentStatus := gwv1beta1.RouteParentStatus{
				ParentRef:      parentRef,
				ControllerName: constants.GatewayController,
				Conditions:     make([]metav1.Condition, 0),
			}

			allowedListeners := gwutils.GetAllowedListenersAndSetStatus(parentRef, params.RouteNs, params.RouteGvk, params.RouteGeneration, validListeners, routeParentStatus)
			//if len(allowedListeners) == 0 {
			//
			//}

			count := 0
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, params.RouteHostnames)

				//if len(hostnames) == 0 {
				//	continue
				//}

				count += len(hostnames)
			}

			switch params.RouteGvk.Kind {
			case constants.GatewayAPIHTTPRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPIGRPCRouteKind:
				if count == 0 && metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionAccepted)) == nil {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
						Type:               string(gwv1beta1.RouteConditionAccepted),
						Status:             metav1.ConditionFalse,
						ObservedGeneration: params.RouteGeneration,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1beta1.RouteReasonNoMatchingListenerHostname),
						Message:            "No matching hostnames were found between the listener and the route.",
					})
				}
			}

			if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionResolvedRefs)) == nil {
				metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
					Type:               string(gwv1beta1.RouteConditionResolvedRefs),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: params.RouteGeneration,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1beta1.RouteReasonResolvedRefs),
					Message:            fmt.Sprintf("References of %s is resolved", params.RouteGvk.Kind),
				})
			}

			if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1beta1.RouteConditionAccepted)) == nil {
				metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
					Type:               string(gwv1beta1.RouteConditionAccepted),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: params.RouteGeneration,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1beta1.RouteReasonAccepted),
					Message:            fmt.Sprintf("%s is Accepted", params.RouteGvk.Kind),
				})
			}

			status = append(status, routeParentStatus)
		}
	}

	return status
}
