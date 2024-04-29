package cache

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) processTCPRoute(gw *gwv1.Gateway, validListeners []gwtypes.Listener, tcpRoute *gwv1alpha2.TCPRoute, rules map[int32]fgw.RouteRule) {
	for _, ref := range tcpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.informers.GetListers().Namespace, gw, ref, gwutils.ToRouteContext(tcpRoute), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1.TLSProtocolType:
				if listener.TLS == nil {
					continue
				}

				if listener.TLS.Mode == nil {
					continue
				}

				if *listener.TLS.Mode != gwv1.TLSModeTerminate {
					continue
				}

				hostnames := gwutils.GetValidHostnames(listener.Hostname, nil)

				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}

				tlsRule := fgw.TLSTerminateRouteRule{}
				for _, hostname := range hostnames {
					tlsRule[hostname] = c.generateTLSTerminateRouteCfg(tcpRoute)
				}

				rules[int32(listener.Port)] = tlsRule
			case gwv1.TCPProtocolType:
				rules[int32(listener.Port)] = c.generateTCPRouteCfg(tcpRoute)
			}
		}
	}
}

func (c *GatewayCache) processTCPBackends(tcpRoute *gwv1alpha2.TCPRoute, services map[string]serviceContext) {
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(tcpRoute, backend.BackendObjectReference); svcPort != nil {
				services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func (c *GatewayCache) generateTCPRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) fgw.RouteRule {
	backends := fgw.TCPRouteRule{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(tcpRoute, bk.BackendObjectReference); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}
