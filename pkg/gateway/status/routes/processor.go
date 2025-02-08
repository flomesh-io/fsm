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

package routes

import (
	"context"
	"fmt"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/tools/record"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// RouteStatusProcessor is responsible for computing the status of a Route
type RouteStatusProcessor struct {
	client        cache.Cache
	recorder      record.EventRecorder
	statusUpdater status.Updater
}

func NewRouteStatusProcessor(cache cache.Cache, recorder record.EventRecorder, statusUpdater status.Updater) *RouteStatusProcessor {
	return &RouteStatusProcessor{
		client:        cache,
		recorder:      recorder,
		statusUpdater: statusUpdater,
	}
}

// Process computes the status of a Route
func (p *RouteStatusProcessor) Process(_ context.Context, rs status.RouteStatusObject, parentRefs []gwv1.ParentReference) error {
	//if activeGateways := gwutils.GetActiveGateways(p.client); len(activeGateways) > 0 {
	//	p.computeRouteParentStatus(activeGateways, rs, parentRefs)
	//
	//	updater.Send(status.Update{
	//		Resource:       rs.GetResource(),
	//		NamespacedName: rs.GetFullName(),
	//		Mutator:        rs,
	//	})
	//}

	defer func() {
		p.statusUpdater.Send(status.Update{
			Resource:       rs.GetResource(),
			NamespacedName: rs.GetFullName(),
			Mutator:        rs,
		})
	}()

	for _, parentRef := range parentRefs {
		p.computeRouteParentStatus(rs, parentRef)
	}

	return nil
}

//gocyclo:ignore
func (p *RouteStatusProcessor) computeRouteParentStatus(rs status.RouteStatusObject, parentRef gwv1.ParentReference) {
	rps := rs.StatusUpdateFor(parentRef)

	gvk := rs.GroupVersionKind()
	rps.AddCondition(
		gwv1.RouteConditionAccepted,
		metav1.ConditionTrue,
		gwv1.RouteReasonAccepted,
		fmt.Sprintf("%s is accepted", gvk.Kind),
	)

	defer func() {
		if metautil.IsStatusConditionTrue(rps.GetRouteStatusObject().ConditionsForParentRef(parentRef), string(gwv1.RouteConditionAccepted)) {
			defer p.recorder.Eventf(rs.GetResource(), corev1.EventTypeNormal, string(gwv1.RouteReasonAccepted), "%s is accepted", gvk.Kind)
		}
	}()

	if parentRef.Group != nil && *parentRef.Group != gwv1.GroupName {
		p.addNotAcceptedCondition(rs.GetResource(), rps, gwv1.RouteReasonUnsupportedValue, fmt.Sprintf("Group %q is not supported as parent of Route", *parentRef.Group))

		return
	}

	if parentRef.Kind != nil && *parentRef.Kind != constants.GatewayAPIGatewayKind {
		p.addNotAcceptedCondition(rs.GetResource(), rps, gwv1.RouteReasonUnsupportedValue, fmt.Sprintf("Kind %q is not supported as parent of Route", *parentRef.Kind))

		return
	}

	parentKey := types.NamespacedName{
		Namespace: gwutils.NamespaceDerefOr(parentRef.Namespace, rs.GetFullName().Namespace),
		Name:      string(parentRef.Name),
	}
	parent := &gwv1.Gateway{}
	if err := p.client.Get(context.Background(), parentKey, parent); err != nil {
		if errors.IsNotFound(err) {
			p.addNotAcceptedCondition(rs.GetResource(), rps, gwv1.RouteReasonNoMatchingParent, fmt.Sprintf("Parent %s not found", parentKey))
		} else {
			p.addNotAcceptedCondition(rs.GetResource(), rps, gwv1.RouteReasonNoMatchingParent, fmt.Sprintf("Failed to get Parent %s: %s", parentKey, err))
		}

		return
	}

	if !gwutils.IsActiveGateway(parent) {
		p.addNotAcceptedCondition(rs.GetResource(), rps, gwv1.RouteReasonNoMatchingParent, fmt.Sprintf("Parent %s is not accepted or programmed", parentKey))
		return
	}

	resolver := gwutils.NewGatewayListenerResolver(NewRouteParentListenerConditionProvider(rps, p.recorder), p.client, rps)
	allowedListeners := resolver.GetAllowedListeners(parent)
	if len(allowedListeners) == 0 {
		return
	}

	count := 0
	for _, listener := range allowedListeners {
		hostnames := gwutils.GetValidHostnames(listener.Hostname, rs.GetHostnames())

		//if len(hostnames) == 0 {
		//	continue
		//}

		count += len(hostnames)
	}

	switch gvk.Kind {
	case constants.GatewayAPIHTTPRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPIGRPCRouteKind:
		if count == 0 && !rps.ConditionExists(gwv1.RouteConditionAccepted) {
			p.addNotAcceptedCondition(rs.GetResource(), rps, gwv1.RouteReasonNoMatchingListenerHostname, "No matching hostnames were found between the listener and the route.")
			return
		}
	}

	switch route := rs.GetResource().(type) {
	case *gwv1.HTTPRoute:
		if !p.processHTTPRouteStatus(route, parentRef, rps) {
			return
		}
	case *gwv1.GRPCRoute:
		if !p.processGRPCRouteStatus(route, parentRef, rps) {
			return
		}
	case *gwv1alpha2.TLSRoute:
		//for _, rule := range route.Spec.Rules {
		//
		//}
	case *gwv1alpha2.TCPRoute:
		if !p.processTCPRouteStatus(route, parentRef, rps) {
			return
		}
	case *gwv1alpha2.UDPRoute:
		if !p.processUDPRouteStatus(route, parentRef, rps) {
			return
		}
	default:
		return
	}
}

func (p *RouteStatusProcessor) backendRefToServicePortName(route client.Object, backendRef gwv1.BackendObjectReference, rps status.RouteConditionAccessor) *fgwv2.ServicePortName {
	return gwutils.BackendRefToServicePortName(p.client, route, backendRef, func(reason gwv1.RouteConditionReason, message string) {
		p.addNotResolvedRefsCondition(route, rps, reason, message)
	})
}
