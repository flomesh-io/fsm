package routes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

func (p *RouteStatusProcessor) processUDPRouteStatus(route *gwv1alpha2.UDPRoute, rps status.RouteParentStatusObject) bool {
	for _, rule := range route.Spec.Rules {
		if !p.processUDPRouteRuleBackendRefs(route, rule.BackendRefs, rps) {
			return false
		}
	}

	// All backend references of all rules have been resolved successfully for the parent
	rps.AddCondition(
		gwv1.RouteConditionResolvedRefs,
		metav1.ConditionTrue,
		gwv1.RouteReasonResolvedRefs,
		"All backend references are resolved",
	)

	return true
}

func (p *RouteStatusProcessor) processUDPRouteRuleBackendRefs(route *gwv1alpha2.UDPRoute, backendRefs []gwv1alpha2.BackendRef, rps status.RouteParentStatusObject) bool {
	for _, bk := range backendRefs {
		if !p.processUDPRouteBackend(route, bk, rps) {
			return false
		}
	}

	return true
}

func (p *RouteStatusProcessor) processUDPRouteBackend(route *gwv1alpha2.UDPRoute, bk gwv1alpha2.BackendRef, rps status.RouteParentStatusObject) bool {
	svcPort := p.backendRefToServicePortName(route, bk.BackendObjectReference, rps)
	if svcPort == nil {
		return false
	}

	if svcPort.AppProtocol != nil {
		rps.AddCondition(
			gwv1.RouteConditionResolvedRefs,
			metav1.ConditionFalse,
			gwv1.RouteReasonUnsupportedProtocol,
			"AppProtocol is not supported for UDPRoute backend",
		)

		return false
	}

	log.Debug().Msgf("BackendRef: %v, svcPort: %s", bk.BackendObjectReference, svcPort.String())

	return true
}