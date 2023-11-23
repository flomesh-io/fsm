package cache

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func processUDPRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, udpRoute *gwv1alpha2.UDPRoute, rules map[int32]routecfg.RouteRule) {
	for _, ref := range udpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, udpRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1beta1.UDPProtocolType:
				rules[int32(listener.Port)] = generateUDPRouteCfg(udpRoute)
			}
		}
	}
}

func processUDPBackends(udpRoute *gwv1alpha2.UDPRoute, services map[string]serviceInfo) {
	for _, rule := range udpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(backend.BackendObjectReference, udpRoute.Namespace); svcPort != nil {
				services[svcPort.String()] = serviceInfo{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func generateUDPRouteCfg(udpRoute *gwv1alpha2.UDPRoute) routecfg.RouteRule {
	backends := routecfg.UDPRouteRule{}

	for _, rule := range udpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendObjectReference, udpRoute.Namespace); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}
