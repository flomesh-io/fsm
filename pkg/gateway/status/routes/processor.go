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
	"strings"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"github.com/flomesh-io/fsm/pkg/gateway/status/policies"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rs/zerolog/log"

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
	statusUpdater status.Updater
}

func NewRouteStatusProcessor(cache cache.Cache, statusUpdater status.Updater) *RouteStatusProcessor {
	return &RouteStatusProcessor{
		client:        cache,
		statusUpdater: statusUpdater,
	}
}

// Process computes the status of a Route
func (p *RouteStatusProcessor) Process(_ context.Context, update status.RouteStatusObject, parentRefs []gwv1.ParentReference) error {
	//if activeGateways := gwutils.GetActiveGateways(p.client); len(activeGateways) > 0 {
	//	p.computeRouteParentStatus(activeGateways, update, parentRefs)
	//
	//	updater.Send(status.Update{
	//		Resource:       update.GetResource(),
	//		NamespacedName: update.GetFullName(),
	//		Mutator:        update,
	//	})
	//}

	p.computeRouteParentStatus(update, parentRefs)

	p.statusUpdater.Send(status.Update{
		Resource:       update.GetResource(),
		NamespacedName: update.GetFullName(),
		Mutator:        update,
	})

	return nil
}

//gocyclo:ignore
func (p *RouteStatusProcessor) computeRouteParentStatus(rs status.RouteStatusObject, parentRefs []gwv1.ParentReference) {
	for _, parentRef := range parentRefs {
		rps := rs.StatusUpdateFor(parentRef)

		gvk := rs.GetTypeMeta().GroupVersionKind()
		rps.AddCondition(
			gwv1.RouteConditionAccepted,
			metav1.ConditionTrue,
			gwv1.RouteReasonAccepted,
			fmt.Sprintf("%s is Accepted", gvk.Kind),
		)

		if parentRef.Group != nil && *parentRef.Group != gwv1.GroupName {
			rps.AddCondition(
				gwv1.RouteConditionAccepted,
				metav1.ConditionFalse,
				gwv1.RouteReasonUnsupportedValue,
				fmt.Sprintf("Group %q is not supported as parent of xRoute", *parentRef.Group),
			)
			continue
		}

		if parentRef.Kind != nil && *parentRef.Kind != constants.GatewayAPIGatewayKind {
			rps.AddCondition(
				gwv1.RouteConditionAccepted,
				metav1.ConditionFalse,
				gwv1.RouteReasonUnsupportedValue,
				fmt.Sprintf("Kind %q is not supported as parent of xRoute", *parentRef.Kind),
			)
			continue
		}

		parentKey := types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(parentRef.Namespace, rs.GetFullName().Namespace),
			Name:      string(parentRef.Name),
		}
		parent := &gwv1.Gateway{}
		if err := p.client.Get(context.Background(), parentKey, parent); err != nil {
			if errors.IsNotFound(err) {
				rps.AddCondition(
					gwv1.RouteConditionAccepted,
					metav1.ConditionFalse,
					gwv1.RouteReasonNoMatchingParent,
					fmt.Sprintf("Parent %s not found", parentKey),
				)
			} else {
				rps.AddCondition(
					gwv1.RouteConditionAccepted,
					metav1.ConditionFalse,
					gwv1.RouteReasonNoMatchingParent,
					fmt.Sprintf("Failed to get Parent %s: %s", parentKey, err),
				)
			}
			continue
		}

		if !gwutils.IsActiveGateway(parent) {
			rps.AddCondition(
				gwv1.RouteConditionAccepted,
				metav1.ConditionFalse,
				gwv1.RouteReasonNoMatchingParent,
				fmt.Sprintf("Parent %s is not accepted or programmed", parentKey),
			)
			continue
		}

		allowedListeners := gwutils.GetAllowedListeners(p.client, parent, rps)
		if len(allowedListeners) == 0 {
			continue
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
				rps.AddCondition(
					gwv1.RouteConditionAccepted,
					metav1.ConditionFalse,
					gwv1.RouteReasonNoMatchingListenerHostname,
					"No matching hostnames were found between the listener and the route.",
				)
				continue
			}
		}

		switch route := rs.GetResource().(type) {
		case *gwv1.HTTPRoute:
			if !p.processHTTPRouteStatus(route, rps, parentRef) {
				continue
			}
		case *gwv1.GRPCRoute:
			if !p.processGRPCRouteStatus(route, rps, parentRef) {
				continue
			}
		case *gwv1alpha2.TLSRoute:
			//for _, rule := range route.Spec.Rules {
			//
			//}
		case *gwv1alpha2.TCPRoute:
			if !p.processTCPRouteStatus(route, rps, parentRef) {
				continue
			}
		case *gwv1alpha2.UDPRoute:
			if !p.processUDPRouteStatus(route, rps) {
				continue
			}
		default:
			continue
		}
	}
}

func (p *RouteStatusProcessor) processUDPRouteStatus(route *gwv1alpha2.UDPRoute, rps status.RouteParentStatusObject) bool {
	for _, rule := range route.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
			if svcPort == nil {
				return false
			}

			log.Debug().Msgf("BackendRef: %v, svcPort: %s", bk.BackendObjectReference, svcPort.String())
		}
	}

	return true
}

func (p *RouteStatusProcessor) processTCPRouteStatus(route *gwv1alpha2.TCPRoute, rps status.RouteParentStatusObject, parentRef gwv1.ParentReference) bool {
	for _, rule := range route.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
			if svcPort == nil {
				return false
			}

			log.Debug().Msgf("BackendRef: %v, svcPort: %s", bk.BackendObjectReference, svcPort.String())
			p.computeBackendTLSPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
		}
	}

	return true
}

func (p *RouteStatusProcessor) processGRPCRouteStatus(route *gwv1.GRPCRoute, rps status.RouteParentStatusObject, parentRef gwv1.ParentReference) bool {
	for _, rule := range route.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
			if svcPort == nil {
				return false
			}

			log.Debug().Msgf("BackendRef: %v, svcPort: %s", bk.BackendObjectReference, svcPort.String())
			p.computeBackendTLSPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
			p.computeBackendLBPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
		}
	}

	return true
}

func (p *RouteStatusProcessor) processHTTPRouteStatus(route *gwv1.HTTPRoute, rps status.RouteParentStatusObject, parentRef gwv1.ParentReference) bool {
	for _, rule := range route.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
			if svcPort == nil {
				return false
			}

			log.Debug().Msgf("BackendRef: %v, svcPort: %s", bk.BackendObjectReference, svcPort.String())
			p.computeBackendTLSPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
			p.computeBackendLBPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
			p.computeHealthCheckPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
			p.computeRetryPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)
		}
	}

	return true
}

func (p *RouteStatusProcessor) computeBackendTLSPolicyStatus(route client.Object, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName, routeParentRef gwv1.ParentReference) {
	targetRef := gwv1alpha2.LocalPolicyTargetReferenceWithSectionName{
		LocalPolicyTargetReference: gwv1alpha2.LocalPolicyTargetReference{
			Group: ptr.Deref(backendRef.Group, corev1.GroupName),
			Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
			Name:  backendRef.Name,
		},
		SectionName: ptr.To(gwv1alpha2.SectionName(svcPort.SectionName)),
	}

	policy, found := gwutils.FindBackendTLSPolicy(p.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	psu := policies.NewPolicyStatusUpdateWithLocalPolicyTargetReferenceWithSectionName(
		policy,
		&policy.ObjectMeta,
		&policy.TypeMeta,
		policy.Spec.TargetRefs,
		gwutils.ToSlicePtr(policy.Status.Ancestors),
	)

	ancestorStatus := psu.StatusUpdateFor(routeParentRef)

	defer func() {
		p.statusUpdater.Send(status.Update{
			Resource:       psu.GetResource(),
			NamespacedName: psu.GetFullName(),
			Mutator:        psu,
		})
	}()

	if policy.Spec.Validation.WellKnownCACertificates != nil && *policy.Spec.Validation.WellKnownCACertificates != "" {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			".spec.validation.wellKnownCACertificates is unsupported.",
		)
		return
	}

	hostname := string(policy.Spec.Validation.Hostname)
	if err := gwutils.IsValidHostname(hostname); err != nil {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf(".spec.validation.hostname %q is invalid. Hostname must be a valid RFC 1123 fully qualified domain name.", hostname),
		)

		return
	}

	if strings.Contains(hostname, "*") {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionFalse,
			gwv1alpha2.PolicyReasonInvalid,
			fmt.Sprintf(".spec.validation.hostname %q is invalid. Wildcard domains and numeric IP addresses are not allowed", hostname),
		)

		return
	}

	for _, ref := range policy.Spec.Validation.CACertificateRefs {
		ref := gwv1.ObjectReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Namespace: ptr.To(gwv1.Namespace(policy.Namespace)),
			Name:      ref.Name,
		}
		if ca := gwutils.ObjectRefToCACertificate(p.client, policy, ref, ancestorStatus); len(ca) == 0 {
			log.Error().Msgf("Failed to get CA certificate %s", ref.Name)
		}
	}

	if !ancestorStatus.ConditionExists(gwv1alpha2.PolicyConditionAccepted) {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionTrue,
			gwv1alpha2.PolicyReasonAccepted,
			fmt.Sprintf("Policy is accepted for ancestor %s/%s", gwutils.NamespaceDerefOr(routeParentRef.Namespace, route.GetNamespace()), routeParentRef.Name),
		)
	}
}

func (p *RouteStatusProcessor) computeBackendLBPolicyStatus(route client.Object, backendRef gwv1.BackendObjectReference, _ *fgwv2.ServicePortName, routeParentRef gwv1.ParentReference) {
	targetRef := gwv1alpha2.LocalPolicyTargetReference{
		Group: ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Name:  backendRef.Name,
	}

	policy, found := gwutils.FindBackendLBPolicy(p.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	psu := policies.NewPolicyStatusUpdateWithLocalPolicyTargetReference(
		policy,
		&policy.ObjectMeta,
		&policy.TypeMeta,
		policy.Spec.TargetRefs,
		gwutils.ToSlicePtr(policy.Status.Ancestors),
	)

	ancestorStatus := psu.StatusUpdateFor(routeParentRef)
	defer func() {
		p.statusUpdater.Send(status.Update{
			Resource:       psu.GetResource(),
			NamespacedName: psu.GetFullName(),
			Mutator:        psu,
		})
	}()

	if !ancestorStatus.ConditionExists(gwv1alpha2.PolicyConditionAccepted) {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionTrue,
			gwv1alpha2.PolicyReasonAccepted,
			fmt.Sprintf("Policy is accepted for ancestor %s/%s", gwutils.NamespaceDerefOr(routeParentRef.Namespace, route.GetNamespace()), routeParentRef.Name),
		)
	}
}

func (p *RouteStatusProcessor) computeHealthCheckPolicyStatus(route client.Object, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName, routeParentRef gwv1.ParentReference) {
	targetRef := gwv1alpha2.NamespacedPolicyTargetReference{
		Group:     ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:      ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Namespace: backendRef.Namespace,
		Name:      backendRef.Name,
	}

	policy, _, found := gwutils.FindHealthCheckPolicy(p.client, targetRef, route.GetNamespace(), svcPort)
	if !found {
		return
	}

	psu := policies.NewPolicyStatusUpdateWithNamespacedPolicyTargetReference(
		policy,
		&policy.ObjectMeta,
		&policy.TypeMeta,
		policy.Spec.TargetRefs,
		gwutils.ToSlicePtr(policy.Status.Ancestors),
	)

	ancestorStatus := psu.StatusUpdateFor(routeParentRef)
	defer func() {
		p.statusUpdater.Send(status.Update{
			Resource:       psu.GetResource(),
			NamespacedName: psu.GetFullName(),
			Mutator:        psu,
		})
	}()

	if gwutils.HasAccessToBackendTargetRef(p.client, policy, targetRef, ancestorStatus) {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionTrue,
			gwv1alpha2.PolicyReasonAccepted,
			fmt.Sprintf("Policy is accepted for ancestor %s/%s", gwutils.NamespaceDerefOr(routeParentRef.Namespace, route.GetNamespace()), routeParentRef.Name),
		)
	}
}

func (p *RouteStatusProcessor) computeRetryPolicyStatus(route client.Object, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName, routeParentRef gwv1.ParentReference) {
	targetRef := gwv1alpha2.NamespacedPolicyTargetReference{
		Group:     ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:      ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Namespace: backendRef.Namespace,
		Name:      backendRef.Name,
	}

	policy, _, found := gwutils.FindRetryPolicy(p.client, targetRef, route.GetNamespace(), svcPort)
	if !found {
		return
	}

	psu := policies.NewPolicyStatusUpdateWithNamespacedPolicyTargetReference(
		policy,
		&policy.ObjectMeta,
		&policy.TypeMeta,
		policy.Spec.TargetRefs,
		gwutils.ToSlicePtr(policy.Status.Ancestors),
	)

	ancestorStatus := psu.StatusUpdateFor(routeParentRef)
	defer func() {
		p.statusUpdater.Send(status.Update{
			Resource:       psu.GetResource(),
			NamespacedName: psu.GetFullName(),
			Mutator:        psu,
		})
	}()

	if gwutils.HasAccessToBackendTargetRef(p.client, policy, targetRef, ancestorStatus) {
		ancestorStatus.AddCondition(
			gwv1alpha2.PolicyConditionAccepted,
			metav1.ConditionTrue,
			gwv1alpha2.PolicyReasonAccepted,
			fmt.Sprintf("Policy is accepted for ancestor %s/%s", gwutils.NamespaceDerefOr(routeParentRef.Namespace, route.GetNamespace()), routeParentRef.Name),
		)
	}
}

func (p *RouteStatusProcessor) backendRefToServicePortName(route client.Object, backendRef gwv1.BackendObjectReference, rps status.RouteParentStatusObject) *fgwv2.ServicePortName {
	return gwutils.BackendRefToServicePortName(p.client, route, backendRef, rps)
}
