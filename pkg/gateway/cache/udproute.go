package cache

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) processUDPRoute(gw *gwv1.Gateway, validListeners []gwtypes.Listener, udpRoute *gwv1alpha2.UDPRoute, rules map[int32]fgw.RouteRule) {
	for _, ref := range udpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.informers.GetListers().Namespace, gw, ref, gwutils.ToRouteContext(udpRoute), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1.UDPProtocolType:
				rules[int32(listener.Port)] = c.generateUDPRouteCfg(udpRoute)
			}
		}
	}
}

func (c *GatewayCache) processUDPBackends(udpRoute *gwv1alpha2.UDPRoute, services map[string]serviceContext) {
	for _, rule := range udpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(udpRoute, backend.BackendObjectReference); svcPort != nil {
				services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func (c *GatewayCache) generateUDPRouteCfg(udpRoute *gwv1alpha2.UDPRoute) fgw.RouteRule {
	backends := fgw.UDPRouteRule{}

	for _, rule := range udpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(udpRoute, bk.BackendObjectReference); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}
