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

package route

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// RouteStatusProcessor is responsible for computing the status of a Route
type RouteStatusProcessor struct {
	client cache.Cache
}

func NewRouteStatusProcessor(cache cache.Cache) *RouteStatusProcessor {
	return &RouteStatusProcessor{
		client: cache,
	}
}

// Process computes the status of a Route
func (p *RouteStatusProcessor) Process(ctx context.Context, update status.RouteStatusObject, parentRefs []gwv1.ParentReference) error {
	class, err := gwutils.FindEffectiveGatewayClass(p.client)
	if err != nil {
		log.Error().Msgf("Failed to find GatewayClass: %v", err)
		return err
	}

	list := &gwv1.GatewayList{}
	if err := p.client.List(ctx, list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, class.Name),
	}); err != nil {
		log.Error().Msgf("Failed to list Gateways: %v", err)
		return err
	}

	activeGateways := gwutils.GetActiveGateways(gwutils.ToSlicePtr(list.Items))

	if len(activeGateways) > 0 {
		//params := gwutils.ToRouteContext(route)
		//if params == nil {
		//	return nil, fmt.Errorf("failed to convert route to route context, unsupported route type %T", route)
		//}

		//return p.computeRouteParentStatus(activeGateways, params, update), nil
		p.computeRouteParentStatus(activeGateways, update, parentRefs)
	}

	return nil
}

func (p *RouteStatusProcessor) computeRouteParentStatus(activeGateways []*gwv1.Gateway, update status.RouteStatusObject, parentRefs []gwv1.ParentReference) {
	//status := make([]gwv1.RouteParentStatus, 0)

	for _, gw := range activeGateways {
		//validListeners := gwutils.GetValidListenersForGateway(gw)

		for _, parentRef := range parentRefs {
			if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(gw)) {
				continue
			}

			u := update.StatusUpdateFor(parentRef)

			//routeParentStatus := gwv1.RouteParentStatus{
			//	ParentRef:      parentRef,
			//	ControllerName: constants.GatewayController,
			//	Conditions:     make([]metav1.Condition, 0),
			//}

			//allowedListeners := p.getAllowedListenersAndSetStatus(gw, parentRef, params, validListeners, routeParentStatus)
			//allowedListeners, conditions := gwutils.GetAllowedListeners(p.client, gw, parentRef, params, validListeners, u)
			//if len(allowedListeners) == 0 {
			//
			//}
			//if len(conditions) > 0 {
			//	for _, condition := range conditions {
			//		metautil.SetStatusCondition(&routeParentStatus.Conditions, condition)
			//	}
			//}

			allowedListeners := gwutils.GetAllowedListeners(p.client, gw, u)

			count := 0
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, update.GetHostnames())

				//if len(hostnames) == 0 {
				//	continue
				//}

				count += len(hostnames)
			}

			switch update.GetTypeMeta().GroupVersionKind().Kind {
			case constants.GatewayAPIHTTPRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPIGRPCRouteKind:
				//if count == 0 && metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1.RouteConditionAccepted)) == nil {
				//	metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
				//		Type:               string(gwv1.RouteConditionAccepted),
				//		Status:             metav1.ConditionFalse,
				//		ObservedGeneration: params.Generation,
				//		LastTransitionTime: metav1.Time{Time: time.Now()},
				//		Reason:             string(gwv1.RouteReasonNoMatchingListenerHostname),
				//		Message:            "No matching hostnames were found between the listener and the route.",
				//	})
				//}
				if count == 0 && !u.ConditionExists(gwv1.RouteConditionAccepted) {
					u.AddCondition(
						gwv1.RouteConditionAccepted,
						metav1.ConditionFalse,
						gwv1.RouteReasonNoMatchingListenerHostname,
						"No matching hostnames were found between the listener and the route.",
					)
				}
			}

			//if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1.RouteConditionResolvedRefs)) == nil {
			//	metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
			//		Type:               string(gwv1.RouteConditionResolvedRefs),
			//		Status:             metav1.ConditionTrue,
			//		ObservedGeneration: params.Generation,
			//		LastTransitionTime: metav1.Time{Time: time.Now()},
			//		Reason:             string(gwv1.RouteReasonResolvedRefs),
			//		Message:            fmt.Sprintf("References of %s is resolved", params.GVK.Kind),
			//	})
			//}

			if !u.ConditionExists(gwv1.RouteConditionResolvedRefs) {
				u.AddCondition(
					gwv1.RouteConditionResolvedRefs,
					metav1.ConditionTrue,
					gwv1.RouteReasonResolvedRefs,
					fmt.Sprintf("References of %s is resolved", update.GetTypeMeta().GroupVersionKind().Kind),
				)
			}

			//if metautil.FindStatusCondition(routeParentStatus.Conditions, string(gwv1.RouteConditionAccepted)) == nil {
			//	metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
			//		Type:               string(gwv1.RouteConditionAccepted),
			//		Status:             metav1.ConditionTrue,
			//		ObservedGeneration: params.Generation,
			//		LastTransitionTime: metav1.Time{Time: time.Now()},
			//		Reason:             string(gwv1.RouteReasonAccepted),
			//		Message:            fmt.Sprintf("%s is Accepted", params.GVK.Kind),
			//	})
			//}

			if !u.ConditionExists(gwv1.RouteConditionAccepted) {
				u.AddCondition(
					gwv1.RouteConditionAccepted,
					metav1.ConditionTrue,
					gwv1.RouteReasonAccepted,
					fmt.Sprintf("%s is Accepted", update.GetTypeMeta().GroupVersionKind().Kind),
				)
			}

			//status = append(status, routeParentStatus)
		}
	}

	//return status
}
