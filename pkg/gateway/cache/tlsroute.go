package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func processTLSRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, tlsRoute *gwv1alpha2.TLSRoute, rules map[int32]routecfg.RouteRule) {
	for _, ref := range tlsRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, tlsRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			if listener.Protocol != gwv1beta1.TLSProtocolType {
				continue
			}

			if listener.TLS == nil {
				continue
			}

			if listener.TLS.Mode == nil {
				continue
			}

			if *listener.TLS.Mode != gwv1beta1.TLSModePassthrough {
				continue
			}

			hostnames := gwutils.GetValidHostnames(listener.Hostname, tlsRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			tlsRule := routecfg.TLSPassthroughRouteRule{}
			for _, hostname := range hostnames {
				if target := generateTLSPassthroughRouteCfg(tlsRoute); target != nil {
					tlsRule[hostname] = *target
				}
			}

			rules[int32(listener.Port)] = tlsRule
		}
	}
}

func processTLSBackends(_ *gwv1alpha2.TLSRoute, _ map[string]serviceInfo) {
	// DO nothing for now
}

func generateTLSTerminateRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) routecfg.TLSBackendService {
	backends := routecfg.TLSBackendService{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendObjectReference, tcpRoute.Namespace); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}

func generateTLSPassthroughRouteCfg(tlsRoute *gwv1alpha2.TLSRoute) *string {
	for _, rule := range tlsRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			// return the first ONE
			return passthroughTarget(bk)
		}
	}

	return nil
}
