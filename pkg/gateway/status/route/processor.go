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
func (p *RouteStatusProcessor) Process(_ context.Context, updater status.Updater, update status.RouteStatusObject, parentRefs []gwv1.ParentReference) error {
	if activeGateways := gwutils.GetActiveGateways(p.client); len(activeGateways) > 0 {
		p.computeRouteParentStatus(activeGateways, update, parentRefs)

		updater.Send(status.Update{
			Resource:       update.GetResource(),
			NamespacedName: update.GetFullName(),
			Mutator:        update,
		})
	}

	return nil
}

func (p *RouteStatusProcessor) computeRouteParentStatus(activeGateways []*gwv1.Gateway, update status.RouteStatusObject, parentRefs []gwv1.ParentReference) {
	for _, gw := range activeGateways {
		for _, parentRef := range parentRefs {
			if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(gw)) {
				continue
			}

			u := update.StatusUpdateFor(parentRef)

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
				if count == 0 && !u.ConditionExists(gwv1.RouteConditionAccepted) {
					u.AddCondition(
						gwv1.RouteConditionAccepted,
						metav1.ConditionFalse,
						gwv1.RouteReasonNoMatchingListenerHostname,
						"No matching hostnames were found between the listener and the route.",
					)
				}
			}

			if !u.ConditionExists(gwv1.RouteConditionResolvedRefs) {
				u.AddCondition(
					gwv1.RouteConditionResolvedRefs,
					metav1.ConditionTrue,
					gwv1.RouteReasonResolvedRefs,
					fmt.Sprintf("References of %s is resolved", update.GetTypeMeta().GroupVersionKind().Kind),
				)
			}

			if !u.ConditionExists(gwv1.RouteConditionAccepted) {
				u.AddCondition(
					gwv1.RouteConditionAccepted,
					metav1.ConditionTrue,
					gwv1.RouteReasonAccepted,
					fmt.Sprintf("%s is Accepted", update.GetTypeMeta().GroupVersionKind().Kind),
				)
			}
		}
	}
}
