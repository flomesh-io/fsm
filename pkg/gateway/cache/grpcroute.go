package cache

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/gateway/status/route"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessor) processGRPCRoutes() {
	list := &gwv1.GRPCRouteList{}
	err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayGRPCRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	})
	if err != nil {
		log.Error().Msgf("Failed to list GRPCRoutes: %v", err)
		return
	}

	for _, grpcRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		c.processGRPCRoute(grpcRoute)
	}
}

func (c *GatewayProcessor) processGRPCRoute(grpcRoute *gwv1.GRPCRoute) {
	hostnameEnrichers := c.getHostnamePolicyEnrichers(grpcRoute)
	rsh := route.NewRouteStatusHolder(
		grpcRoute,
		&grpcRoute.ObjectMeta,
		&grpcRoute.TypeMeta,
		grpcRoute.Spec.Hostnames,
		gwutils.ToSlicePtr(grpcRoute.Status.Parents),
	)

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

			grpcRule := fgw.L7RouteRule{}
			for _, hostname := range hostnames {
				r := c.generateGRPCRouteCfg(grpcRoute)

				for _, enricher := range hostnameEnrichers {
					enricher.Enrich(hostname, r)
				}

				grpcRule[hostname] = r
			}

			port := int32(listener.Port)
			if rule, exists := c.rules[port]; exists {
				if l7Rule, ok := rule.(fgw.L7RouteRule); ok {
					c.rules[port] = mergeL7RouteRule(l7Rule, grpcRule)
				}
			} else {
				c.rules[port] = grpcRule
			}
		}
	}
}

func (c *GatewayProcessor) generateGRPCRouteCfg(grpcRoute *gwv1.GRPCRoute) *fgw.GRPCRouteRuleSpec {
	grpcSpec := &fgw.GRPCRouteRuleSpec{
		RouteType: fgw.L7RouteTypeGRPC,
		Matches:   make([]fgw.GRPCTrafficMatch, 0),
	}
	enrichers := c.getGRPCRoutePolicyEnrichers(grpcRoute)

	for _, rule := range grpcRoute.Spec.Rules {
		backends := map[string]fgw.BackendServiceConfig{}

		for _, bk := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(grpcRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
				svcLevelFilters := make([]fgw.Filter, 0)
				for _, filter := range bk.Filters {
					svcLevelFilters = append(svcLevelFilters, c.toFSMGRPCRouteFilter(grpcRoute, filter))
				}

				backends[svcPort.String()] = fgw.BackendServiceConfig{
					Weight:  backendWeight(bk.BackendRef),
					Filters: svcLevelFilters,
				}

				c.services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		}

		ruleLevelFilters := make([]fgw.Filter, 0)
		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters = append(ruleLevelFilters, c.toFSMGRPCRouteFilter(grpcRoute, ruleFilter))
		}

		for _, m := range rule.Matches {
			match := &fgw.GRPCTrafficMatch{
				BackendService: backends,
				Filters:        ruleLevelFilters,
			}

			if m.Method != nil {
				match.Method = &fgw.GRPCMethod{
					MatchType: grpcMethodMatchType(m.Method.Type),
					Service:   m.Method.Service,
					Method:    m.Method.Method,
				}
			}

			if len(m.Headers) > 0 {
				match.Headers = grpcMatchHeaders(m)
			}

			for _, enricher := range enrichers {
				enricher.Enrich(m, match)
			}

			grpcSpec.Matches = append(grpcSpec.Matches, *match)
		}
	}

	grpcSpec.Sort()
	return grpcSpec
}
