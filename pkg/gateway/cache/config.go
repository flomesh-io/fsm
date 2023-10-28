package cache

import (
	"fmt"
	"sort"

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
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// BuildConfigs builds the configs for all the gateways in the cache
func (c *GatewayCache) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	configs := make(map[string]*routecfg.ConfigSpec)

	for ns, key := range c.gateways {
		obj, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIGateway, key.String())
		if !exists {
			log.Error().Msgf("Gateway %s doesn't exist", key)
			continue
		}

		if err != nil {
			log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
			continue
		}

		gw := obj.(*gwv1beta1.Gateway)
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
				// RateLimitPolicy is attached to Gateway
				if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gwutils.ObjectKey(gw)) {
					// A matched rate limit policy and no rate limit is set on the listener,
					//  as the rate limits are sorted by timestamp, the first one wins
					if len(rateLimit.Spec.Ports) > 0 &&
						gwutils.PortMatchesRateLimitPolicy(l.Port, rateLimit) &&
						listener.BpsLimit == nil {
						listener.BpsLimit = bpsRateLimit(l.Port, rateLimit)
					}
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
		policy, exists, err := c.informers.GetByKey(informers.InformerKeyRateLimitPolicy, key.String())
		if !exists {
			log.Error().Msgf("RateLimitPolicy %s does not exist", key)
			continue
		}

		if err != nil {
			log.Error().Msgf("Failed to get RateLimitPolicy %s: %s", key, err)
			continue
		}

		rateLimitPolicy := policy.(*gwpav1alpha1.RateLimitPolicy)
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
		if string(*ref.Kind) == KindSecret && string(*ref.Group) == GroupCore {
			ns := getSecretRefNamespace(gw, ref)
			name := string(ref.Name)
			secret, err := c.informers.GetListers().Secret.Secrets(ns).Get(name)

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
		route, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIHTTPRoute, key.String())
		if !exists {
			log.Error().Msgf("HTTPRoute %s does not exist", key)
			continue
		}

		if err != nil {
			log.Error().Msgf("Failed to get HTTPRoute %s: %s", key, err)
			continue
		}

		httpRoute := route.(*gwv1beta1.HTTPRoute)
		log.Debug().Msgf("Processing HTTPRoute %v", httpRoute)
		processHTTPRoute(gw, validListeners, httpRoute, acceptedRateLimits, rules, services)
		//processHTTPRouteBackendFilters(httpRoute, services)
	}

	log.Debug().Msgf("Processing %d GRPCRoutes", len(c.grpcroutes))
	for key := range c.grpcroutes {
		route, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIGRPCRoute, key.String())
		if !exists {
			log.Error().Msgf("GRPCRoute %s does not exist", key)
			continue
		}

		if err != nil {
			log.Error().Msgf("Failed to get GRPCRoute %s: %s", key, err)
			continue
		}

		grpcRoute := route.(*gwv1alpha2.GRPCRoute)
		processGRPCRoute(gw, validListeners, grpcRoute, acceptedRateLimits, rules, services)
		//processGRPCRouteBackendFilters(grpcRoute, services)
	}

	log.Debug().Msgf("Processing %d TLSRoutes", len(c.tlsroutes))
	for key := range c.tlsroutes {
		route, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPITLSRoute, key.String())
		if !exists {
			log.Error().Msgf("TLSRoute %s does not exist", key)
			continue
		}

		if err != nil {
			log.Error().Msgf("Failed to get TLSRoute %s: %s", key, err)
			continue
		}

		tlsRoute := route.(*gwv1alpha2.TLSRoute)
		processTLSRoute(gw, validListeners, tlsRoute, rules)
		processTLSBackends(tlsRoute, services)
	}

	log.Debug().Msgf("Processing %d TCPRoutes", len(c.tcproutes))
	for key := range c.tcproutes {
		route, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPITCPRoute, key.String())
		if !exists {
			log.Error().Msgf("TCPRoute %s does not exist", key)
			continue
		}

		if err != nil {
			log.Error().Msgf("Failed to get TCPRoute %s: %s", key, err)
			continue
		}

		tcpRoute := route.(*gwv1alpha2.TCPRoute)
		processTCPRoute(gw, validListeners, tcpRoute, rules)
		processTCPBackends(tcpRoute, services)
	}

	return rules, services
}

func processHTTPRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, httpRoute *gwv1beta1.HTTPRoute, acceptedRateLimits map[RateLimitPolicyMatchType][]gwpav1alpha1.RateLimitPolicy, rules map[int32]routecfg.RouteRule, services map[string]serviceInfo) {
	hostnamesRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeHostnames]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeHostnames] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gwutils.ObjectKey(httpRoute)) {
				hostnamesRateLimits = append(hostnamesRateLimits, rateLimit)
			}
		}
	}

	routeRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeRoute]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeRoute] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gwutils.ObjectKey(httpRoute)) {
				routeRateLimits = append(routeRateLimits, rateLimit)
			}
		}
	}

	for _, ref := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, httpRouteGVK, validListeners)
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
					if gwutils.RouteHostnameMatchesRateLimitPolicy(hostname, rateLimit) && r.RateLimit == nil {
						r.RateLimit = newHostnameRateLimit(hostname, rateLimit)
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
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gwutils.ObjectKey(grpcRoute)) {
				hostnamesRateLimits = append(hostnamesRateLimits, rateLimit)
			}
		}
	}

	routeRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	if len(acceptedRateLimits[RateLimitPolicyMatchTypeRoute]) > 0 {
		for _, rateLimit := range acceptedRateLimits[RateLimitPolicyMatchTypeRoute] {
			if gwutils.IsRefToTarget(rateLimit.Spec.TargetRef, gwutils.ObjectKey(grpcRoute)) {
				routeRateLimits = append(routeRateLimits, rateLimit)
			}
		}
	}

	for _, ref := range grpcRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, grpcRouteGVK, validListeners)
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
					if gwutils.RouteHostnameMatchesRateLimitPolicy(hostname, rateLimit) && r.RateLimit == nil {
						r.RateLimit = newHostnameRateLimit(hostname, rateLimit)
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

		allowedListeners := allowedListeners(ref, tlsRouteGVK, validListeners)
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

		allowedListeners := allowedListeners(ref, tcpRouteGVK, validListeners)
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

func (c *GatewayCache) serviceConfigs(services map[string]serviceInfo) map[string]routecfg.ServiceConfig {
	configs := make(map[string]routecfg.ServiceConfig)

	for svcPortName, svcInfo := range services {
		svcKey := svcInfo.svcPortName.NamespacedName
		obj, exists, err := c.informers.GetByKey(informers.InformerKeyService, svcKey.String())
		if err != nil {
			log.Error().Msgf("Failed to get Service %s: %s", svcKey, err)
			continue
		}
		if !exists {
			log.Error().Msgf("Service %s doesn't exist", svcKey)
			continue
		}

		svc := obj.(*corev1.Service)

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

				if gwutils.HTTPRouteMatchesRateLimitPolicy(m, rateLimit) && match.RateLimit == nil {
					match.RateLimit = newHTTPRouteRateLimit(m, rateLimit)
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

				if gwutils.GRPCRouteMatchesRateLimitPolicy(m, rateLimit) && match.RateLimit == nil {
					match.RateLimit = newGRPCRouteRateLimit(m, rateLimit)
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
