package v2

import (
	"context"

	"github.com/google/uuid"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	routestatus "github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *ConfigGenerator) processGRPCRoutes() []fgwv2.Resource {
	list := &gwv1.GRPCRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayGRPCRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		return nil
	}

	routes := make([]fgwv2.Resource, 0)
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

func (c *ConfigGenerator) toV2GRPCRoute(grpcRoute *gwv1.GRPCRoute, holder status.RouteParentStatusObject) *fgwv2.GRPCRoute {
	g2 := &fgwv2.GRPCRoute{}
	if err := gwutils.DeepCopy(g2, grpcRoute); err != nil {
		log.Error().Msgf("Failed to copy GRPCRoute: %v", err)
		return nil
	}

	g2.Spec.Rules = make([]fgwv2.GRPCRouteRule, 0)
	for _, rule := range grpcRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2GRPCRouteRule(grpcRoute, &rule, holder); r2 != nil {
			g2.Spec.Rules = append(g2.Spec.Rules, *r2)
		}
	}

	if len(g2.Spec.Rules) == 0 {
		return nil
	}

	return g2
}

func (c *ConfigGenerator) toV2GRPCRouteRule(grpcRoute *gwv1.GRPCRoute, rule *gwv1.GRPCRouteRule, holder status.RouteParentStatusObject) *fgwv2.GRPCRouteRule {
	r2 := &fgwv2.GRPCRouteRule{}
	if err := gwutils.DeepCopy(r2, rule); err != nil {
		log.Error().Msgf("Failed to copy GRPCRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2GRPCBackendRefs(grpcRoute, rule, holder)
	if len(r2.BackendRefs) == 0 {
		return nil
	}

	if len(r2.Filters) > 0 {
		r2.Filters = c.toV2GRPCRouteFilters(grpcRoute, rule.Filters, holder)
	}

	return r2
}

func (c *ConfigGenerator) toV2GRPCBackendRefs(grpcRoute *gwv1.GRPCRoute, rule *gwv1.GRPCRouteRule, holder status.RouteParentStatusObject) []fgwv2.GRPCBackendRef {
	backendRefs := make([]fgwv2.GRPCBackendRef, 0)
	for _, bk := range rule.BackendRefs {
		if svcPort := c.backendRefToServicePortName(grpcRoute, bk.BackendRef.BackendObjectReference, holder); svcPort != nil {
			b2 := fgwv2.NewGRPCBackendRef(svcPort.String(), backendWeight(bk.BackendRef))

			if len(bk.Filters) > 0 {
				b2.Filters = c.toV2GRPCRouteFilters(grpcRoute, bk.Filters, holder)
			}

			backendRefs = append(backendRefs, b2)

			for _, processor := range c.getBackendPolicyProcessors(grpcRoute) {
				processor.Process(grpcRoute, holder.GetParentRef(), rule, bk.BackendObjectReference, svcPort)
			}

			c.services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2GRPCRouteFilters(grpcRoute *gwv1.GRPCRoute, routeFilters []gwv1.GRPCRouteFilter, holder status.RouteParentStatusObject) []fgwv2.GRPCRouteFilter {
	filters := make([]fgwv2.GRPCRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.GRPCRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(grpcRoute, f.RequestMirror.BackendRef, holder); svcPort != nil {
				f2 := fgwv2.GRPCRouteFilter{Key: uuid.NewString()}
				if err := gwutils.DeepCopy(&f2, &f); err != nil {
					log.Error().Msgf("Failed to copy RequestMirrorFilter: %v", err)
					continue
				}

				if f2.RequestMirror != nil {
					f2.RequestMirror.BackendRef = fgwv2.NewBackendRefWithWeight(svcPort.String(), 1)
				}

				filters = append(filters, f2)

				c.services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		case gwv1.GRPCRouteFilterExtensionRef:
			filter := gwutils.ExtensionRefToFilter(c.client, grpcRoute, f.ExtensionRef)
			if filter == nil {
				continue
			}

			filterType := filter.Spec.Type
			filters = append(filters, fgwv2.GRPCRouteFilter{
				Type:            gwv1.GRPCRouteFilterType(filterType),
				ExtensionConfig: c.resolveFilterConfig(filter.Spec.ConfigRef),
				Key:             uuid.NewString(),
			})

			definition := c.resolveFilterDefinition(filterType, extv1alpha1.FilterScopeRoute, filter.Spec.DefinitionRef)
			if definition == nil {
				continue
			}

			filterProtocol := ptr.Deref(definition.Spec.Protocol, extv1alpha1.FilterProtocolHTTP)
			if filterProtocol != extv1alpha1.FilterProtocolHTTP {
				continue
			}

			if c.filters[filterProtocol] == nil {
				c.filters[filterProtocol] = map[extv1alpha1.FilterType]string{}
			}
			if _, ok := c.filters[filterProtocol][filterType]; !ok {
				c.filters[filterProtocol][filterType] = definition.Spec.Script
			}
		default:
			f2 := fgwv2.GRPCRouteFilter{Key: uuid.NewString()}
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
