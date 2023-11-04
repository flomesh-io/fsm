package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func processGRPCRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, grpcRoute *gwv1alpha2.GRPCRoute, policies globalPolicyAttachments, rules map[int32]routecfg.RouteRule, services map[string]serviceInfo) {
	routePolicies := filterPoliciesByRoute(policies, grpcRoute)
	hostnameEnrichers := getHostnamePolicyEnrichers(routePolicies)

	for _, ref := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, grpcRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, grpcRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			grpcRule := routecfg.L7RouteRule{}
			for _, hostname := range hostnames {
				r := generateGRPCRouteCfg(grpcRoute, routePolicies, services)

				for _, enricher := range hostnameEnrichers {
					enricher.Enrich(hostname, r)
				}

				grpcRule[hostname] = r
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(routecfg.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, grpcRule)
				}
			} else {
				rules[port] = grpcRule
			}
		}
	}
}

func generateGRPCRouteCfg(grpcRoute *gwv1alpha2.GRPCRoute, routePolicies routePolicies, services map[string]serviceInfo) routecfg.GRPCRouteRuleSpec {
	grpcSpec := routecfg.GRPCRouteRuleSpec{
		RouteType: routecfg.L7RouteTypeGRPC,
		Matches:   make([]routecfg.GRPCTrafficMatch, 0),
	}
	enrichers := getGRPCRoutePolicyEnrichers(routePolicies)

	for _, rule := range grpcRoute.Spec.Rules {
		backends := map[string]routecfg.BackendServiceConfig{}

		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendRef.BackendObjectReference, grpcRoute.Namespace); svcPort != nil {
				svcLevelFilters := make([]routecfg.Filter, 0)
				for _, filter := range bk.Filters {
					svcLevelFilters = append(svcLevelFilters, toFSMGRPCRouteFilter(filter, grpcRoute.Namespace, services))
				}

				backends[svcPort.String()] = routecfg.BackendServiceConfig{
					Weight:  backendWeight(bk.BackendRef),
					Filters: svcLevelFilters,
				}

				services[svcPort.String()] = serviceInfo{
					svcPortName: *svcPort,
				}
			}
		}

		ruleLevelFilters := make([]routecfg.Filter, 0)
		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters = append(ruleLevelFilters, toFSMGRPCRouteFilter(ruleFilter, grpcRoute.Namespace, services))
		}

		for _, m := range rule.Matches {
			match := routecfg.GRPCTrafficMatch{
				BackendService: backends,
				Filters:        ruleLevelFilters,
			}

			if m.Method != nil {
				match.Method = &routecfg.GRPCMethod{
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

			grpcSpec.Matches = append(grpcSpec.Matches, match)
		}
	}

	return grpcSpec
}
