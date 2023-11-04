package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func processTCPRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, tcpRoute *gwv1alpha2.TCPRoute, rules map[int32]routecfg.RouteRule) {
	for _, ref := range tcpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, tcpRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1beta1.TLSProtocolType:
				if listener.TLS == nil {
					continue
				}

				if listener.TLS.Mode == nil {
					continue
				}

				if *listener.TLS.Mode != gwv1beta1.TLSModeTerminate {
					continue
				}

				hostnames := gwutils.GetValidHostnames(listener.Hostname, nil)

				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}

				tlsRule := routecfg.TLSTerminateRouteRule{}
				for _, hostname := range hostnames {
					tlsRule[hostname] = generateTLSTerminateRouteCfg(tcpRoute)
				}

				rules[int32(listener.Port)] = tlsRule
			case gwv1beta1.TCPProtocolType:
				rules[int32(listener.Port)] = generateTCPRouteCfg(tcpRoute)
			}
		}
	}
}

func processTCPBackends(tcpRoute *gwv1alpha2.TCPRoute, services map[string]serviceInfo) {
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(backend.BackendObjectReference, tcpRoute.Namespace); svcPort != nil {
				services[svcPort.String()] = serviceInfo{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func generateTCPRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) routecfg.RouteRule {
	backends := routecfg.TCPRouteRule{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendObjectReference, tcpRoute.Namespace); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}
