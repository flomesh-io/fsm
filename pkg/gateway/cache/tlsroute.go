package cache

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessor) processTLSRoutes() {
	list := &gwv1alpha2.TLSRouteList{}
	err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTLSRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	})
	if err != nil {
		log.Error().Msgf("Failed to list TLSRoutes: %v", err)
		return
	}

	for _, tlsRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		c.processTLSRoute(tlsRoute)
	}
}

func (c *GatewayProcessor) processTLSRoute(tlsRoute *gwv1alpha2.TLSRoute) {
	for _, ref := range tlsRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, c.gateway) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.cache.client, c.gateway, ref, gwutils.ToRouteContext(tlsRoute), c.validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			if listener.Protocol != gwv1.TLSProtocolType {
				continue
			}

			if listener.TLS == nil {
				continue
			}

			if listener.TLS.Mode == nil {
				continue
			}

			if *listener.TLS.Mode != gwv1.TLSModePassthrough {
				continue
			}

			hostnames := gwutils.GetValidHostnames(listener.Hostname, tlsRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			tlsRule := fgw.TLSPassthroughRouteRule{}
			for _, hostname := range hostnames {
				if target := generateTLSPassthroughRouteCfg(tlsRoute); target != nil {
					tlsRule[hostname] = *target
				}
			}

			c.rules[int32(listener.Port)] = tlsRule
		}
	}

	c.processTLSBackends(tlsRoute)
}

func (c *GatewayProcessor) processTLSBackends(_ *gwv1alpha2.TLSRoute) {
	// DO nothing for now
}

func (c *GatewayProcessor) generateTLSTerminateRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) fgw.TLSBackendService {
	backends := fgw.TLSBackendService{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(tcpRoute, bk.BackendObjectReference); svcPort != nil {
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
