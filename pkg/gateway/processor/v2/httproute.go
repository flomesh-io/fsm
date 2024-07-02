package v2

import (
	"context"

	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"github.com/flomesh-io/fsm/pkg/gateway/status/route"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *ConfigGenerator) processHTTPRoutes() []interface{} {
	list := &gwv1.HTTPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
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

		holder := rsh.StatusUpdateFor(
			gwv1.ParentReference{
				Namespace: ptr.To(gwv1.Namespace(c.gateway.Namespace)),
				Name:      gwv1.ObjectName(c.gateway.Name),
			},
		)

		if h2 := c.toV2HTTPRoute(httpRoute, holder); h2 != nil {
			routes = append(routes, h2)
		}
	}

	return routes
}

func (c *ConfigGenerator) toV2HTTPRoute(httpRoute *gwv1.HTTPRoute, holder status.RouteParentStatusObject) *v2.HTTPRoute {
	h2 := &v2.HTTPRoute{}
	if err := gwutils.DeepCopy(h2, httpRoute); err != nil {
		log.Error().Msgf("Failed to copy HTTPRoute: %v", err)
		return nil
	}

	h2.Spec.Rules = make([]v2.HTTPRouteRule, 0)
	for _, rule := range httpRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2HTTPRouteRule(httpRoute, rule, holder); r2 != nil {
			h2.Spec.Rules = append(h2.Spec.Rules, *r2)
		}
	}

	if len(h2.Spec.Rules) == 0 {
		return nil
	}

	return h2
}

func (c *ConfigGenerator) toV2HTTPRouteRule(httpRoute *gwv1.HTTPRoute, rule gwv1.HTTPRouteRule, holder status.RouteParentStatusObject) *v2.HTTPRouteRule {
	r2 := &v2.HTTPRouteRule{}
	if err := gwutils.DeepCopy(r2, &rule); err != nil {
		log.Error().Msgf("Failed to copy HTTPRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2HTTPBackendRefs(httpRoute, rule.BackendRefs, holder)
	if len(r2.BackendRefs) == 0 {
		return nil
	}

	if len(rule.Filters) > 0 {
		r2.Filters = c.toV2HTTPRouteFilters(httpRoute, rule.Filters, holder)
	}

	return r2
}

func (c *ConfigGenerator) toV2HTTPBackendRefs(httpRoute *gwv1.HTTPRoute, refs []gwv1.HTTPBackendRef, holder status.RouteParentStatusObject) []v2.HTTPBackendRef {
	backendRefs := make([]v2.HTTPBackendRef, 0)
	for _, bk := range refs {
		if svcPort := c.backendRefToServicePortName(httpRoute, bk.BackendRef.BackendObjectReference, holder); svcPort != nil {
			b2 := v2.NewHTTPBackendRef(svcPort.String(), backendWeight(bk.BackendRef))

			if len(bk.Filters) > 0 {
				b2.Filters = c.toV2HTTPRouteFilters(httpRoute, bk.Filters, holder)
			}

			backendRefs = append(backendRefs, b2)

			for _, processor := range c.getBackendPolicyProcessors(httpRoute) {
				processor.Process(httpRoute, holder.GetParentRef(), bk.BackendObjectReference, svcPort)
			}

			c.services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2HTTPRouteFilters(httpRoute *gwv1.HTTPRoute, routeFilters []gwv1.HTTPRouteFilter, holder status.RouteParentStatusObject) []v2.HTTPRouteFilter {
	filters := make([]v2.HTTPRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.HTTPRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(httpRoute, f.RequestMirror.BackendRef, holder); svcPort != nil {
				filters = append(filters, v2.HTTPRouteFilter{
					Type: gwv1.HTTPRouteFilterRequestMirror,
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
			f2 := v2.HTTPRouteFilter{}
			if err := gwutils.DeepCopy(f2, f); err != nil {
				continue
			}
			filters = append(filters, f2)
		}
	}

	return filters
}

func (c *ConfigGenerator) ignoreHTTPRoute(httpRoute *gwv1.HTTPRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range httpRoute.Spec.ParentRefs {
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
