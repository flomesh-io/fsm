package v2

import (
	"context"

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
			&udpRoute.ObjectMeta,
			&udpRoute.TypeMeta,
			nil,
			gwutils.ToSlicePtr(udpRoute.Status.Parents),
		)

		if c.ignoreUDPRoute(udpRoute, rsh) {
			continue
		}

		holder := rsh.StatusUpdateFor(
			gwv1.ParentReference{
				Namespace: ptr.To(gwv1.Namespace(c.gateway.Namespace)),
				Name:      gwv1.ObjectName(c.gateway.Name),
			},
		)

		if u2 := c.toV2UDPRoute(udpRoute, holder); u2 != nil {
			routes = append(routes, u2)
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

	return r2
}

func (c *ConfigGenerator) toV2UDPBackendRefs(udpRoute *gwv1alpha2.UDPRoute, rule gwv1alpha2.UDPRouteRule, holder status.RouteParentStatusObject) []fgwv2.BackendRef {
	backendRefs := make([]fgwv2.BackendRef, 0)
	for _, backend := range rule.BackendRefs {
		backend := backend
		if svcPort := c.backendRefToServicePortName(udpRoute, backend.BackendObjectReference, holder); svcPort != nil {
			if c.toFGWBackend(svcPort) == nil && c.cfg.GetFeatureFlags().DropRouteRuleIfNoAvailableBackends {
				continue
			}

			backendRefs = append(backendRefs, fgwv2.NewBackendRefWithWeight(svcPort.String(), backend.Weight))
		}
	}

	return backendRefs
}

func (c *ConfigGenerator) ignoreUDPRoute(udpRoute *gwv1alpha2.UDPRoute, rsh status.RouteStatusObject) bool {
	for _, parentRef := range udpRoute.Spec.ParentRefs {
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
				return false
			}
		}
	}

	return true
}
