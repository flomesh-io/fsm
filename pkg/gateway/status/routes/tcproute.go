package routes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

func (p *RouteStatusProcessor) processTCPRouteStatus(route *gwv1alpha2.TCPRoute, parentRef gwv1.ParentReference, rps status.RouteParentStatusObject) bool {
	for _, rule := range route.Spec.Rules {
		if !p.processTCPRouteRuleBackendRefs(route, parentRef, rule, rps) {
			return false
		}

		p.computeRouteRuleFilterPolicyStatus(route, rule.Name, parentRef)
	}

	// All backend references of all rules have been resolved successfully for the parent
	p.addResolvedRefsCondition(route, rps, gwv1.RouteReasonResolvedRefs, "All backend references are resolved")

	return true
}

func (p *RouteStatusProcessor) processTCPRouteRuleBackendRefs(route *gwv1alpha2.TCPRoute, parentRef gwv1.ParentReference, rule gwv1alpha2.TCPRouteRule, rps status.RouteParentStatusObject) bool {
	for _, bk := range rule.BackendRefs {
		if !p.processTCPRouteBackend(route, parentRef, bk, rps) {
			return false
		}
	}

	return true
}

func (p *RouteStatusProcessor) processTCPRouteBackend(route *gwv1alpha2.TCPRoute, parentRef gwv1.ParentReference, bk gwv1alpha2.BackendRef, rps status.RouteParentStatusObject) bool {
	svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
	if svcPort == nil {
		return false
	}

	if svcPort.Protocol != corev1.ProtocolTCP {
		p.addNotResolvedRefsCondition(route, rps, gwv1.RouteReasonUnsupportedProtocol, fmt.Sprintf("Unsupported protocol %q for backend %s", svcPort.Protocol, svcPort.String()))
		return false
	}

	if svcPort.AppProtocol != nil {
		p.addNotResolvedRefsCondition(route, rps, gwv1.RouteReasonUnsupportedProtocol, "AppProtocol is not supported for TCPRoute backend")
		return false
	}

	p.computeBackendTLSPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef, func(found bool) {})

	return true
}
