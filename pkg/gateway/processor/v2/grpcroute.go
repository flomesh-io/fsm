package v2

import (
	"context"

	routestatus "github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *ConfigGenerator) processGRPCRoutes() []interface{} {
	list := &gwv1.GRPCRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayGRPCRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		return nil
	}

	routes := make([]interface{}, 0)
	for _, grpcRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := routestatus.NewRouteStatusHolder(
			grpcRoute,
			&grpcRoute.ObjectMeta,
			&grpcRoute.TypeMeta,
			grpcRoute.Spec.Hostnames,
			gwutils.ToSlicePtr(grpcRoute.Status.Parents),
		)

		if c.ignoreGRPCRoute(grpcRoute, rsh) {
			continue
		}

		holder := rsh.StatusUpdateFor(
			gwv1.ParentReference{
				Namespace: ptr.To(gwv1.Namespace(c.gateway.Namespace)),
				Name:      gwv1.ObjectName(c.gateway.Name),
			},
		)

		if g2 := c.toV2GRPCRoute(grpcRoute, holder); g2 != nil {
			routes = append(routes, g2)
		}
	}

	return routes
}

func (c *ConfigGenerator) toV2GRPCRoute(grpcRoute *gwv1.GRPCRoute, holder status.RouteParentStatusObject) *v2.GRPCRoute {
	g2 := &v2.GRPCRoute{}
	if err := gwutils.DeepCopy(g2, grpcRoute); err != nil {
		log.Error().Msgf("Failed to copy GRPCRoute: %v", err)
		return nil
	}

	g2.Spec.Rules = make([]v2.GRPCRouteRule, 0)
	for _, rule := range grpcRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2GRPCRouteRule(grpcRoute, rule, holder); r2 != nil {
			g2.Spec.Rules = append(g2.Spec.Rules, *r2)
		}
	}

	if len(g2.Spec.Rules) == 0 {
		return nil
	}

	return g2
}

func (c *ConfigGenerator) toV2GRPCRouteRule(grpcRoute *gwv1.GRPCRoute, rule gwv1.GRPCRouteRule, holder status.RouteParentStatusObject) *v2.GRPCRouteRule {
	r2 := &v2.GRPCRouteRule{}
	if err := gwutils.DeepCopy(r2, &rule); err != nil {
		log.Error().Msgf("Failed to copy GRPCRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2GRPCBackendRefs(grpcRoute, rule.BackendRefs, holder)
	if len(r2.BackendRefs) == 0 {
		return nil
	}

	if len(r2.Filters) > 0 {
		r2.Filters = c.toV2GRPCRouteFilters(grpcRoute, rule.Filters, holder)
	}

	return r2
}

func (c *ConfigGenerator) toV2GRPCBackendRefs(grpcRoute *gwv1.GRPCRoute, refs []gwv1.GRPCBackendRef, holder status.RouteParentStatusObject) []v2.GRPCBackendRef {
	backendRefs := make([]v2.GRPCBackendRef, 0)
	for _, bk := range refs {
		if svcPort := c.backendRefToServicePortName(grpcRoute, bk.BackendRef.BackendObjectReference, holder); svcPort != nil {
			b2 := v2.NewGRPCBackendRef(svcPort.String(), backendWeight(bk.BackendRef))

			if len(bk.Filters) > 0 {
				b2.Filters = c.toV2GRPCRouteFilters(grpcRoute, bk.Filters, holder)
			}

			backendRefs = append(backendRefs, b2)

			for _, processor := range c.getBackendPolicyProcessors(grpcRoute) {
				processor.Process(grpcRoute, holder.GetParentRef(), bk.BackendObjectReference, svcPort)
			}

			c.services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2GRPCRouteFilters(grpcRoute *gwv1.GRPCRoute, routeFilters []gwv1.GRPCRouteFilter, holder status.RouteParentStatusObject) []v2.GRPCRouteFilter {
	filters := make([]v2.GRPCRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.GRPCRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(grpcRoute, f.RequestMirror.BackendRef, holder); svcPort != nil {
				filters = append(filters, v2.GRPCRouteFilter{
					Type: gwv1.GRPCRouteFilterRequestMirror,
					RequestMirror: &v2.HTTPRequestMirrorFilter{
						BackendRef: v2.NewBackendRefWithWeight(svcPort.String(), 1),
					},
				})

				// TODO: process backend level policies here??? TBD

				c.services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		default:
			f2 := v2.GRPCRouteFilter{}
			if err := gwutils.DeepCopy(&f2, &f); err != nil {
				continue
			}
			filters = append(filters, f2)
		}
	}

	return filters
}

func (c *ConfigGenerator) ignoreGRPCRoute(grpcRoute *gwv1.GRPCRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		if !gwutils.IsEffectiveRouteForParent(rsh, parentRef) {
			continue
		}

		allowedListeners := gwutils.GetAllowedListeners(c.client, c.gateway, h)
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
