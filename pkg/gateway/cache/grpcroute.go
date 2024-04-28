package cache

import (
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) processGRPCRoute(gw *gwv1.Gateway, validListeners []gwtypes.Listener, grpcRoute *gwv1alpha2.GRPCRoute, policies globalPolicyAttachments, rules map[int32]fgw.RouteRule, services map[string]serviceInfo) {
	referenceGrants := c.getResourcesFromCache(informers.ReferenceGrantResourceType, false)
	routePolicies := filterPoliciesByRoute(referenceGrants, policies, grpcRoute)
	hostnameEnrichers := getHostnamePolicyEnrichers(routePolicies)

	for _, ref := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.informers.GetListers().Namespace, gw, ref, gwutils.ToRouteContext(grpcRoute), validListeners)
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
				r := c.generateGRPCRouteCfg(grpcRoute, routePolicies, services)

				for _, enricher := range hostnameEnrichers {
					enricher.Enrich(hostname, r)
				}

				grpcRule[hostname] = r
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(fgw.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, grpcRule)
				}
			} else {
				rules[port] = grpcRule
			}
		}
	}
}

func (c *GatewayCache) generateGRPCRouteCfg(grpcRoute *gwv1alpha2.GRPCRoute, routePolicies routePolicies, services map[string]serviceInfo) *fgw.GRPCRouteRuleSpec {
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
					svcLevelFilters = append(svcLevelFilters, c.toFSMGRPCRouteFilter(grpcRoute, filter, services))
				}

				backends[svcPort.String()] = fgw.BackendServiceConfig{
					Weight:  backendWeight(bk.BackendRef),
					Filters: svcLevelFilters,
				}

				services[svcPort.String()] = serviceInfo{
					svcPortName: *svcPort,
				}
			}
		}

		ruleLevelFilters := make([]fgw.Filter, 0)
		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters = append(ruleLevelFilters, c.toFSMGRPCRouteFilter(grpcRoute, ruleFilter, services))
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
