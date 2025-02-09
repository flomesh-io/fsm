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
			grpcRoute.GroupVersionKind(),
			grpcRoute.Spec.Hostnames,
			gwutils.ToSlicePtr(grpcRoute.Status.Parents),
		)

		if parentRef := c.getGRPCRouteParentRefToGateway(grpcRoute, rsh); parentRef == nil {
			continue
		} else {
			holder := rsh.StatusUpdateFor(*parentRef)

			if g2 := c.toV2GRPCRoute(grpcRoute, holder); g2 != nil {
				routes = append(routes, g2)
			}
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
	if c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends && len(r2.BackendRefs) == 0 {
		return nil
	}

	if len(r2.Filters) > 0 {
		r2.Filters = c.toV2GRPCRouteFilters(grpcRoute, rule.Filters)
	}

	return r2
}

func (c *ConfigGenerator) toV2GRPCBackendRefs(grpcRoute *gwv1.GRPCRoute, rule *gwv1.GRPCRouteRule, holder status.RouteParentStatusObject) []fgwv2.GRPCBackendRef {
	backendRefs := make([]fgwv2.GRPCBackendRef, 0)
	for _, bk := range rule.BackendRefs {
		if svcPort := c.backendRefToServicePortName(grpcRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
			if c.toFGWBackend(svcPort) == nil && c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends {
				continue
			}

			b2 := fgwv2.NewGRPCBackendRef(svcPort.String(), bk.BackendRef.Weight)

			if len(bk.Filters) > 0 {
				b2.Filters = c.toV2GRPCRouteFilters(grpcRoute, bk.Filters)
			}

			backendRefs = append(backendRefs, b2)

			for _, processor := range c.getBackendPolicyProcessors(grpcRoute) {
				processor.Process(grpcRoute, holder.GetParentRef(), rule, bk.BackendObjectReference, svcPort)
			}
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2GRPCRouteFilters(grpcRoute *gwv1.GRPCRoute, routeFilters []gwv1.GRPCRouteFilter) []fgwv2.GRPCRouteFilter {
	filters := make([]fgwv2.GRPCRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.GRPCRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(grpcRoute, f.RequestMirror.BackendRef); svcPort != nil {
				if c.toFGWBackend(svcPort) == nil {
					continue
				}

				f2 := fgwv2.GRPCRouteFilter{Key: uuid.NewString()}
				if err := gwutils.DeepCopy(&f2, &f); err != nil {
					log.Error().Msgf("Failed to copy RequestMirrorFilter: %v", err)
					continue
				}

				if f2.RequestMirror != nil {
					f2.RequestMirror.BackendRef = fgwv2.NewBackendRef(svcPort.String())
				}

				filters = append(filters, f2)
			}
		case gwv1.GRPCRouteFilterExtensionRef:
			filter := gwutils.ExtensionRefToFilter(c.client, grpcRoute, f.ExtensionRef)
			if filter == nil {
				continue
			}

			filterType := filter.Spec.Type
			filters = append(filters, fgwv2.GRPCRouteFilter{
				Type:            gwv1.GRPCRouteFilterType(filterType),
				ExtensionConfig: c.resolveFilterConfig(filter.Namespace, filter.Spec.ConfigRef),
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
				log.Error().Msgf("Failed to copy GRPCRouteFilter: %v", err)
				continue
			}
			filters = append(filters, f2)
		}
	}

	return filters
}

func (c *ConfigGenerator) getGRPCRouteParentRefToGateway(grpcRoute *gwv1.GRPCRoute, rsh status.RouteStatusObject) *gwv1.ParentReference {
	for _, parentRef := range grpcRoute.Spec.ParentRefs {
		parentRef := parentRef

		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		if !gwutils.IsEffectiveRouteForParent(rsh, parentRef) {
			continue
		}

		resolver := gwutils.NewGatewayListenerResolver(&DummyGatewayListenerConditionProvider{}, c.client, h)
		allowedListeners := resolver.GetAllowedListeners(c.gateway)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, grpcRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			return &parentRef
		}
	}

	return nil
}
