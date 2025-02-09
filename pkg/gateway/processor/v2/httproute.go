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

func (c *ConfigGenerator) processHTTPRoutes() []fgwv2.Resource {
	list := &gwv1.HTTPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayHTTPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return nil
	}

	routes := make([]fgwv2.Resource, 0)
	for _, httpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := routestatus.NewRouteStatusHolder(
			httpRoute,
			httpRoute.GroupVersionKind(),
			httpRoute.Spec.Hostnames,
			gwutils.ToSlicePtr(httpRoute.Status.Parents),
		)

		if parentRef := c.getHTTPRouteParentRefToGateway(httpRoute, rsh); parentRef == nil {
			continue
		} else {
			holder := rsh.StatusUpdateFor(*parentRef)

			if h2 := c.toV2HTTPRoute(httpRoute, holder); h2 != nil {
				routes = append(routes, h2)
			}
		}
	}

	return routes
}

func (c *ConfigGenerator) toV2HTTPRoute(httpRoute *gwv1.HTTPRoute, holder status.RouteParentStatusObject) *fgwv2.HTTPRoute {
	h2 := &fgwv2.HTTPRoute{}
	if err := gwutils.DeepCopy(h2, httpRoute); err != nil {
		log.Error().Msgf("Failed to copy HTTPRoute: %v", err)
		return nil
	}

	h2.Spec.Rules = make([]fgwv2.HTTPRouteRule, 0)
	for _, rule := range httpRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2HTTPRouteRule(httpRoute, &rule, holder); r2 != nil {
			h2.Spec.Rules = append(h2.Spec.Rules, *r2)
		}
	}

	if len(h2.Spec.Rules) == 0 {
		return nil
	}

	return h2
}

func (c *ConfigGenerator) toV2HTTPRouteRule(httpRoute *gwv1.HTTPRoute, rule *gwv1.HTTPRouteRule, holder status.RouteParentStatusObject) *fgwv2.HTTPRouteRule {
	r2 := &fgwv2.HTTPRouteRule{}
	if err := gwutils.DeepCopy(r2, rule); err != nil {
		log.Error().Msgf("Failed to copy HTTPRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2HTTPBackendRefs(httpRoute, rule, holder)
	if c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends && len(r2.BackendRefs) == 0 {
		return nil
	}

	if len(rule.Filters) > 0 {
		r2.Filters = c.toV2HTTPRouteFilters(httpRoute, rule.Filters)
	}

	return r2
}

func (c *ConfigGenerator) toV2HTTPBackendRefs(httpRoute *gwv1.HTTPRoute, rule *gwv1.HTTPRouteRule, holder status.RouteParentStatusObject) []fgwv2.HTTPBackendRef {
	backendRefs := make([]fgwv2.HTTPBackendRef, 0)
	for _, bk := range rule.BackendRefs {
		if svcPort := c.backendRefToServicePortName(httpRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
			if c.toFGWBackend(svcPort) == nil && c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends {
				continue
			}

			b2 := fgwv2.NewHTTPBackendRef(svcPort.String(), bk.BackendRef.Weight)

			if len(bk.Filters) > 0 {
				b2.Filters = c.toV2HTTPRouteFilters(httpRoute, bk.Filters)
			}

			backendRefs = append(backendRefs, b2)

			for _, processor := range c.getBackendPolicyProcessors(httpRoute) {
				processor.Process(httpRoute, holder.GetParentRef(), rule, bk.BackendObjectReference, svcPort)
			}
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2HTTPRouteFilters(httpRoute *gwv1.HTTPRoute, routeFilters []gwv1.HTTPRouteFilter) []fgwv2.HTTPRouteFilter {
	filters := make([]fgwv2.HTTPRouteFilter, 0)
	for _, f := range routeFilters {
		f := f
		switch f.Type {
		case gwv1.HTTPRouteFilterRequestMirror:
			if svcPort := c.backendRefToServicePortName(httpRoute, f.RequestMirror.BackendRef); svcPort != nil {
				if c.toFGWBackend(svcPort) == nil {
					continue
				}

				f2 := fgwv2.HTTPRouteFilter{Key: uuid.NewString()}
				if err := gwutils.DeepCopy(&f2, &f); err != nil {
					log.Error().Msgf("Failed to copy RequestMirrorFilter: %v", err)
					continue
				}

				if f2.RequestMirror != nil {
					f2.RequestMirror.BackendRef = fgwv2.NewBackendRef(svcPort.String())
				}

				filters = append(filters, f2)
			}
		case gwv1.HTTPRouteFilterExtensionRef:
			filter := gwutils.ExtensionRefToFilter(c.client, httpRoute, f.ExtensionRef)
			if filter == nil {
				continue
			}

			filterType := filter.Spec.Type
			filters = append(filters, fgwv2.HTTPRouteFilter{
				Type:            gwv1.HTTPRouteFilterType(filterType),
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
			f2 := fgwv2.HTTPRouteFilter{Key: uuid.NewString()}
			if err := gwutils.DeepCopy(&f2, &f); err != nil {
				log.Error().Msgf("Failed to copy HTTPRouteFilter: %v", err)
				continue
			}
			filters = append(filters, f2)
		}
	}

	return filters
}

func (c *ConfigGenerator) getHTTPRouteParentRefToGateway(httpRoute *gwv1.HTTPRoute, rsh status.RouteStatusObject) *gwv1.ParentReference {
	for _, parentRef := range httpRoute.Spec.ParentRefs {
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
			hostnames := gwutils.GetValidHostnames(listener.Hostname, httpRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			return &parentRef
		}
	}

	return nil
}
