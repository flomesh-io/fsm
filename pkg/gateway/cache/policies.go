package cache

import (
	"sort"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/retry"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/sessionsticky"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/loadbalancer"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/circuitbreaking"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/healthcheck"
	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/upstreamtls"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/gateway/policy"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) policyAttachments() globalPolicyAttachments {
	return globalPolicyAttachments{
		rateLimits:      c.rateLimits(),
		accessControls:  c.accessControls(),
		faultInjections: c.faultInjections(),
	}
}

func (c *GatewayCache) getPortPolicyEnrichers(policies globalPolicyAttachments) []policy.PortPolicyEnricher {
	return []policy.PortPolicyEnricher{
		&policy.RateLimitPortEnricher{Data: policies.rateLimits[RateLimitPolicyMatchTypePort]},
		&policy.AccessControlPortEnricher{Data: policies.accessControls[AccessControlPolicyMatchTypePort]},
		&policy.GatewayTLSPortEnricher{Data: c.gatewayTLS()},
	}
}

func getHostnamePolicyEnrichers(routePolicies routePolicies) []policy.HostnamePolicyEnricher {
	return []policy.HostnamePolicyEnricher{
		&policy.RateLimitHostnameEnricher{Data: routePolicies.hostnamesRateLimits},
		&policy.AccessControlHostnameEnricher{Data: routePolicies.hostnamesAccessControls},
		&policy.FaultInjectionHostnameEnricher{Data: routePolicies.hostnamesFaultInjections},
	}
}

func getHTTPRoutePolicyEnrichers(routePolicies routePolicies) []policy.HTTPRoutePolicyEnricher {
	return []policy.HTTPRoutePolicyEnricher{
		&policy.RateLimitHTTPRouteEnricher{Data: routePolicies.httpRouteRateLimits},
		&policy.AccessControlHTTPRouteEnricher{Data: routePolicies.httpRouteAccessControls},
		&policy.FaultInjectionHTTPRouteEnricher{Data: routePolicies.httpRouteFaultInjections},
	}
}

func getGRPCRoutePolicyEnrichers(routePolicies routePolicies) []policy.GRPCRoutePolicyEnricher {
	return []policy.GRPCRoutePolicyEnricher{
		&policy.RateLimitGRPCRouteEnricher{Data: routePolicies.grpcRouteRateLimits},
		&policy.AccessControlGRPCRouteEnricher{Data: routePolicies.grpcRouteAccessControls},
		&policy.FaultInjectionGRPCRouteEnricher{Data: routePolicies.grpcRouteFaultInjections},
	}
}

func (c *GatewayCache) getServicePolicyEnrichers() []policy.ServicePolicyEnricher {
	return []policy.ServicePolicyEnricher{
		&policy.SessionStickyPolicyEnricher{Data: c.sessionStickies()},
		&policy.LoadBalancerPolicyEnricher{Data: c.loadBalancers()},
		&policy.CircuitBreakingPolicyEnricher{Data: c.circuitBreakings()},
		&policy.HealthCheckPolicyEnricher{Data: c.healthChecks()},
		&policy.UpstreamTLSPolicyEnricher{Data: c.upstreamTLS()},
		&policy.RetryPolicyEnricher{Data: c.retryConfigs()},
	}
}

func (c *GatewayCache) rateLimits() map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy {
	rateLimits := make(map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy)
	for _, matchType := range []RateLimitPolicyMatchType{
		RateLimitPolicyMatchTypePort,
		RateLimitPolicyMatchTypeHostnames,
		RateLimitPolicyMatchTypeHTTPRoute,
		RateLimitPolicyMatchTypeGRPCRoute,
	} {
		rateLimits[matchType] = make([]gwpav1alpha1.RateLimitPolicy, 0)
	}

	for key := range c.ratelimits {
		p, err := c.getRateLimitPolicyFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get RateLimitPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
			spec := p.Spec
			targetRef := spec.TargetRef

			switch {
			case gwutils.IsTargetRefToGVK(targetRef, gatewayGVK) && len(spec.Ports) > 0:
				rateLimits[RateLimitPolicyMatchTypePort] = append(rateLimits[RateLimitPolicyMatchTypePort], *p)
			case (gwutils.IsTargetRefToGVK(targetRef, httpRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, grpcRouteGVK)) && len(spec.Hostnames) > 0:
				rateLimits[RateLimitPolicyMatchTypeHostnames] = append(rateLimits[RateLimitPolicyMatchTypeHostnames], *p)
			case gwutils.IsTargetRefToGVK(targetRef, httpRouteGVK) && len(spec.HTTPRateLimits) > 0:
				rateLimits[RateLimitPolicyMatchTypeHTTPRoute] = append(rateLimits[RateLimitPolicyMatchTypeHTTPRoute], *p)
			case gwutils.IsTargetRefToGVK(targetRef, grpcRouteGVK) && len(spec.GRPCRateLimits) > 0:
				rateLimits[RateLimitPolicyMatchTypeGRPCRoute] = append(rateLimits[RateLimitPolicyMatchTypeGRPCRoute], *p)
			}
		}
	}

	// sort each type of rate limits by creation timestamp
	for matchType, policies := range rateLimits {
		sort.Slice(policies, func(i, j int) bool {
			if policies[i].CreationTimestamp.Time.Equal(policies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&policies[i]).String() < client.ObjectKeyFromObject(&policies[j]).String()
			}

			return policies[i].CreationTimestamp.Time.Before(policies[j].CreationTimestamp.Time)
		})
		rateLimits[matchType] = policies
	}

	return rateLimits
}

func (c *GatewayCache) accessControls() map[AccessControlPolicyMatchType][]gwpav1alpha1.AccessControlPolicy {
	accessControls := make(map[AccessControlPolicyMatchType][]gwpav1alpha1.AccessControlPolicy)
	for _, matchType := range []AccessControlPolicyMatchType{
		AccessControlPolicyMatchTypePort,
		AccessControlPolicyMatchTypeHostnames,
		AccessControlPolicyMatchTypeHTTPRoute,
		AccessControlPolicyMatchTypeGRPCRoute,
	} {
		accessControls[matchType] = make([]gwpav1alpha1.AccessControlPolicy, 0)
	}

	for key := range c.accesscontrols {
		p, err := c.getAccessControlPolicyFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get AccessControlPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
			spec := p.Spec
			targetRef := spec.TargetRef

			switch {
			case gwutils.IsTargetRefToGVK(targetRef, gatewayGVK) && len(spec.Ports) > 0:
				accessControls[AccessControlPolicyMatchTypePort] = append(accessControls[AccessControlPolicyMatchTypePort], *p)
			case (gwutils.IsTargetRefToGVK(targetRef, httpRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, grpcRouteGVK)) && len(spec.Hostnames) > 0:
				accessControls[AccessControlPolicyMatchTypeHostnames] = append(accessControls[AccessControlPolicyMatchTypeHostnames], *p)
			case gwutils.IsTargetRefToGVK(targetRef, httpRouteGVK) && len(spec.HTTPAccessControls) > 0:
				accessControls[AccessControlPolicyMatchTypeHTTPRoute] = append(accessControls[AccessControlPolicyMatchTypeHTTPRoute], *p)
			case gwutils.IsTargetRefToGVK(targetRef, grpcRouteGVK) && len(spec.GRPCAccessControls) > 0:
				accessControls[AccessControlPolicyMatchTypeGRPCRoute] = append(accessControls[AccessControlPolicyMatchTypeGRPCRoute], *p)
			}
		}
	}

	// sort each type of access controls by creation timestamp
	for matchType, policies := range accessControls {
		sort.Slice(policies, func(i, j int) bool {
			if policies[i].CreationTimestamp.Time.Equal(policies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&policies[i]).String() < client.ObjectKeyFromObject(&policies[j]).String()
			}

			return policies[i].CreationTimestamp.Time.Before(policies[j].CreationTimestamp.Time)
		})
		accessControls[matchType] = policies
	}

	return accessControls
}

func (c *GatewayCache) faultInjections() map[FaultInjectionPolicyMatchType][]gwpav1alpha1.FaultInjectionPolicy {
	faultInjections := make(map[FaultInjectionPolicyMatchType][]gwpav1alpha1.FaultInjectionPolicy)
	for _, matchType := range []FaultInjectionPolicyMatchType{
		FaultInjectionPolicyMatchTypeHostnames,
		FaultInjectionPolicyMatchTypeHTTPRoute,
		FaultInjectionPolicyMatchTypeGRPCRoute,
	} {
		faultInjections[matchType] = make([]gwpav1alpha1.FaultInjectionPolicy, 0)
	}

	for key := range c.faultinjections {
		p, err := c.getFaultInjectionPolicyFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get FaultInjectionPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
			spec := p.Spec
			targetRef := spec.TargetRef

			switch {
			case (gwutils.IsTargetRefToGVK(targetRef, httpRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, grpcRouteGVK)) && len(spec.Hostnames) > 0:
				faultInjections[FaultInjectionPolicyMatchTypeHostnames] = append(faultInjections[FaultInjectionPolicyMatchTypeHostnames], *p)
			case gwutils.IsTargetRefToGVK(targetRef, httpRouteGVK) && len(spec.HTTPFaultInjections) > 0:
				faultInjections[FaultInjectionPolicyMatchTypeHTTPRoute] = append(faultInjections[FaultInjectionPolicyMatchTypeHTTPRoute], *p)
			case gwutils.IsTargetRefToGVK(targetRef, grpcRouteGVK) && len(spec.GRPCFaultInjections) > 0:
				faultInjections[FaultInjectionPolicyMatchTypeGRPCRoute] = append(faultInjections[FaultInjectionPolicyMatchTypeGRPCRoute], *p)
			}
		}
	}

	// sort each type of fault injections by creation timestamp
	for matchType, policies := range faultInjections {
		sort.Slice(policies, func(i, j int) bool {
			if policies[i].CreationTimestamp.Time.Equal(policies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&policies[i]).String() < client.ObjectKeyFromObject(&policies[j]).String()
			}

			return policies[i].CreationTimestamp.Time.Before(policies[j].CreationTimestamp.Time)
		})
		faultInjections[matchType] = policies
	}

	return faultInjections
}

func filterPoliciesByRoute(policies globalPolicyAttachments, route client.Object) routePolicies {
	result := routePolicies{
		hostnamesRateLimits:      make([]gwpav1alpha1.RateLimitPolicy, 0),
		httpRouteRateLimits:      make([]gwpav1alpha1.RateLimitPolicy, 0),
		grpcRouteRateLimits:      make([]gwpav1alpha1.RateLimitPolicy, 0),
		hostnamesAccessControls:  make([]gwpav1alpha1.AccessControlPolicy, 0),
		httpRouteAccessControls:  make([]gwpav1alpha1.AccessControlPolicy, 0),
		grpcRouteAccessControls:  make([]gwpav1alpha1.AccessControlPolicy, 0),
		hostnamesFaultInjections: make([]gwpav1alpha1.FaultInjectionPolicy, 0),
		httpRouteFaultInjections: make([]gwpav1alpha1.FaultInjectionPolicy, 0),
		grpcRouteFaultInjections: make([]gwpav1alpha1.FaultInjectionPolicy, 0),
	}

	if len(policies.rateLimits[RateLimitPolicyMatchTypeHostnames]) > 0 {
		for _, rateLimit := range policies.rateLimits[RateLimitPolicyMatchTypeHostnames] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, route) {
				result.hostnamesRateLimits = append(result.hostnamesRateLimits, rateLimit)
			}
		}
	}

	if len(policies.rateLimits[RateLimitPolicyMatchTypeHTTPRoute]) > 0 {
		for _, rateLimit := range policies.rateLimits[RateLimitPolicyMatchTypeHTTPRoute] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, route) {
				result.httpRouteRateLimits = append(result.httpRouteRateLimits, rateLimit)
			}
		}
	}

	if len(policies.rateLimits[RateLimitPolicyMatchTypeGRPCRoute]) > 0 {
		for _, rateLimit := range policies.rateLimits[RateLimitPolicyMatchTypeGRPCRoute] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, route) {
				result.grpcRouteRateLimits = append(result.grpcRouteRateLimits, rateLimit)
			}
		}
	}

	if len(policies.accessControls[AccessControlPolicyMatchTypeHostnames]) > 0 {
		for _, ac := range policies.accessControls[AccessControlPolicyMatchTypeHostnames] {
			if gwutils.IsRefToTarget(ac.Spec.TargetRef, route) {
				result.hostnamesAccessControls = append(result.hostnamesAccessControls, ac)
			}
		}
	}

	if len(policies.accessControls[AccessControlPolicyMatchTypeHTTPRoute]) > 0 {
		for _, ac := range policies.accessControls[AccessControlPolicyMatchTypeHTTPRoute] {
			if gwutils.IsRefToTarget(ac.Spec.TargetRef, route) {
				result.httpRouteAccessControls = append(result.httpRouteAccessControls, ac)
			}
		}
	}

	if len(policies.accessControls[AccessControlPolicyMatchTypeGRPCRoute]) > 0 {
		for _, ac := range policies.accessControls[AccessControlPolicyMatchTypeGRPCRoute] {
			if gwutils.IsRefToTarget(ac.Spec.TargetRef, route) {
				result.grpcRouteAccessControls = append(result.grpcRouteAccessControls, ac)
			}
		}
	}

	if len(policies.faultInjections[FaultInjectionPolicyMatchTypeHostnames]) > 0 {
		for _, fj := range policies.faultInjections[FaultInjectionPolicyMatchTypeHostnames] {
			if gwutils.IsRefToTarget(fj.Spec.TargetRef, route) {
				result.hostnamesFaultInjections = append(result.hostnamesFaultInjections, fj)
			}
		}
	}

	if len(policies.faultInjections[FaultInjectionPolicyMatchTypeHTTPRoute]) > 0 {
		for _, fj := range policies.faultInjections[FaultInjectionPolicyMatchTypeHTTPRoute] {
			if gwutils.IsRefToTarget(fj.Spec.TargetRef, route) {
				result.httpRouteFaultInjections = append(result.httpRouteFaultInjections, fj)
			}
		}
	}

	if len(policies.faultInjections[FaultInjectionPolicyMatchTypeGRPCRoute]) > 0 {
		for _, fj := range policies.faultInjections[FaultInjectionPolicyMatchTypeGRPCRoute] {
			if gwutils.IsRefToTarget(fj.Spec.TargetRef, route) {
				result.grpcRouteFaultInjections = append(result.grpcRouteFaultInjections, fj)
			}
		}
	}

	return result
}

func (c *GatewayCache) sessionStickies() map[string]*gwpav1alpha1.SessionStickyConfig {
	sessionStickies := make(map[string]*gwpav1alpha1.SessionStickyConfig)

	for key := range c.sessionstickies {
		sessionSticky, err := c.getSessionStickyPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get SessionStickyPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(sessionSticky.Status.Conditions) {
			for _, p := range sessionSticky.Spec.Ports {
				if svcPortName := targetRefToServicePortName(sessionSticky.Spec.TargetRef, sessionSticky.Namespace, int32(p.Port)); svcPortName != nil {
					cfg := sessionsticky.ComputeSessionStickyConfig(p.Config, sessionSticky.Spec.DefaultConfig)

					if cfg == nil {
						continue
					}

					sessionStickies[svcPortName.String()] = cfg
				}
			}
		}
	}

	return sessionStickies
}

func (c *GatewayCache) loadBalancers() map[string]*gwpav1alpha1.LoadBalancerType {
	loadBalancers := make(map[string]*gwpav1alpha1.LoadBalancerType)

	for key := range c.loadbalancers {
		lb, err := c.getLoadBalancerPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get LoadBalancerPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(lb.Status.Conditions) {
			for _, p := range lb.Spec.Ports {
				if svcPortName := targetRefToServicePortName(lb.Spec.TargetRef, lb.Namespace, int32(p.Port)); svcPortName != nil {
					t := loadbalancer.ComputeLoadBalancerType(p.Type, lb.Spec.DefaultType)

					if t == nil {
						continue
					}

					loadBalancers[svcPortName.String()] = t
				}
			}
		}
	}

	return loadBalancers
}

func (c *GatewayCache) circuitBreakings() map[string]*gwpav1alpha1.CircuitBreakingConfig {
	configs := make(map[string]*gwpav1alpha1.CircuitBreakingConfig)

	for key := range c.circuitbreakings {
		circuitBreaking, err := c.getCircuitBreakingPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get CircuitBreakingPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(circuitBreaking.Status.Conditions) {
			for _, p := range circuitBreaking.Spec.Ports {
				if svcPortName := targetRefToServicePortName(circuitBreaking.Spec.TargetRef, circuitBreaking.Namespace, int32(p.Port)); svcPortName != nil {
					cfg := circuitbreaking.ComputeCircuitBreakingConfig(p.Config, circuitBreaking.Spec.DefaultConfig)

					if cfg == nil {
						continue
					}

					configs[svcPortName.String()] = cfg
				}
			}
		}
	}

	return configs
}

func (c *GatewayCache) healthChecks() map[string]*gwpav1alpha1.HealthCheckConfig {
	configs := make(map[string]*gwpav1alpha1.HealthCheckConfig)

	for key := range c.healthchecks {
		healthCheck, err := c.getHealthCheckPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get HealthCheckPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(healthCheck.Status.Conditions) {
			for _, p := range healthCheck.Spec.Ports {
				if svcPortName := targetRefToServicePortName(healthCheck.Spec.TargetRef, healthCheck.Namespace, int32(p.Port)); svcPortName != nil {
					cfg := healthcheck.ComputeHealthCheckConfig(p.Config, healthCheck.Spec.DefaultConfig)

					if cfg == nil {
						continue
					}

					configs[svcPortName.String()] = cfg
				}
			}
		}
	}

	return configs
}

func (c *GatewayCache) upstreamTLS() map[string]*policy.UpstreamTLSConfig {
	configs := make(map[string]*policy.UpstreamTLSConfig)

	for key := range c.upstreamstls {
		upstreamTLS, err := c.getUpstreamTLSPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get UpstreamTLSPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(upstreamTLS.Status.Conditions) {
			for _, p := range upstreamTLS.Spec.Ports {
				if svcPortName := targetRefToServicePortName(upstreamTLS.Spec.TargetRef, upstreamTLS.Namespace, int32(p.Port)); svcPortName != nil {
					cfg := upstreamtls.ComputeUpstreamTLSConfig(p.Config, upstreamTLS.Spec.DefaultConfig)

					if cfg == nil {
						continue
					}

					if string(*cfg.CertificateRef.Group) != constants.KubernetesCoreGroup {
						continue
					}

					if string(*cfg.CertificateRef.Kind) != constants.KubernetesSecretKind {
						continue
					}

					secretKey := client.ObjectKey{
						Namespace: gwutils.Namespace(cfg.CertificateRef.Namespace, upstreamTLS.Namespace),
						Name:      string(cfg.CertificateRef.Name),
					}

					secret, err := c.getSecretFromCache(secretKey)
					if err != nil {
						log.Error().Msgf("Failed to get Secret %s: %s", secretKey, err)
						continue
					}

					configs[svcPortName.String()] = &policy.UpstreamTLSConfig{
						MTLS:   cfg.MTLS,
						Secret: secret,
					}
				}
			}
		}
	}

	return configs
}

func (c *GatewayCache) retryConfigs() map[string]*gwpav1alpha1.RetryConfig {
	configs := make(map[string]*gwpav1alpha1.RetryConfig)

	for key := range c.retries {
		retryPolicy, err := c.getRetryPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get RetryPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedPolicyAttachment(retryPolicy.Status.Conditions) {
			for _, p := range retryPolicy.Spec.Ports {
				if svcPortName := targetRefToServicePortName(retryPolicy.Spec.TargetRef, retryPolicy.Namespace, int32(p.Port)); svcPortName != nil {
					cfg := retry.ComputeRetryConfig(p.Config, retryPolicy.Spec.DefaultConfig)

					if cfg == nil {
						continue
					}

					configs[svcPortName.String()] = cfg
				}
			}
		}
	}

	return configs
}

func (c *GatewayCache) gatewayTLS() []gwpav1alpha1.GatewayTLSPolicy {
	policies := make([]gwpav1alpha1.GatewayTLSPolicy, 0)

	for key := range c.gatewaytls {
		gatewayTLSPolicy, err := c.getGatewayTLSPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get GatewayTLSPolicy %s: %s", key, err)
			continue
		}

		if !gwutils.IsAcceptedPolicyAttachment(gatewayTLSPolicy.Status.Conditions) {
			continue
		}

		if !gwutils.IsTargetRefToGVK(gatewayTLSPolicy.Spec.TargetRef, gatewayGVK) {
			continue
		}

		if len(gatewayTLSPolicy.Spec.Ports) == 0 {
			continue
		}

		policies = append(policies, *gatewayTLSPolicy)
	}

	return policies
}
