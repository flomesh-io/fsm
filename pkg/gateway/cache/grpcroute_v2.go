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

func (c *GatewayProcessorV2) processGRPCRoutes() []interface{} {
	list := &gwv1.GRPCRouteList{}
	if err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayGRPCRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		return nil
	}

	routes := make([]interface{}, 0)
	for _, grpcRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := route.NewRouteStatusHolder(
			grpcRoute,
			&grpcRoute.ObjectMeta,
			&grpcRoute.TypeMeta,
			grpcRoute.Spec.Hostnames,
			gwutils.ToSlicePtr(grpcRoute.Status.Parents),
		)

		if c.ignoreGRPCRoute(grpcRoute, rsh) {
			continue
		}

		if g2 := c.toV2GRPCRoute(grpcRoute); g2 != nil {
			routes = append(routes, g2)
		}
	}

	return routes
}

func (c *GatewayProcessorV2) toV2GRPCRoute(grpcRoute *gwv1.GRPCRoute) *v2.GRPCRoute {
	g2 := &v2.GRPCRoute{}
	if err := copier.CopyWithOption(g2, grpcRoute, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		log.Error().Msgf("Failed to copy GRPCRoute: %v", err)
		return nil
	}

	g2.Spec.Rules = make([]v2.GRPCRouteRule, 0)
	for _, rule := range grpcRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2GRPCRouteRule(grpcRoute, rule); r2 != nil {
			g2.Spec.Rules = append(g2.Spec.Rules, *r2)
		}
	}

	if len(g2.Spec.Rules) == 0 {
		return nil
	}

	return g2
}

func (c *GatewayProcessorV2) toV2GRPCRouteRule(grpcRoute *gwv1.GRPCRoute, rule gwv1.GRPCRouteRule) *v2.GRPCRouteRule {
	r2 := &v2.GRPCRouteRule{}
	if err := copier.CopyWithOption(r2, &rule, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		log.Error().Msgf("Failed to copy GRPCRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2GRPCBackendRefs(grpcRoute, rule.BackendRefs)
	if len(r2.BackendRefs) == 0 {
		return nil
	}

	if len(r2.Filters) > 0 {
		r2.Filters = c.toV2GRPCRouteFilters(grpcRoute, rule.Filters)
	}

	return r2
}

func (c *GatewayProcessorV2) toV2GRPCBackendRefs(grpcRoute *gwv1.GRPCRoute, refs []gwv1.GRPCBackendRef) []v2.GRPCBackendRef {
	backendRefs := make([]v2.GRPCBackendRef, 0)
	for _, bk := range refs {
		if svcPort := c.backendRefToServicePortName(grpcRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
			b2 := v2.GRPCBackendRef{
				Kind:   "Backend",
				Name:   svcPort.String(),
				Weight: backendWeight(bk.BackendRef),
			}

			if len(bk.Filters) > 0 {
				b2.Filters = c.toV2GRPCRouteFilters(grpcRoute, bk.Filters)
			}

			backendRefs = append(backendRefs, b2)

			c.services[svcPort.String()] = serviceContextV2{
				svcPortName: *svcPort,
			}
		}
	}

	return backendRefs
}

func (c *GatewayProcessorV2) toV2GRPCRouteFilters(grpcRoute *gwv1.GRPCRoute, routeFilters []gwv1.GRPCRouteFilter) []v2.GRPCRouteFilter {
	filters := make([]v2.GRPCRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.GRPCRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(grpcRoute, f.RequestMirror.BackendRef); svcPort != nil {
				filters = append(filters, v2.GRPCRouteFilter{
					Type: gwv1.GRPCRouteFilterRequestMirror,
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
			f2 := v2.GRPCRouteFilter{}
			if err := copier.CopyWithOption(&f2, &f, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
				continue
			}
			filters = append(filters, f2)
		}
	}

	return filters
}

func (c *GatewayProcessorV2) ignoreGRPCRoute(grpcRoute *gwv1.GRPCRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		allowedListeners := gwutils.GetAllowedListeners(c.cache.client, c.gateway, h)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, grpcRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			return false
		}
	}

	return true
}
