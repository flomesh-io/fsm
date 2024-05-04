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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// RouteStatusProcessor is responsible for computing the status of a Route
type RouteStatusProcessor struct {
	Listers *informers.Lister
}

// ProcessRouteStatus computes the status of a Route
func (p *RouteStatusProcessor) ProcessRouteStatus(_ context.Context, route client.Object) ([]gwv1.RouteParentStatus, error) {
	gateways, err := p.Listers.Gateway.Gateways(corev1.NamespaceAll).List(labels.Set{}.AsSelector())
	if err != nil {
		return nil, err
	}

	activeGateways := make([]*gwv1.Gateway, 0)
	for _, gw := range gateways {
		if gwutils.IsActiveGateway(gw) {
			activeGateways = append(activeGateways, gw)
		}
	}

	if len(activeGateways) > 0 {
		params := gwutils.ToRouteContext(route)
		if params == nil {
			return nil, fmt.Errorf("failed to convert route to route context, unsupported route type %T", route)
		}

		return p.computeRouteParentStatus(activeGateways, params), nil
	}

	return nil, nil
}

func (p *RouteStatusProcessor) computeRouteParentStatus(
	activeGateways []*gwv1.Gateway,
	params *gwtypes.RouteContext,
) []gwv1.RouteParentStatus {
	status := make([]gwv1.RouteParentStatus, 0)

	for _, gw := range activeGateways {
		validListeners := gwutils.GetValidListenersForGateway(gw)

		for _, parentRef := range params.ParentRefs {
			if !gwutils.IsRefToGateway(parentRef, gwutils.ObjectKey(gw)) {
				continue
			}

			routeParentStatus := gwv1.RouteParentStatus{
				ParentRef:      parentRef,
				ControllerName: constants.GatewayController,
				Conditions:     make([]metav1.Condition, 0),
			}

			//allowedListeners := p.getAllowedListenersAndSetStatus(gw, parentRef, params, validListeners, routeParentStatus)
			allowedListeners, conditions := gwutils.GetAllowedListeners(p.Listers.Namespace, gw, parentRef, params, validListeners)
			//if len(allowedListeners) == 0 {
			//
			//}
			if len(conditions) > 0 {
				for _, condition := range conditions {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, condition)
				}
			}

			count := 0
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, params.Hostnames)

				//if len(hostnames) == 0 {
				//	continue
				//}

				count += len(hostnames)
			}

			switch params.GVK.Kind {
			case constants.GatewayAPIHTTPRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPIGRPCRouteKind:
				if count == 0 && metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1.RouteConditionAccepted)) == nil {
					metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
						Type:               string(gwv1.RouteConditionAccepted),
						Status:             metav1.ConditionFalse,
						ObservedGeneration: params.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1.RouteReasonNoMatchingListenerHostname),
						Message:            "No matching hostnames were found between the listener and the route.",
					})
				}
			}

			if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1.RouteConditionResolvedRefs)) == nil {
				metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
					Type:               string(gwv1.RouteConditionResolvedRefs),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: params.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1.RouteReasonResolvedRefs),
					Message:            fmt.Sprintf("References of %s is resolved", params.GVK.Kind),
				})
			}

			if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1.RouteConditionAccepted)) == nil {
				metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
					Type:               string(gwv1.RouteConditionAccepted),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: params.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1.RouteReasonAccepted),
					Message:            fmt.Sprintf("%s is Accepted", params.GVK.Kind),
				})
			}

			status = append(status, routeParentStatus)
		}
	}

	return status
}
