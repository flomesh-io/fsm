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

func (c *GatewayProcessorV2) processHTTPRoutes() []interface{} {
	list := &gwv1.HTTPRouteList{}
	if err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayHTTPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return nil
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

			if len(rule.Filters) > 0 {
				r2.Filters = c.toV2HTTPRouteFilters(httpRoute, rule.Filters)
			}

			r2.BackendRefs = make([]v2.HTTPBackendRef, 0)
			for _, bk := range rule.BackendRefs {
				if svcPort := c.backendRefToServicePortName(httpRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
					//bkCopy := bk.DeepCopy()
					b2 := v2.HTTPBackendRef{
						Kind:   "Backend",
						Name:   svcPort.String(),
						Weight: backendWeight(bk.BackendRef),
					}

					if len(bk.Filters) > 0 {
						b2.Filters = c.toV2HTTPRouteFilters(httpRoute, bk.Filters)
					}

					r2.BackendRefs = append(r2.BackendRefs, b2)

					c.services[svcPort.String()] = serviceContextV2{
						svcPortName: *svcPort,
					}
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

	//c.resources = append(c.resources, routes...)

	return routes
}

func (c *GatewayProcessorV2) toV2HTTPRouteFilters(httpRoute *gwv1.HTTPRoute, routeFilters []gwv1.HTTPRouteFilter) []v2.HTTPRouteFilter {
	filters := make([]v2.HTTPRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.HTTPRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(httpRoute, f.RequestMirror.BackendRef); svcPort != nil {
				filters = append(filters, v2.HTTPRouteFilter{
					Type: gwv1.HTTPRouteFilterRequestMirror,
					RequestMirror: &v2.HTTPRequestMirrorFilter{
						BackendRef: v2.BackendRef{
							Kind:   "Backend",
							Name:   svcPort.String(),
							Weight: 1,
						},
					},
				})
				c.services[svcPort.String()] = serviceContextV2{
					svcPortName: *svcPort,
				}
			}
		default:
			f2 := v2.HTTPRouteFilter{}
			if err := copier.CopyWithOption(&f2, &f, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
				continue
			}
			filters = append(filters, f2)
		}
	}

	return filters
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
