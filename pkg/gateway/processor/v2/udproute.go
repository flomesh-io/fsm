package v2

import (
	"context"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/google/uuid"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"k8s.io/utils/ptr"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
	routestatus "github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *ConfigGenerator) processUDPRoutes() []fgwv2.Resource {
	list := &gwv1alpha2.UDPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayUDPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
		return nil
	}

	routes := make([]fgwv2.Resource, 0)
	for _, udpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := routestatus.NewRouteStatusHolder(
			udpRoute,
			udpRoute.GroupVersionKind(),
			nil,
			gwutils.ToSlicePtr(udpRoute.Status.Parents),
		)

		if parentRef := c.getUDPRouteParentRefToGateway(udpRoute, rsh); parentRef == nil {
			continue
		} else {
			holder := rsh.StatusUpdateFor(*parentRef)

			if u2 := c.toV2UDPRoute(udpRoute, holder); u2 != nil {
				routes = append(routes, u2)
			}
		}
	}

	return routes
}

func (c *ConfigGenerator) toV2UDPRoute(udpRoute *gwv1alpha2.UDPRoute, holder status.RouteParentStatusObject) *fgwv2.UDPRoute {
	u2 := &fgwv2.UDPRoute{}
	if err := gwutils.DeepCopy(u2, udpRoute); err != nil {
		log.Error().Msgf("Failed to copy UDPRoute: %v", err)
		return nil
	}

	u2.Spec.Rules = make([]fgwv2.UDPRouteRule, 0)
	for _, rule := range udpRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2UDPRouteRule(udpRoute, rule, holder); r2 != nil {
			u2.Spec.Rules = append(u2.Spec.Rules, *r2)
		}
	}

	if len(u2.Spec.Rules) == 0 {
		return nil
	}

	return u2
}

func (c *ConfigGenerator) toV2UDPRouteRule(udpRoute *gwv1alpha2.UDPRoute, rule gwv1alpha2.UDPRouteRule, holder status.RouteParentStatusObject) *fgwv2.UDPRouteRule {
	r2 := &fgwv2.UDPRouteRule{}
	if err := gwutils.DeepCopy(r2, &rule); err != nil {
		log.Error().Msgf("Failed to copy UDPRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2UDPBackendRefs(udpRoute, rule, holder)
	if c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends && len(r2.BackendRefs) == 0 {
		return nil
	}

	var filterRefs []gwpav1alpha2.LocalFilterReference
	for _, processor := range c.getFilterPolicyProcessors(udpRoute) {
		filterRefs = append(filterRefs, processor.Process(udpRoute, holder.GetParentRef(), rule.Name)...)
	}

	if len(filterRefs) > 0 {
		r2.Filters = c.toV2UDPRouteFilters(udpRoute, filterRefs)
	}

	return r2
}

func (c *ConfigGenerator) toV2UDPBackendRefs(udpRoute *gwv1alpha2.UDPRoute, rule gwv1alpha2.UDPRouteRule, holder status.RouteParentStatusObject) []fgwv2.BackendRef {
	backendRefs := make([]fgwv2.BackendRef, 0)
	for _, backend := range rule.BackendRefs {
		backend := backend
		if svcPort := c.backendRefToServicePortName(udpRoute, backend.BackendObjectReference); svcPort != nil {
			if c.toFGWBackend(svcPort) == nil && c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends {
				continue
			}

			backendRefs = append(backendRefs, fgwv2.NewBackendRefWithWeight(svcPort.String(), backend.Weight))
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2UDPRouteFilters(udpRoute *gwv1alpha2.UDPRoute, filterRefs []gwpav1alpha2.LocalFilterReference) []fgwv2.NonHTTPRouteFilter {
	var filters []fgwv2.NonHTTPRouteFilter

	for _, filterRef := range gwutils.SortFilterRefs(filterRefs) {
		filter := gwutils.FilterRefToFilter(c.client, udpRoute, filterRef)
		if filter == nil {
			continue
		}

		filterType := filter.Spec.Type
		filters = append(filters, fgwv2.NonHTTPRouteFilter{
			Type:            fgwv2.NonHTTPRouteFilterType(filterType),
			ExtensionConfig: c.resolveFilterConfig(filter.Namespace, filter.Spec.ConfigRef),
			Key:             uuid.NewString(),
			Priority:        ptr.Deref(filterRef.Priority, 100),
		})

		definition := c.resolveFilterDefinition(filterType, extv1alpha1.FilterScopeRoute, filter.Spec.DefinitionRef)
		if definition == nil {
			continue
		}

		filterProtocol := ptr.Deref(definition.Spec.Protocol, extv1alpha1.FilterProtocolHTTP)
		if filterProtocol != extv1alpha1.FilterProtocolUDP {
			continue
		}

		if c.filters[filterProtocol] == nil {
			c.filters[filterProtocol] = map[extv1alpha1.FilterType]string{}
		}
		if _, ok := c.filters[filterProtocol][filterType]; !ok {
			c.filters[filterProtocol][filterType] = definition.Spec.Script
		}
	}

	return filters
}

func (c *ConfigGenerator) getUDPRouteParentRefToGateway(udpRoute *gwv1alpha2.UDPRoute, rsh status.RouteStatusObject) *gwv1.ParentReference {
	for _, parentRef := range udpRoute.Spec.ParentRefs {
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
			switch listener.Protocol {
			case gwv1.UDPProtocolType:
				return &parentRef
			}
		}
	}

	return nil
}
