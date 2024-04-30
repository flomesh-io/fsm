package cache

import (
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) processHTTPRoute(gw *gwv1.Gateway, validListeners []gwtypes.Listener, httpRoute *gwv1.HTTPRoute, policies globalPolicyAttachments, rules map[int32]fgw.RouteRule, services map[string]serviceContext) {
	referenceGrants := c.getResourcesFromCache(informers.ReferenceGrantResourceType, false)
	routePolicies := filterPoliciesByRoute(referenceGrants, policies, httpRoute)
	log.Debug().Msgf("[GW-CACHE] routePolicies: %v", routePolicies)
	hostnameEnrichers := getHostnamePolicyEnrichers(routePolicies)

	for _, ref := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners, _ := gwutils.GetAllowedListeners(c.informers.GetListers().Namespace, gw, ref, gwutils.ToRouteContext(httpRoute), validListeners)
		log.Debug().Msgf("allowedListeners: %v", allowedListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, httpRoute.Spec.Hostnames)
			log.Debug().Msgf("hostnames: %v", hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			httpRule := fgw.L7RouteRule{}
			for _, hostname := range hostnames {
				r := c.generateHTTPRouteConfig(httpRoute, routePolicies, services)

				for _, enricher := range hostnameEnrichers {
					enricher.Enrich(hostname, r)
				}

				httpRule[hostname] = r
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(fgw.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, httpRule)
				}
			} else {
				rules[port] = httpRule
			}
		}
	}
}

func (c *GatewayCache) generateHTTPRouteConfig(httpRoute *gwv1.HTTPRoute, routePolicies routePolicies, services map[string]serviceContext) *fgw.HTTPRouteRuleSpec {
	httpSpec := &fgw.HTTPRouteRuleSpec{
		RouteType: fgw.L7RouteTypeHTTP,
		Matches:   make([]fgw.HTTPTrafficMatch, 0),
	}
	enrichers := getHTTPRoutePolicyEnrichers(routePolicies)

	for _, rule := range httpRoute.Spec.Rules {
		backends := map[string]fgw.BackendServiceConfig{}

		for _, bk := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(httpRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
				log.Debug().Msgf("Found svcPort: %v", svcPort)
				svcLevelFilters := make([]fgw.Filter, 0)
				for _, filter := range bk.Filters {
					svcLevelFilters = append(svcLevelFilters, c.toFSMHTTPRouteFilter(httpRoute, filter, services))
				}

				backends[svcPort.String()] = fgw.BackendServiceConfig{
					Weight:  backendWeight(bk.BackendRef),
					Filters: svcLevelFilters,
				}

				services[svcPort.String()] = serviceContext{
					svcPortName: *svcPort,
				}
			}
		}

		ruleLevelFilters := make([]fgw.Filter, 0)
		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters = append(ruleLevelFilters, c.toFSMHTTPRouteFilter(httpRoute, ruleFilter, services))
		}

		for _, m := range rule.Matches {
			match := &fgw.HTTPTrafficMatch{
				BackendService: backends,
				Filters:        ruleLevelFilters,
			}

			if m.Path != nil {
				match.Path = &fgw.Path{
					MatchType: httpPathMatchType(m.Path.Type),
					Path:      httpPath(m.Path.Value),
				}
			}

			if m.Method != nil {
				match.Methods = []string{string(*m.Method)}
			}

			if len(m.Headers) > 0 {
				match.Headers = httpMatchHeaders(m)
			}

			if len(m.QueryParams) > 0 {
				match.RequestParams = httpMatchQueryParams(m)
			}

			for _, enricher := range enrichers {
				enricher.Enrich(m, match)
			}

			httpSpec.Matches = append(httpSpec.Matches, *match)
		}
	}

	httpSpec.Sort()
	return httpSpec
}
