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

func (c *GatewayProcessor) processTCPRoutes() {
	list := &gwv1alpha2.TCPRouteList{}
	err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTCPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	})
	if err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
		return
	}

	for _, tcpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		c.processTCPRoute(tcpRoute)
	}
}

func (c *GatewayProcessor) processTCPRoute(tcpRoute *gwv1alpha2.TCPRoute) {
	for _, ref := range tcpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.cache.client, c.gateway, ref, gwutils.ToRouteContext(tcpRoute), c.validListeners)
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

				c.rules[int32(listener.Port)] = tlsRule
			case gwv1.TCPProtocolType:
				c.rules[int32(listener.Port)] = c.generateTCPRouteCfg(tcpRoute)
			}
		}
	}

	c.processTCPBackends(tcpRoute)
}

func (c *GatewayProcessor) processTCPBackends(tcpRoute *gwv1alpha2.TCPRoute) {
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(tcpRoute, backend.BackendObjectReference); svcPort != nil {
				c.services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func (c *GatewayProcessor) generateTCPRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) fgw.RouteRule {
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
