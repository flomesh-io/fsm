package cache

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessor) processGRPCRoute(grpcRoute *gwv1.GRPCRoute) {
	routePolicies := filterPoliciesByRoute(c.referenceGrants, c.policies, grpcRoute)
	hostnameEnrichers := getHostnamePolicyEnrichers(routePolicies)

	for _, ref := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(c.gateway)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.getNamespaceLister(), c.gateway, ref, gwutils.ToRouteContext(grpcRoute), c.validListeners)
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
				r := c.generateGRPCRouteCfg(grpcRoute, routePolicies)

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

func (c *GatewayProcessor) generateGRPCRouteCfg(grpcRoute *gwv1.GRPCRoute, routePolicies routePolicies) *fgw.GRPCRouteRuleSpec {
	grpcSpec := &fgw.GRPCRouteRuleSpec{
		RouteType: fgw.L7RouteTypeGRPC,
		Matches:   make([]fgw.GRPCTrafficMatch, 0),
	}
	enrichers := getGRPCRoutePolicyEnrichers(routePolicies)

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
