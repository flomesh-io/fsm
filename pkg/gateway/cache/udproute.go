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

func (c *GatewayProcessor) processUDPRoutes() {
	list := &gwv1alpha2.UDPRouteList{}
	err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayUDPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	})
	if err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
		return
	}

	for _, udpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		c.processUDPRoute(udpRoute)
	}
}

func (c *GatewayProcessor) processUDPRoute(udpRoute *gwv1alpha2.UDPRoute) {
	for _, ref := range udpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.cache.client, c.gateway, ref, gwutils.ToRouteContext(udpRoute), c.validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1.UDPProtocolType:
				c.rules[int32(listener.Port)] = c.generateUDPRouteCfg(udpRoute)
			}
		}
	}

	c.processUDPBackends(udpRoute)
}

func (c *GatewayProcessor) processUDPBackends(udpRoute *gwv1alpha2.UDPRoute) {
	for _, rule := range udpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(udpRoute, backend.BackendObjectReference); svcPort != nil {
				c.services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func (c *GatewayProcessor) generateUDPRouteCfg(udpRoute *gwv1alpha2.UDPRoute) fgw.RouteRule {
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
