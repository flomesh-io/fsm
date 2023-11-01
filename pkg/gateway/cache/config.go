package cache

import (
	"fmt"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// BuildConfigs builds the configs for all the gateways in the cache
func (c *GatewayCache) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	configs := make(map[string]*routecfg.ConfigSpec)

	for ns, key := range c.gateways {
		gw, err := c.getGatewayFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
			continue
		}

		validListeners := gwutils.GetValidListenersFromGateway(gw)
		log.Debug().Msgf("[GW-CACHE] validListeners: %v", validListeners)
		acceptedRateLimits := c.rateLimits()

		listenerCfg := c.listeners(gw, validListeners, acceptedRateLimits)
		rules, referredServices := c.routeRules(gw, validListeners, acceptedRateLimits)
		svcConfigs := c.serviceConfigs(referredServices)

		configSpec := &routecfg.ConfigSpec{
			Defaults:   c.defaults(),
			Listeners:  listenerCfg,
			RouteRules: rules,
			Services:   svcConfigs,
			Chains:     c.chains(),
		}
		configSpec.Version = utils.SimpleHash(configSpec)
		configs[ns] = configSpec
	}

	for ns, cfg := range configs {
		gatewayPath := utils.GatewayCodebasePath(ns)
		if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
			continue
		}

		jsonVersion, err := c.getVersionOfConfigJSON(gatewayPath)
		if err != nil {
			continue
		}

		if jsonVersion == cfg.Version {
			// config not changed, ignore updating
			log.Debug().Msgf("%s/config.json doesn't change, ignore updating...", gatewayPath)
			continue
		}

		go func(cfg *routecfg.ConfigSpec) {
			//if err := c.repoClient.DeriveCodebase(gatewayPath, parentPath); err != nil {
			//	log.Error().Msgf("Gateway codebase %q failed to derive codebase %q: %s", gatewayPath, parentPath, err)
			//	return
			//}

			batches := []repo.Batch{
				{
					Basepath: gatewayPath,
					Items: []repo.BatchItem{
						{
							Path:     "",
							Filename: "config.json",
							Content:  cfg,
						},
					},
				},
			}

			if err := c.repoClient.Batch(batches); err != nil {
				log.Error().Msgf("Sync gateway config to repo failed: %s", err)
				return
			}
		}(cfg)
	}
}

func (c *GatewayCache) getVersionOfConfigJSON(basepath string) (string, error) {
	path := fmt.Sprintf("%s/config.json", basepath)

	json, err := c.repoClient.GetFile(path)
	if err != nil {
		log.Error().Msgf("Get %q from pipy repo error: %s", path, err)
		return "", err
	}

	version := gjson.Get(json, "Version").String()

	return version, nil
}

func (c *GatewayCache) defaults() routecfg.Defaults {
	return routecfg.Defaults{
		EnableDebug:                    c.isDebugEnabled(),
		DefaultPassthroughUpstreamPort: c.cfg.GetFGWSSLPassthroughUpstreamPort(),
		StripAnyHostPort:               c.cfg.IsFGWStripAnyHostPort(),
		HTTP1PerRequestLoadBalancing:   c.cfg.IsFGWHTTP1PerRequestLoadBalancingEnabled(),
		HTTP2PerRequestLoadBalancing:   c.cfg.IsFGWHTTP2PerRequestLoadBalancingEnabled(),
	}
}

func (c *GatewayCache) isDebugEnabled() bool {
	switch c.cfg.GetFGWLogLevel() {
	case "debug", "trace":
		return true
	default:
		return false
	}
}

func (c *GatewayCache) listeners(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, acceptedRateLimits map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy) []routecfg.Listener {
	listeners := make([]routecfg.Listener, 0)
	for _, l := range validListeners {
		listener := routecfg.Listener{
			Protocol: l.Protocol,
			Listen:   c.listenPort(l),
			Port:     l.Port,
		}

		if tls := c.tls(gw, l); tls != nil {
			listener.TLS = tls
		}

		l4RateLimits := acceptedRateLimits[RateLimitPolicyMatchTypePort]
		if len(l4RateLimits) > 0 {
			for _, rateLimit := range l4RateLimits {
				if !gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gw) {
					continue
				}

				if len(rateLimit.Spec.Ports) == 0 {
					continue
				}

				if r := gwutils.GetRateLimitIfPortMatchesPolicy(l.Port, rateLimit); r != nil && listener.BpsLimit == nil {
					listener.BpsLimit = r
				}
			}
		}

		listeners = append(listeners, listener)
	}

	return listeners
}

func (c *GatewayCache) rateLimits() map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy {
	rateLimits := make(map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy)
	for _, matchType := range []RateLimitPolicyMatchType{
		RateLimitPolicyMatchTypePort,
		RateLimitPolicyMatchTypeHostnames,
		RateLimitPolicyMatchTypeRoute,
	} {
		rateLimits[matchType] = make([]gwpav1alpha1.RateLimitPolicy, 0)
	}

	for key := range c.ratelimits {
		rateLimitPolicy, err := c.getRateLimitPolicyFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get RateLimitPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedRateLimitPolicy(rateLimitPolicy) {
			switch {
			case len(rateLimitPolicy.Spec.Ports) > 0:
				rateLimits[RateLimitPolicyMatchTypePort] = append(rateLimits[RateLimitPolicyMatchTypePort], *rateLimitPolicy)
			case len(rateLimitPolicy.Spec.Hostnames) > 0:
				rateLimits[RateLimitPolicyMatchTypeHostnames] = append(rateLimits[RateLimitPolicyMatchTypeHostnames], *rateLimitPolicy)
			case len(rateLimitPolicy.Spec.HTTPRateLimits) > 0 || len(rateLimitPolicy.Spec.GRPCRateLimits) > 0:
				rateLimits[RateLimitPolicyMatchTypeRoute] = append(rateLimits[RateLimitPolicyMatchTypeRoute], *rateLimitPolicy)
			}
		}
	}

	// sort each type of rate limits by creation timestamp
	for matchType, policies := range rateLimits {
		sort.Slice(policies, func(i, j int) bool {
			if policies[i].CreationTimestamp.Time.Equal(policies[j].CreationTimestamp.Time) {
				return policies[i].Name < policies[j].Name
			}

			return policies[i].CreationTimestamp.Time.Before(policies[j].CreationTimestamp.Time)
		})
		rateLimits[matchType] = policies
	}

	return rateLimits
}

func (c *GatewayCache) listenPort(l gwtypes.Listener) gwv1beta1.PortNumber {
	if l.Port < 1024 {
		return l.Port + 60000
	}

	return l.Port
}

func (c *GatewayCache) tls(gw *gwv1beta1.Gateway, l gwtypes.Listener) *routecfg.TLS {
	switch l.Protocol {
	case gwv1beta1.HTTPSProtocolType:
		// Terminate
		if l.TLS != nil {
			if l.TLS.Mode == nil || *l.TLS.Mode == gwv1beta1.TLSModeTerminate {
				return c.tlsTerminateCfg(gw, l)
			}
		}
	case gwv1beta1.TLSProtocolType:
		// Terminate & Passthrough
		if l.TLS != nil {
			if l.TLS.Mode == nil {
				return c.tlsTerminateCfg(gw, l)
			}

			switch *l.TLS.Mode {
			case gwv1beta1.TLSModeTerminate:
				return c.tlsTerminateCfg(gw, l)
			case gwv1beta1.TLSModePassthrough:
				return c.tlsPassthroughCfg()
			}
		}
	}

	return nil
}

func (c *GatewayCache) tlsTerminateCfg(gw *gwv1beta1.Gateway, l gwtypes.Listener) *routecfg.TLS {
	return &routecfg.TLS{
		TLSModeType:  gwv1beta1.TLSModeTerminate,
		MTLS:         isMTLSEnabled(gw),
		Certificates: c.certificates(gw, l),
	}
}

func (c *GatewayCache) tlsPassthroughCfg() *routecfg.TLS {
	return &routecfg.TLS{
		TLSModeType: gwv1beta1.TLSModePassthrough,
		MTLS:        false,
	}
}

func (c *GatewayCache) certificates(gw *gwv1beta1.Gateway, l gwtypes.Listener) []routecfg.Certificate {
	certs := make([]routecfg.Certificate, 0)
	for _, ref := range l.TLS.CertificateRefs {
		if string(*ref.Kind) == constants.KubernetesSecretKind && string(*ref.Group) == constants.KubernetesCoreGroup {
			ns := getSecretRefNamespace(gw, ref)
			name := string(ref.Name)
			secret, err := c.getSecretFromCache(client.ObjectKey{Namespace: ns, Name: name})

			if err != nil {
				log.Error().Msgf("Failed to get Secret %s/%s: %s", ns, name, err)
				continue
			}

			cert := routecfg.Certificate{
				CertChain:  string(secret.Data[corev1.TLSCertKey]),
				PrivateKey: string(secret.Data[corev1.TLSPrivateKeyKey]),
			}

			ca := string(secret.Data[corev1.ServiceAccountRootCAKey])
			if len(ca) > 0 {
				cert.IssuingCA = ca
			}

			certs = append(certs, cert)
		}
	}
	return certs
}

func (c *GatewayCache) routeRules(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, acceptedRateLimits map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy) (map[int32]routecfg.RouteRule, map[string]serviceInfo) {
	rules := make(map[int32]routecfg.RouteRule)
	services := make(map[string]serviceInfo)

	log.Debug().Msgf("Processing %d HTTPRoutes", len(c.httproutes))
	for key := range c.httproutes {
		httpRoute, err := c.getHTTPRouteFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get HTTPRoute %s: %s", key, err)
			continue
		}

		log.Debug().Msgf("Processing HTTPRoute %v", httpRoute)
		processHTTPRoute(gw, validListeners, httpRoute, acceptedRateLimits, rules, services)
	}

	log.Debug().Msgf("Processing %d GRPCRoutes", len(c.grpcroutes))
	for key := range c.grpcroutes {
		grpcRoute, err := c.getGRPCRouteFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get GRPCRoute %s: %s", key, err)
			continue
		}

		processGRPCRoute(gw, validListeners, grpcRoute, acceptedRateLimits, rules, services)
	}

	log.Debug().Msgf("Processing %d TLSRoutes", len(c.tlsroutes))
	for key := range c.tlsroutes {
		tlsRoute, err := c.getTLSRouteFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get TLSRoute %s: %s", key, err)
			continue
		}

		processTLSRoute(gw, validListeners, tlsRoute, rules)
		processTLSBackends(tlsRoute, services)
	}

	log.Debug().Msgf("Processing %d TCPRoutes", len(c.tcproutes))
	for key := range c.tcproutes {
		tcpRoute, err := c.getTCPRouteFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get TCPRoute %s: %s", key, err)
			continue
		}

		processTCPRoute(gw, validListeners, tcpRoute, rules)
		processTCPBackends(tcpRoute, services)
	}

	return rules, services
}

func processHTTPRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, httpRoute *gwv1beta1.HTTPRoute, acceptedRateLimits map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy, rules map[int32]routecfg.RouteRule, services map[string]serviceInfo) {
	hostnamesRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeHostnames]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeHostnames] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, httpRoute) {
				hostnamesRateLimits = append(hostnamesRateLimits, rateLimit)
			}
		}
	}

	routeRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeRoute]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeRoute] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, httpRoute) {
				routeRateLimits = append(routeRateLimits, rateLimit)
			}
		}
	}

	for _, ref := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, httpRoute.GroupVersionKind(), validListeners)
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

			httpRule := routecfg.L7RouteRule{}
			for _, hostname := range hostnames {
				r := generateHTTPRouteConfig(httpRoute, routeRateLimits, services)

				for _, rateLimit := range hostnamesRateLimits {
					if rl := gwutils.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, rateLimit); rl != nil && r.RateLimit == nil {
						r.RateLimit = newRateLimitConfig(rl)
					}
				}

				httpRule[hostname] = r
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(routecfg.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, httpRule)
				}
			} else {
				rules[port] = httpRule
			}
		}
	}
}

func processGRPCRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, grpcRoute *gwv1alpha2.GRPCRoute, acceptedRateLimits map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy, rules map[int32]routecfg.RouteRule, services map[string]serviceInfo) {
	hostnamesRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeHostnames]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeHostnames] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, grpcRoute) {
				hostnamesRateLimits = append(hostnamesRateLimits, rateLimit)
			}
		}
	}

	routeRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeRoute]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeRoute] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, grpcRoute) {
				routeRateLimits = append(routeRateLimits, rateLimit)
			}
		}
	}

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
				r := generateGRPCRouteCfg(grpcRoute, routeRateLimits, services)

				for _, rateLimit := range hostnamesRateLimits {
					if rl := gwutils.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, rateLimit); rl != nil && r.RateLimit == nil {
						r.RateLimit = newRateLimitConfig(rl)
					}
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

func processTLSRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, tlsRoute *gwv1alpha2.TLSRoute, rules map[int32]routecfg.RouteRule) {
	for _, ref := range tlsRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, tlsRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			if listener.Protocol != gwv1beta1.TLSProtocolType {
				continue
			}

			if listener.TLS == nil {
				continue
			}

			if listener.TLS.Mode == nil {
				continue
			}

			if *listener.TLS.Mode != gwv1beta1.TLSModePassthrough {
				continue
			}

			hostnames := gwutils.GetValidHostnames(listener.Hostname, tlsRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			tlsRule := routecfg.TLSPassthroughRouteRule{}
			for _, hostname := range hostnames {
				if target := generateTLSPassthroughRouteCfg(tlsRoute); target != nil {
					tlsRule[hostname] = *target
				}
			}

			rules[int32(listener.Port)] = tlsRule
		}
	}
}

func processTCPRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, tcpRoute *gwv1alpha2.TCPRoute, rules map[int32]routecfg.RouteRule) {
	for _, ref := range tcpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, tcpRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			switch listener.Protocol {
			case gwv1beta1.TLSProtocolType:
				if listener.TLS == nil {
					continue
				}

				if listener.TLS.Mode == nil {
					continue
				}

				if *listener.TLS.Mode != gwv1beta1.TLSModeTerminate {
					continue
				}

				hostnames := gwutils.GetValidHostnames(listener.Hostname, nil)

				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}

				tlsRule := routecfg.TLSTerminateRouteRule{}
				for _, hostname := range hostnames {
					tlsRule[hostname] = generateTLSTerminateRouteCfg(tcpRoute)
				}

				rules[int32(listener.Port)] = tlsRule
			case gwv1beta1.TCPProtocolType:
				rules[int32(listener.Port)] = generateTCPRouteCfg(tcpRoute)
			}
		}
	}
}

func processTLSBackends(_ *gwv1alpha2.TLSRoute, _ map[string]serviceInfo) {
	// DO nothing for now
}

func processTCPBackends(tcpRoute *gwv1alpha2.TCPRoute, services map[string]serviceInfo) {
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(backend.BackendObjectReference, tcpRoute.Namespace); svcPort != nil {
				services[svcPort.String()] = serviceInfo{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func (c *GatewayCache) sessionStickies() map[string]*gwpav1alpha1.SessionStickyConfig {
	sessionStickies := make(map[string]*gwpav1alpha1.SessionStickyConfig)

	for key := range c.sessionstickies {
		sessionSticky, err := c.getSessionStickyPolicyFromCache(key)

		if err != nil {
			log.Error().Msgf("Failed to get SessionStickyPolicy %s: %s", key, err)
			continue
		}

		if gwutils.IsAcceptedSessionStickyPolicy(sessionSticky) {
			for _, p := range sessionSticky.Spec.Ports {
				if svcPortName := targetRefToServicePortName(sessionSticky.Spec.TargetRef, sessionSticky.Namespace, int32(p.Port)); svcPortName != nil {
					c := p.Config
					if c == nil {
						c = sessionSticky.Spec.DefaultConfig
					}

					if c == nil {
						continue
					}

					sessionStickies[svcPortName.String()] = c
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

		if gwutils.IsAcceptedLoadBalancerPolicy(lb) {
			for _, p := range lb.Spec.Ports {
				if svcPortName := targetRefToServicePortName(lb.Spec.TargetRef, lb.Namespace, int32(p.Port)); svcPortName != nil {
					t := p.Type
					if t == nil {
						t = lb.Spec.DefaultType
					}

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

		if gwutils.IsAcceptedCircuitBreakingPolicy(circuitBreaking) {
			for _, p := range circuitBreaking.Spec.Ports {
				if svcPortName := targetRefToServicePortName(circuitBreaking.Spec.TargetRef, circuitBreaking.Namespace, int32(p.Port)); svcPortName != nil {
					c := p.Config
					if c == nil {
						c = circuitBreaking.Spec.DefaultConfig
					}

					if c == nil {
						continue
					}

					configs[svcPortName.String()] = c
				}
			}
		}
	}

	return configs
}

func (c *GatewayCache) serviceConfigs(services map[string]serviceInfo) map[string]routecfg.ServiceConfig {
	configs := make(map[string]routecfg.ServiceConfig)
	sessionStickies := c.sessionStickies()
	loadBalancers := c.loadBalancers()
	circuitBreakings := c.circuitBreakings()

	for svcPortName, svcInfo := range services {
		svcKey := svcInfo.svcPortName.NamespacedName
		svc, err := c.getServiceFromCache(svcKey)

		if err != nil {
			log.Error().Msgf("Failed to get Service %s: %s", svcKey, err)
			continue
		}

		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.KubernetesEndpointSliceServiceNameLabel: svc.Name,
			},
		})
		if err != nil {
			log.Error().Msgf("Failed to convert LabelSelector to Selector: %s", err)
			continue
		}

		endpointSliceList, err := c.informers.GetListers().EndpointSlice.EndpointSlices(svc.Namespace).List(selector)
		if err != nil {
			log.Error().Msgf("Failed to list EndpointSlice of Service %s: %s", svcKey, err)
			continue
		}

		if len(endpointSliceList) == 0 {
			continue
		}

		svcPort, err := getServicePort(svc, svcInfo.svcPortName.Port)
		if err != nil {
			log.Error().Msgf("Failed to get ServicePort: %s", err)
			continue
		}

		filteredSlices := filterEndpointSliceList(endpointSliceList, svcPort)
		if len(filteredSlices) == 0 {
			log.Error().Msgf("no valid endpoints found for Service %s and port %+v", svcKey, svcPort)
			continue
		}

		endpointSet := make(map[endpointInfo]struct{})
		for _, eps := range filteredSlices {
			for _, endpoint := range eps.Endpoints {
				if !isEndpointReady(endpoint) {
					continue
				}
				endpointPort := findPort(eps.Ports, svcPort)

				for _, address := range endpoint.Addresses {
					ep := endpointInfo{address: address, port: endpointPort}
					endpointSet[ep] = struct{}{}
				}
			}
		}

		svcCfg := routecfg.ServiceConfig{
			//Filters:   svcInfo.filters,
			Endpoints: make(map[string]routecfg.Endpoint),
		}

		for ep := range endpointSet {
			hostport := fmt.Sprintf("%s:%d", ep.address, ep.port)
			svcCfg.Endpoints[hostport] = routecfg.Endpoint{
				Weight: 1,
			}
		}

		if ssCfg, exists := sessionStickies[svcPortName]; exists {
			svcCfg.StickyCookieName = ssCfg.CookieName
			svcCfg.StickyCookieExpires = ssCfg.Expires
		}

		if lbType, exists := loadBalancers[svcPortName]; exists {
			svcCfg.LoadBalancer = lbType
		}

		if cbCfg, exists := circuitBreakings[svcPortName]; exists {
			svcCfg.CircuitBreaking = &routecfg.CircuitBreaking{
				MinRequestAmount:        cbCfg.MinRequestAmount,
				StatTimeWindow:          cbCfg.StatTimeWindow,
				SlowAmountThreshold:     cbCfg.SlowAmountThreshold,
				SlowRatioThreshold:      cbCfg.SlowRatioThreshold,
				SlowTimeThreshold:       cbCfg.SlowTimeThreshold,
				ErrorAmountThreshold:    cbCfg.ErrorAmountThreshold,
				ErrorRatioThreshold:     cbCfg.ErrorRatioThreshold,
				DegradedTimeWindow:      cbCfg.DegradedTimeWindow,
				DegradedStatusCode:      cbCfg.DegradedStatusCode,
				DegradedResponseContent: cbCfg.DegradedResponseContent,
			}
		}

		configs[svcPortName] = svcCfg
	}

	return configs
}

func (c *GatewayCache) chains() routecfg.Chains {
	if c.cfg.GetFeatureFlags().EnableGatewayAgentService {
		return routecfg.Chains{
			HTTPRoute:      insertAgentServiceScript(defaultHTTPChains),
			HTTPSRoute:     insertAgentServiceScript(defaultHTTPSChains),
			TLSPassthrough: defaultTLSPassthroughChains,
			TLSTerminate:   defaultTLSTerminateChains,
			TCPRoute:       defaultTCPChains,
		}
	}

	return routecfg.Chains{
		HTTPRoute:      defaultHTTPChains,
		HTTPSRoute:     defaultHTTPSChains,
		TLSPassthrough: defaultTLSPassthroughChains,
		TLSTerminate:   defaultTLSTerminateChains,
		TCPRoute:       defaultTCPChains,
	}
}

func generateHTTPRouteConfig(httpRoute *gwv1beta1.HTTPRoute, routeRateLimits []gwpav1alpha1.RateLimitPolicy, services map[string]serviceInfo) routecfg.HTTPRouteRuleSpec {
	httpSpec := routecfg.HTTPRouteRuleSpec{
		RouteType: routecfg.L7RouteTypeHTTP,
		Matches:   make([]routecfg.HTTPTrafficMatch, 0),
	}

	for _, rule := range httpRoute.Spec.Rules {
		backends := map[string]routecfg.BackendServiceConfig{}

		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendRef.BackendObjectReference, httpRoute.Namespace); svcPort != nil {
				svcLevelFilters := make([]routecfg.Filter, 0)
				for _, filter := range bk.Filters {
					svcLevelFilters = append(svcLevelFilters, toFSMHTTPRouteFilter(filter, httpRoute.Namespace, services))
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
			ruleLevelFilters = append(ruleLevelFilters, toFSMHTTPRouteFilter(ruleFilter, httpRoute.Namespace, services))
		}

		for _, m := range rule.Matches {
			match := routecfg.HTTPTrafficMatch{
				BackendService: backends,
				Filters:        ruleLevelFilters,
			}

			if m.Path != nil {
				match.Path = &routecfg.Path{
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

			for _, rateLimit := range routeRateLimits {
				if len(rateLimit.Spec.HTTPRateLimits) == 0 {
					continue
				}

				if r := gwutils.GetRateLimitIfHTTPRouteMatchesPolicy(m, rateLimit); r != nil && match.RateLimit == nil {
					match.RateLimit = newRateLimitConfig(r)
				}
			}

			httpSpec.Matches = append(httpSpec.Matches, match)
		}
	}
	return httpSpec
}

func generateGRPCRouteCfg(grpcRoute *gwv1alpha2.GRPCRoute, routeRateLimits []gwpav1alpha1.RateLimitPolicy, services map[string]serviceInfo) routecfg.GRPCRouteRuleSpec {
	grpcSpec := routecfg.GRPCRouteRuleSpec{
		RouteType: routecfg.L7RouteTypeGRPC,
		Matches:   make([]routecfg.GRPCTrafficMatch, 0),
	}

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

			for _, rateLimit := range routeRateLimits {
				if len(rateLimit.Spec.GRPCRateLimits) == 0 {
					continue
				}

				if r := gwutils.GetRateLimitIfGRPCRouteMatchesPolicy(m, rateLimit); r != nil && match.RateLimit == nil {
					match.RateLimit = newRateLimitConfig(r)
				}
			}

			grpcSpec.Matches = append(grpcSpec.Matches, match)
		}
	}

	return grpcSpec
}

func generateTLSTerminateRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) routecfg.TLSBackendService {
	backends := routecfg.TLSBackendService{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendObjectReference, tcpRoute.Namespace); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}

func generateTLSPassthroughRouteCfg(tlsRoute *gwv1alpha2.TLSRoute) *string {
	for _, rule := range tlsRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			// return the first ONE
			return passthroughTarget(bk)
		}
	}

	return nil
}

func generateTCPRouteCfg(tcpRoute *gwv1alpha2.TCPRoute) routecfg.RouteRule {
	backends := routecfg.TCPRouteRule{}

	for _, rule := range tcpRoute.Spec.Rules {
		for _, bk := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(bk.BackendObjectReference, tcpRoute.Namespace); svcPort != nil {
				backends[svcPort.String()] = backendWeight(bk)
			}
		}
	}

	return backends
}
