package cache

import (
	"fmt"

	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
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
	policies := c.policyAttachments()

	for ns, key := range c.gateways {
		gw, err := c.getGatewayFromCache(key)
		if err != nil {
			log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
			continue
		}

		validListeners := gwutils.GetValidListenersFromGateway(gw)
		listenerCfg := c.listeners(gw, validListeners, policies)
		rules, referredServices := c.routeRules(gw, validListeners, policies)
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
		SocketTimeout:                  pointer.Int32(60),
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

func (c *GatewayCache) listeners(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, policies globalPolicyAttachments) []routecfg.Listener {
	listeners := make([]routecfg.Listener, 0)
	enrichers := c.getPortPolicyEnrichers(policies)

	for _, l := range validListeners {
		listener := &routecfg.Listener{
			Protocol: l.Protocol,
			Listen:   c.listenPort(l),
			Port:     l.Port,
		}

		if tls := c.tls(gw, l); tls != nil {
			listener.TLS = tls
		}

		for _, enricher := range enrichers {
			enricher.Enrich(gw, l.Port, listener)
		}

		listeners = append(listeners, *listener)
	}

	return listeners
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
		Certificates: c.certificates(gw, l),
	}
}

func (c *GatewayCache) tlsPassthroughCfg() *routecfg.TLS {
	return &routecfg.TLS{
		TLSModeType: gwv1beta1.TLSModePassthrough,
		// set to false and protect it from being overwritten by the user
		MTLS: pointer.Bool(false),
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

func (c *GatewayCache) routeRules(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, policies globalPolicyAttachments) (map[int32]routecfg.RouteRule, map[string]serviceInfo) {
	rules := make(map[int32]routecfg.RouteRule)
	services := make(map[string]serviceInfo)

	log.Debug().Msgf("Processing %d HTTPRoutes", len(c.httproutes))
	for _, httpRoute := range c.getSortedHTTPRoutes() {
		processHTTPRoute(gw, validListeners, httpRoute, policies, rules, services)
	}

	log.Debug().Msgf("Processing %d GRPCRoutes", len(c.grpcroutes))
	for _, grpcRoute := range c.getSortedGRPCRoutes() {
		processGRPCRoute(gw, validListeners, grpcRoute, policies, rules, services)
	}

	log.Debug().Msgf("Processing %d TLSRoutes", len(c.tlsroutes))
	for _, tlsRoute := range c.getSortedTLSRoutes() {
		processTLSRoute(gw, validListeners, tlsRoute, rules)
		processTLSBackends(tlsRoute, services)
	}

	log.Debug().Msgf("Processing %d TCPRoutes", len(c.tcproutes))
	for _, tcpRoute := range c.getSortedTCPRoutes() {
		processTCPRoute(gw, validListeners, tcpRoute, rules)
		processTCPBackends(tcpRoute, services)
	}

	return rules, services
}

func (c *GatewayCache) serviceConfigs(services map[string]serviceInfo) map[string]routecfg.ServiceConfig {
	configs := make(map[string]routecfg.ServiceConfig)
	enrichers := c.getServicePolicyEnrichers()

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

		svcCfg := &routecfg.ServiceConfig{
			//Filters:   svcInfo.filters,
			Endpoints: make(map[string]routecfg.Endpoint),
		}

		for ep := range endpointSet {
			hostport := fmt.Sprintf("%s:%d", ep.address, ep.port)
			svcCfg.Endpoints[hostport] = routecfg.Endpoint{
				Weight: 1,
			}
		}

		for _, enricher := range enrichers {
			enricher.Enrich(svcPortName, svcCfg)
		}

		configs[svcPortName] = *svcCfg
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
