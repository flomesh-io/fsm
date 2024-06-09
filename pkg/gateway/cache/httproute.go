package cache

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/status/route"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayProcessor) processHTTPRoutes() {
	list := &gwv1.HTTPRouteList{}
	err := c.cache.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayHTTPRouteIndex, client.ObjectKeyFromObject(c.gateway).String()),
	})
	if err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return
	}

	for _, httpRoute := range gwutils.SortResources(gwutils.ToSlicePtr(list.Items)) {
		c.processHTTPRoute(httpRoute)
	}
}

func (c *GatewayProcessor) processHTTPRoute(httpRoute *gwv1.HTTPRoute) {
	hostnameEnrichers := c.getHostnamePolicyEnrichers(httpRoute)

	rsh := route.NewRouteStatusHolder(
		httpRoute,
		&httpRoute.ObjectMeta,
		&httpRoute.TypeMeta,
		httpRoute.Spec.Hostnames,
		gwutils.ToSlicePtr(httpRoute.Status.Parents),
	)

	for _, parentRef := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(parentRef, client.ObjectKeyFromObject(c.gateway)) {
			continue
		}

		h := rsh.StatusUpdateFor(parentRef)

		allowedListeners := gwutils.GetAllowedListeners(c.cache.client, c.gateway, h)
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
				r := c.generateHTTPRouteConfig(httpRoute)

				for _, enricher := range hostnameEnrichers {
					enricher.Enrich(hostname, r)
				}

				httpRule[hostname] = r
			}

			port := int32(listener.Port)
			if rule, exists := c.rules[port]; exists {
				if l7Rule, ok := rule.(fgw.L7RouteRule); ok {
					c.rules[port] = mergeL7RouteRule(l7Rule, httpRule)
				}
			} else {
				c.rules[port] = httpRule
			}
		}
	}
}

func (c *GatewayProcessor) generateHTTPRouteConfig(httpRoute *gwv1.HTTPRoute) *fgw.HTTPRouteRuleSpec {
	httpSpec := &fgw.HTTPRouteRuleSpec{
		RouteType: fgw.L7RouteTypeHTTP,
		Matches:   make([]fgw.HTTPTrafficMatch, 0),
	}
	enrichers := c.getHTTPRoutePolicyEnrichers(httpRoute)

	for _, rule := range httpRoute.Spec.Rules {
		backends := map[string]fgw.BackendServiceConfig{}

		for _, bk := range rule.BackendRefs {
			if svcPort := c.backendRefToServicePortName(httpRoute, bk.BackendRef.BackendObjectReference); svcPort != nil {
				log.Debug().Msgf("Found svcPort: %v", svcPort)
				svcLevelFilters := make([]fgw.Filter, 0)
				for _, filter := range bk.Filters {
					svcLevelFilters = append(svcLevelFilters, c.toFSMHTTPRouteFilter(httpRoute, filter))
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
			ruleLevelFilters = append(ruleLevelFilters, c.toFSMHTTPRouteFilter(httpRoute, ruleFilter))
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

			if rule.Timeouts != nil {
				match.Timeouts = &fgw.HTTPRouteTimeouts{
					Request:        rule.Timeouts.Request,
					BackendRequest: rule.Timeouts.BackendRequest,
				}
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
