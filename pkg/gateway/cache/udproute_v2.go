package cache

import (
	"context"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"github.com/jinzhu/copier"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	"github.com/flomesh-io/fsm/pkg/gateway/status/route"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessorV2) processUDPRoutes() []interface{} {
	list := &gwv1alpha2.UDPRouteList{}
	if err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayUDPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
		return nil
	}

	routes := make([]interface{}, 0)
	for _, udpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := route.NewRouteStatusHolder(
			udpRoute,
			&udpRoute.ObjectMeta,
			&udpRoute.TypeMeta,
			nil,
			gwutils.ToSlicePtr(udpRoute.Status.Parents),
		)

		if c.ignoreUDPRoute(udpRoute, rsh) {
			continue
		}

		u2 := &v2.UDPRoute{}
		err := copier.CopyWithOption(u2, udpRoute, copier.Option{IgnoreEmpty: true, DeepCopy: true})
		if err != nil {
			log.Error().Msgf("Failed to copy UDPRoute: %v", err)
			continue
		}

		u2.Spec.Rules = make([]v2.UDPRouteRule, 0)
		for _, rule := range udpRoute.Spec.Rules {
			rule := rule
			r2 := &v2.UDPRouteRule{}
			if err := copier.CopyWithOption(r2, &rule, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
				log.Error().Msgf("Failed to copy UDPRouteRule: %v", err)
				continue
			}

			r2.BackendRefs = make([]v2.BackendRef, 0)
			for _, backend := range rule.BackendRefs {
				backend := backend
				if svcPort := c.backendRefToServicePortName(udpRoute, backend.BackendObjectReference); svcPort != nil {
					r2.BackendRefs = append(r2.BackendRefs, v2.BackendRef{
						Kind: "Backend",
						Name: svcPort.String(),
					})

					c.services[svcPort.String()] = serviceContextV2{
						svcPortName: *svcPort,
					}
				}
			}

			if len(r2.BackendRefs) == 0 {
				continue
			}

			u2.Spec.Rules = append(u2.Spec.Rules, *r2)
		}

		if len(u2.Spec.Rules) == 0 {
			continue
		}

		routes = append(routes, u2)
	}

	//c.resources = append(c.resources, routes...)
	return routes
}

func (c *GatewayProcessorV2) ignoreUDPRoute(udpRoute *gwv1alpha2.UDPRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range udpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		allowedListeners := gwutils.GetAllowedListeners(c.cache.client, c.gateway, h)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1.UDPProtocolType:
				return false
			}
		}
	}

	return true
}
