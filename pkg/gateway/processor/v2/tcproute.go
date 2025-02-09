package v2

import (
	"context"

	"github.com/google/uuid"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	routestatus "github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *ConfigGenerator) processTCPRoutes() []fgwv2.Resource {
	list := &gwv1alpha2.TCPRouteList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTCPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
		return nil
	}

	routes := make([]fgwv2.Resource, 0)
	for _, tcpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		rsh := routestatus.NewRouteStatusHolder(
			tcpRoute,
			tcpRoute.GroupVersionKind(),
			nil,
			gwutils.ToSlicePtr(tcpRoute.Status.Parents),
		)

		if parentRef := c.getTCPRouteParentRefToGateway(tcpRoute, rsh); parentRef == nil {
			continue
		} else {
			holder := rsh.StatusUpdateFor(*parentRef)

			if t2 := c.toV2TCPRoute(tcpRoute, holder); t2 != nil {
				routes = append(routes, t2)
			}
		}
	}

	return routes
}

func (c *ConfigGenerator) toV2TCPRoute(tcpRoute *gwv1alpha2.TCPRoute, holder status.RouteParentStatusObject) *fgwv2.TCPRoute {
	t2 := &fgwv2.TCPRoute{}
	if err := gwutils.DeepCopy(t2, tcpRoute); err != nil {
		log.Error().Msgf("Failed to copy TCPRoute: %v", err)
		return nil
	}

	t2.Spec.Rules = make([]fgwv2.TCPRouteRule, 0)
	for _, rule := range tcpRoute.Spec.Rules {
		rule := rule
		if r2 := c.toV2TCPRouteRule(tcpRoute, rule, holder); r2 != nil {
			t2.Spec.Rules = append(t2.Spec.Rules, *r2)
		}
	}

	if len(t2.Spec.Rules) == 0 {
		return nil
	}

	return t2
}

func (c *ConfigGenerator) toV2TCPRouteRule(tcpRoute *gwv1alpha2.TCPRoute, rule gwv1alpha2.TCPRouteRule, holder status.RouteParentStatusObject) *fgwv2.TCPRouteRule {
	r2 := &fgwv2.TCPRouteRule{}
	if err := gwutils.DeepCopy(r2, &rule); err != nil {
		log.Error().Msgf("Failed to copy TCPRouteRule: %v", err)
		return nil
	}

	r2.BackendRefs = c.toV2TCPBackendRefs(tcpRoute, rule, holder)
	if c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends && len(r2.BackendRefs) == 0 {
		return nil
	}

	var filterRefs []gwpav1alpha2.LocalFilterReference
	for _, processor := range c.getFilterPolicyProcessors(tcpRoute) {
		filterRefs = append(filterRefs, processor.Process(tcpRoute, holder.GetParentRef(), rule.Name)...)
	}

	if len(filterRefs) > 0 {
		r2.Filters = c.toV2TCPRouteFilters(tcpRoute, filterRefs)
	}

	return r2
}

func (c *ConfigGenerator) toV2TCPBackendRefs(tcpRoute *gwv1alpha2.TCPRoute, rule gwv1alpha2.TCPRouteRule, holder status.RouteParentStatusObject) []fgwv2.BackendRef {
	backendRefs := make([]fgwv2.BackendRef, 0)
	for _, backend := range rule.BackendRefs {
		backend := backend
		if svcPort := c.backendRefToServicePortName(tcpRoute, backend.BackendObjectReference); svcPort != nil {
			if c.toFGWBackend(svcPort) == nil && c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends {
				continue
			}

			backendRefs = append(backendRefs, fgwv2.NewBackendRefWithWeight(svcPort.String(), backend.Weight))

			for _, processor := range c.getBackendPolicyProcessors(tcpRoute) {
				processor.Process(tcpRoute, holder.GetParentRef(), rule, backend.BackendObjectReference, svcPort)
			}
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) toV2TCPRouteFilters(udpRoute *gwv1alpha2.TCPRoute, filterRefs []gwpav1alpha2.LocalFilterReference) []fgwv2.NonHTTPRouteFilter {
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
		if filterProtocol != extv1alpha1.FilterProtocolTCP {
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

func (c *ConfigGenerator) getTCPRouteParentRefToGateway(tcpRoute *gwv1alpha2.TCPRoute, rsh status.RouteStatusObject) *gwv1.ParentReference {
	for _, parentRef := range tcpRoute.Spec.ParentRefs {
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
			case gwv1.TLSProtocolType:
				if listener.TLS == nil {
					continue
				}

				if listener.TLS.Mode == nil {
					continue
				}

				if *listener.TLS.Mode != gwv1.TLSModeTerminate {
					continue
				}

				hostnames := gwutils.GetValidHostnames(listener.Hostname, nil)

				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}

				return &parentRef
			case gwv1.TCPProtocolType:
				return &parentRef
			}
		}
	}

	return nil
}
