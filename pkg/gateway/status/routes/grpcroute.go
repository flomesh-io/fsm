package routes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

func (p *RouteStatusProcessor) processGRPCRouteStatus(route *gwv1.GRPCRoute, parentRef gwv1.ParentReference, rps status.RouteParentStatusObject) bool {
	for _, rule := range route.Spec.Rules {
		if !p.processGRPCRouteRuleBackendRefs(route, rps, parentRef, rule.BackendRefs) {
			return false
		}
	}

	// All backend references of all rules have been resolved successfully for the parent
	p.addResolvedRefsCondition(route, rps, gwv1.RouteReasonResolvedRefs, "All backend references are resolved")

	return true
}

func (p *RouteStatusProcessor) processGRPCRouteRuleBackendRefs(route *gwv1.GRPCRoute, rps status.RouteParentStatusObject, parentRef gwv1.ParentReference, backendRefs []gwv1.GRPCBackendRef) bool {
	for _, bk := range backendRefs {
		if !p.processGRPCRouteBackend(route, parentRef, bk, rps) {
			return false
		}
	}

	return true
}

func (p *RouteStatusProcessor) processGRPCRouteBackend(route *gwv1.GRPCRoute, parentRef gwv1.ParentReference, bk gwv1.GRPCBackendRef, rps status.RouteParentStatusObject) bool {
	svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
	if svcPort == nil {
		return false
	}

	if svcPort.AppProtocol != nil {
		switch *svcPort.AppProtocol {
		case constants.AppProtocolH2C:
			log.Debug().Msgf("Backend Protocol: %q for service port %q", *svcPort.AppProtocol, svcPort.String())
			if svcPort.Protocol != corev1.ProtocolTCP {
				p.addNotResolvedRefsCondition(route, rps, gwv1.RouteReasonUnsupportedProtocol, fmt.Sprintf("Unsupported AppProtocol %q for protocol %q", *svcPort.AppProtocol, svcPort.Protocol))
				return false
			}
		default:
			p.addNotResolvedRefsCondition(route, rps, gwv1.RouteReasonUnsupportedProtocol, fmt.Sprintf("Unsupported AppProtocol %q", *svcPort.AppProtocol))
			return false
		}
	}

	p.computeBackendTLSPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef, func(found bool) {})
	p.computeBackendLBPolicyStatus(route, bk.BackendObjectReference, svcPort, parentRef)

	return true
}
