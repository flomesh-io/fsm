package cache

import (
	"context"

	"github.com/jinzhu/copier"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"github.com/flomesh-io/fsm/pkg/gateway/status/route"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessorV2) processHTTPRoutes() {
	list := &gwv1.HTTPRouteList{}
	if err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayHTTPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return
	}

	routes := make([]interface{}, 0)
	for _, httpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := route.NewRouteStatusHolder(
			httpRoute,
			&httpRoute.ObjectMeta,
			&httpRoute.TypeMeta,
			httpRoute.Spec.Hostnames,
			gwutils.ToSlicePtr(httpRoute.Status.Parents),
		)

		if c.ignoreHTTPRoute(httpRoute, rsh) {
			continue
		}

		h2 := &v2.HTTPRoute{}
		err := copier.CopyWithOption(h2, httpRoute, copier.Option{IgnoreEmpty: true, DeepCopy: true})
		if err != nil {
			log.Error().Msgf("Failed to copy HTTPRoute: %v", err)
			continue
		}

		h2.Spec.Rules = make([]v2.HTTPRouteRule, 0)
		for _, rule := range httpRoute.Spec.Rules {
			rule := rule
			r2 := &v2.HTTPRouteRule{}
			if err := copier.CopyWithOption(r2, &rule, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
				log.Error().Msgf("Failed to copy HTTPRouteRule: %v", err)
				continue
			}

			r2.BackendRefs = make([]gwv1.HTTPBackendRef, 0)
			for _, bk := range rule.BackendRefs {
				bk := bk
				if svcPort := c.backendRefToServicePortName(httpRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
					r2.BackendRefs = append(r2.BackendRefs, *bk.DeepCopy())
				}
			}

			if len(r2.BackendRefs) == 0 {
				continue
			}

			h2.Spec.Rules = append(h2.Spec.Rules, *r2)
		}

		if len(h2.Spec.Rules) == 0 {
			continue
		}

		routes = append(routes, h2)
	}

	c.resources = append(c.resources, routes...)
}

func (c *GatewayProcessorV2) ignoreHTTPRoute(httpRoute *gwv1.HTTPRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		allowedListeners := gwutils.GetAllowedListeners(c.cache.client, c.gateway, h)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, httpRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			return false
		}
	}

	return true
}
