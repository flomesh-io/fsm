package cache

import (
	"fmt"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/route"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/utils"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sync"
)

type GatewayCache struct {
	repoClient *repo.PipyRepoClient
	informers  *informers.InformerCollection
	kubeClient kubernetes.Interface

	processors map[ProcessorType]Processor

	gatewayclass   *gwv1beta1.GatewayClass
	gateways       map[string]client.ObjectKey // ns -> gateway
	services       map[client.ObjectKey]struct{}
	serviceimports map[client.ObjectKey]struct{}
	endpoints      map[client.ObjectKey]struct{}
	endpointslices map[client.ObjectKey]map[client.ObjectKey]struct{} // svc -> endpointslices
	secrets        map[client.ObjectKey]struct{}
	httproutes     map[client.ObjectKey]struct{}
	grpcroutes     map[client.ObjectKey]struct{}
	tcproutes      map[client.ObjectKey]struct{}
	tlsroutes      map[client.ObjectKey]struct{}

	mutex *sync.RWMutex
}

func NewGatewayCache(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, cfg configurator.Configurator) *GatewayCache {
	return &GatewayCache{
		repoClient: repo.NewRepoClient(cfg.GetRepoServerIPAddr(), uint16(cfg.GetProxyServerPort())),
		informers:  informerCollection,
		kubeClient: kubeClient,

		processors: map[ProcessorType]Processor{
			ServicesProcessorType:       &ServicesProcessor{},
			ServiceImportsProcessorType: &ServiceImportsProcessor{},
			EndpointSlicesProcessorType: &EndpointSlicesProcessor{},
			//EndpointsProcessorType:      &EndpointsProcessor{},
			SecretsProcessorType:        &SecretProcessor{},
			GatewayClassesProcessorType: &GatewayClassesProcessor{},
			GatewaysProcessorType:       &GatewaysProcessor{},
			HTTPRoutesProcessorType:     &HTTPRoutesProcessor{},
			GRPCRoutesProcessorType:     &GRPCRoutesProcessor{},
			TCPRoutesProcessorType:      &TCPRoutesProcessor{},
			TLSRoutesProcessorType:      &TLSRoutesProcessor{},
		},

		gateways:       make(map[string]client.ObjectKey),
		services:       make(map[client.ObjectKey]struct{}),
		serviceimports: make(map[client.ObjectKey]struct{}),
		endpointslices: make(map[client.ObjectKey]map[client.ObjectKey]struct{}),
		//endpoints:      make(map[client.ObjectKey]struct{}),
		secrets:    make(map[client.ObjectKey]struct{}),
		httproutes: make(map[client.ObjectKey]struct{}),
		grpcroutes: make(map[client.ObjectKey]struct{}),
		tcproutes:  make(map[client.ObjectKey]struct{}),
		tlsroutes:  make(map[client.ObjectKey]struct{}),

		mutex: new(sync.RWMutex),
	}
}

func (c *GatewayCache) Insert(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Insert(obj, c)
	}

	return false
}

func (c *GatewayCache) Delete(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Delete(obj, c)
	}

	return false
}

//func (c *GatewayCache) WaitForCacheSync(ctx context.Context) bool {
//	return c.cache.WaitForCacheSync(ctx)
//}

func (c *GatewayCache) getProcessor(obj interface{}) Processor {
	switch obj.(type) {
	case *corev1.Service:
		return c.processors[ServicesProcessorType]
	case *mcsv1alpha1.ServiceImport:
		return c.processors[ServiceImportsProcessorType]
	//case *corev1.Endpoints:
	//	return c.processors[EndpointsProcessorType]
	case *discoveryv1.EndpointSlice:
		return c.processors[EndpointSlicesProcessorType]
	case *corev1.Secret:
		return c.processors[SecretsProcessorType]
	case *gwv1beta1.GatewayClass:
		return c.processors[GatewayClassesProcessorType]
	case *gwv1beta1.Gateway:
		return c.processors[GatewaysProcessorType]
	case *gwv1beta1.HTTPRoute:
		return c.processors[HTTPRoutesProcessorType]
	case *gwv1alpha2.GRPCRoute:
		return c.processors[GRPCRoutesProcessorType]
	case *gwv1alpha2.TCPRoute:
		return c.processors[TCPRoutesProcessorType]
	case *gwv1alpha2.TLSRoute:
		return c.processors[TLSRoutesProcessorType]
	}

	return nil
}

func (c *GatewayCache) isRoutableService(service client.ObjectKey) bool {
	for key := range c.httproutes {
		// Get HTTPRoute from client-go cache
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiHTTPRoute, key.String()); exists && err == nil {
			r := r.(*gwv1beta1.HTTPRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	for key := range c.grpcroutes {
		// Get GRPCRoute from client-go cache
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiGRPCRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.GRPCRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	for key := range c.tlsroutes {
		// Get TLSRoute from client-go cache
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiTLSRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.TLSRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	for key := range c.tcproutes {
		// Get TCPRoute from client-go cache
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiTCPRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.TCPRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}
				}
			}
		}
	}

	return false
}

func isRefToService(ref gwv1beta1.BackendObjectReference, service client.ObjectKey, ns string) bool {
	if ref.Group != nil {
		switch string(*ref.Group) {
		case "", "flomesh.io":
			log.Info().Msgf("Ref group is %q", string(*ref.Group))
		default:
			return false
		}
	}

	if ref.Kind != nil {
		switch string(*ref.Kind) {
		case "Service", "ServiceImport":
			log.Info().Msgf("Ref kind is %q", string(*ref.Kind))
		default:
			return false
		}
	}

	if ref.Namespace == nil {
		if ns != service.Namespace {
			return false
		}
	} else {
		if string(*ref.Namespace) != service.Namespace {
			return false
		}
	}

	return string(ref.Name) == service.Name
}

func (c *GatewayCache) isEffectiveRoute(parentRefs []gwv1beta1.ParentReference) bool {
	if len(c.gateways) == 0 {
		return false
	}

	for _, parentRef := range parentRefs {
		for _, gw := range c.gateways {
			if gwutils.IsRefToGateway(parentRef, gw) {
				return true
			}
		}
	}

	return false
}

func (c *GatewayCache) isSecretReferredByAnyGateway(secret client.ObjectKey) bool {
	//ctx := context.TODO()
	for _, key := range c.gateways {
		//gw := &gwv1beta1.Gateway{}
		//if err := c.cache.Get(ctx, key, gw); err != nil {
		//	klog.Errorf("Failed to get Gateway %s: %s", key, err)
		//	continue
		//}
		obj, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiGateway, key.String())
		if err != nil {
			log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
			continue
		}
		if !exists {
			log.Error().Msgf("Gateway %s doesn't exist", key)
			continue
		}

		gw := obj.(*gwv1beta1.Gateway)

		for _, l := range gw.Spec.Listeners {
			switch l.Protocol {
			case gwv1beta1.HTTPSProtocolType, gwv1beta1.TLSProtocolType:
				if l.TLS == nil {
					continue
				}

				if l.TLS.Mode == nil || *l.TLS.Mode == gwv1beta1.TLSModeTerminate {
					if len(l.TLS.CertificateRefs) == 0 {
						continue
					}

					for _, ref := range l.TLS.CertificateRefs {
						if isRefToSecret(ref, secret, gw.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func isRefToSecret(ref gwv1beta1.SecretObjectReference, secret client.ObjectKey, ns string) bool {
	if ref.Group != nil {
		switch string(*ref.Group) {
		case "":
			log.Info().Msgf("Ref group is %q", string(*ref.Group))
		default:
			return false
		}
	}

	if ref.Kind != nil {
		switch string(*ref.Kind) {
		case "Secret":
			log.Info().Msgf("Ref kind is %q", string(*ref.Kind))
		default:
			return false
		}
	}

	if ref.Namespace == nil {
		if ns != secret.Namespace {
			return false
		}
	} else {
		if string(*ref.Namespace) != secret.Namespace {
			return false
		}
	}

	return string(ref.Name) == secret.Name
}

func (c *GatewayCache) BuildConfigs() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	configs := make(map[string]*route.ConfigSpec)

	for ns, key := range c.gateways {
		if gw, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiGateway, key.String()); exists && err == nil {
			gw := gw.(*gwv1beta1.Gateway)

			validListeners := gwutils.GetValidListenersFromGateway(gw)
			listenerCfg := c.listeners(gw, validListeners)
			rules, referredServices := c.routeRules(gw, validListeners)
			svcConfigs := c.serviceConfigs(referredServices)

			configSpec := &route.ConfigSpec{
				Defaults:   c.defaults(),
				Listeners:  listenerCfg,
				RouteRules: rules,
				Services:   svcConfigs,
				Chains:     chains(),
			}
			configSpec.Version = utils.SimpleHash(configSpec)
			configs[ns] = configSpec
		}
	}

	for ns, cfg := range configs {
		gatewayPath := utils.GatewayCodebasePath(ns)
		if exists := c.repoClient.CodebaseExists(gatewayPath); !exists {
			continue
		}

		jsonVersion, err := c.getVersionOfConfigJson(gatewayPath)
		if err != nil {
			continue
		}

		if jsonVersion == cfg.Version {
			// config not changed, ignore updating
			log.Info().Msgf("%s/config.json doesn't change, ignore updating...", gatewayPath)
			continue
		}

		go func(cfg *route.ConfigSpec) {
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

			hash := utils.Hash([]byte(cfg.Version))
			if _, err := c.repoClient.Batch(fmt.Sprintf("%d", hash), batches); err != nil {
				log.Error().Msgf("Sync gateway config to repo failed: %s", err)
				return
			}
		}(cfg)
	}
}

func (c *GatewayCache) getVersionOfConfigJson(basepath string) (string, error) {
	path := fmt.Sprintf("%s/config.json", basepath)

	json, err := c.repoClient.GetFile(path)
	if err != nil {
		log.Error().Msgf("Get %q from pipy repo error: %s", path, err)
		return "", err
	}

	version := gjson.Get(json, "Version").String()

	return version, nil
}

func (c *GatewayCache) defaults() route.Defaults {
	return route.Defaults{
		EnableDebug:                    true,
		DefaultPassthroughUpstreamPort: 443,
	}
}

func (c *GatewayCache) listeners(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener) []route.Listener {
	listeners := make([]route.Listener, 0)
	for _, l := range validListeners {
		listener := route.Listener{
			Protocol: l.Protocol,
			Listen:   c.listenPort(l),
			Port:     l.Port,
		}

		if tls := c.tls(gw, l); tls != nil {
			listener.TLS = tls
		}

		listeners = append(listeners, listener)
	}

	return listeners
}

func (c *GatewayCache) listenPort(l gwtypes.Listener) gwv1beta1.PortNumber {
	if l.Port < 1024 {
		return l.Port + 60000
	}

	return l.Port
}

func (c *GatewayCache) tls(gw *gwv1beta1.Gateway, l gwtypes.Listener) *route.TLS {
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

func (c *GatewayCache) tlsTerminateCfg(gw *gwv1beta1.Gateway, l gwtypes.Listener) *route.TLS {
	return &route.TLS{
		TLSModeType:  gwv1beta1.TLSModeTerminate,
		MTLS:         isMTLSEnabled(gw),
		Certificates: c.certificates(gw, l),
	}
}

func (c *GatewayCache) tlsPassthroughCfg() *route.TLS {
	return &route.TLS{
		TLSModeType: gwv1beta1.TLSModePassthrough,
		MTLS:        false,
	}
}

func (c *GatewayCache) certificates(gw *gwv1beta1.Gateway, l gwtypes.Listener) []route.Certificate {
	certs := make([]route.Certificate, 0)
	for _, ref := range l.TLS.CertificateRefs {
		if string(*ref.Kind) == "Secret" && string(*ref.Group) == "" {
			ns := getSecretRefNamespace(gw, ref)
			name := string(ref.Name)
			secret, err := c.informers.GetListers().Secret.Secrets(ns).Get(name)

			if err != nil {
				log.Error().Msgf("Failed to get Secret %s/%s: %s", ns, name, err)
				continue
			}

			cert := route.Certificate{
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

func (c *GatewayCache) routeRules(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener) (map[int32]route.RouteRule, map[string]serviceInfo) {
	rules := make(map[int32]route.RouteRule)
	services := make(map[string]serviceInfo)

	for key := range c.httproutes {
		if httpRoute, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiHTTPRoute, key.String()); exists && err == nil {
			httpRoute := httpRoute.(*gwv1beta1.HTTPRoute)

			processHttpRoute(gw, validListeners, httpRoute, rules)
			processHttpRouteBackendFilters(httpRoute, services)
		}
	}

	for key := range c.grpcroutes {
		if grpcRoute, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiGRPCRoute, key.String()); exists && err == nil {
			grpcRoute := grpcRoute.(*gwv1alpha2.GRPCRoute)

			processGrpcRoute(gw, validListeners, grpcRoute, rules)
			processGrpcRouteBackendFilters(grpcRoute, services)
		}
	}

	for key := range c.tlsroutes {
		if tlsRoute, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiTLSRoute, key.String()); exists && err == nil {
			tlsRoute := tlsRoute.(*gwv1alpha2.TLSRoute)

			processTlsRoute(gw, validListeners, tlsRoute, rules)
			processTlsBackends(tlsRoute, services)
		}
	}

	for key := range c.tcproutes {
		if tcpRoute, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayApiTCPRoute, key.String()); exists && err == nil {
			tcpRoute := tcpRoute.(*gwv1alpha2.TCPRoute)

			processTcpRoute(gw, validListeners, tcpRoute, rules)
			processTcpBackends(tcpRoute, services)
		}
	}

	return rules, services
}

func processHttpRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, httpRoute *gwv1beta1.HTTPRoute, rules map[int32]route.RouteRule) {
	for _, ref := range httpRoute.Spec.ParentRefs {
		if !gwutils.IsRefToGateway(ref, gwutils.ObjectKey(gw)) {
			continue
		}

		allowedListeners := allowedListeners(ref, httpRoute.GroupVersionKind(), validListeners)
		if len(allowedListeners) == 0 {
			continue
		}

		for _, listener := range allowedListeners {
			hostnames := gwutils.GetValidHostnames(listener.Hostname, httpRoute.Spec.Hostnames)

			if len(hostnames) == 0 {
				// no valid hostnames, should ignore it
				continue
			}

			httpRule := route.L7RouteRule{}
			for _, hostname := range hostnames {
				httpRule[hostname] = generateHttpRouteConfig(httpRoute)
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(route.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, httpRule)
				}
			} else {
				rules[port] = httpRule
			}
		}
	}
}

func processGrpcRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, grpcRoute *gwv1alpha2.GRPCRoute, rules map[int32]route.RouteRule) {
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

			grpcRule := route.L7RouteRule{}
			for _, hostname := range hostnames {
				grpcRule[hostname] = generateGrpcRouteCfg(grpcRoute)
			}

			port := int32(listener.Port)
			if rule, exists := rules[port]; exists {
				if l7Rule, ok := rule.(route.L7RouteRule); ok {
					rules[port] = mergeL7RouteRule(l7Rule, grpcRule)
				}
			} else {
				rules[port] = grpcRule
			}
		}
	}
}

func processTlsRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, tlsRoute *gwv1alpha2.TLSRoute, rules map[int32]route.RouteRule) {
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

			tlsRule := route.TLSPassthroughRouteRule{}
			for _, hostname := range hostnames {
				if target := generateTLSPassthroughRouteCfg(tlsRoute); target != nil {
					tlsRule[hostname] = *target
				}
			}

			rules[int32(listener.Port)] = tlsRule
		}
	}
}

func processTcpRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, tcpRoute *gwv1alpha2.TCPRoute, rules map[int32]route.RouteRule) {
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

				tlsRule := route.TLSTerminateRouteRule{}
				for _, hostname := range hostnames {
					tlsRule[hostname] = generateTLSTerminateRouteCfg(tcpRoute)
				}

				rules[int32(listener.Port)] = tlsRule
			case gwv1beta1.TCPProtocolType:
				rules[int32(listener.Port)] = generateTcpRouteCfg(tcpRoute)
			}
		}
	}
}

func processHttpRouteBackendFilters(httpRoute *gwv1beta1.HTTPRoute, services map[string]serviceInfo) {
	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
	for _, rule := range httpRoute.Spec.Rules {
		ruleLevelFilters := make(map[gwv1beta1.HTTPRouteFilterType]route.Filter)

		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters[ruleFilter.Type] = ruleFilter
		}

		for _, backend := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(backend.BackendRef, httpRoute.Namespace); svcPort != nil {
				svcFilters := copyMap(ruleLevelFilters)
				for _, svcFilter := range backend.Filters {
					svcFilters[svcFilter.Type] = svcFilter
				}

				svcInfo := serviceInfo{
					svcPortName: *svcPort,
					filters:     make([]route.Filter, 0),
				}
				for _, f := range svcFilters {
					svcInfo.filters = append(svcInfo.filters, f)
				}
				services[svcPort.String()] = svcInfo
			}
		}
	}
}

func processGrpcRouteBackendFilters(grpcRoute *gwv1alpha2.GRPCRoute, services map[string]serviceInfo) {
	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
	for _, rule := range grpcRoute.Spec.Rules {
		ruleLevelFilters := make(map[gwv1alpha2.GRPCRouteFilterType]route.Filter)

		for _, ruleFilter := range rule.Filters {
			ruleLevelFilters[ruleFilter.Type] = ruleFilter
		}

		for _, backend := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(backend.BackendRef, grpcRoute.Namespace); svcPort != nil {
				svcFilters := copyMap(ruleLevelFilters)
				for _, svcFilter := range backend.Filters {
					svcFilters[svcFilter.Type] = svcFilter
				}

				svcInfo := serviceInfo{
					svcPortName: *svcPort,
					filters:     make([]route.Filter, 0),
				}
				for _, f := range svcFilters {
					svcInfo.filters = append(svcInfo.filters, f)
				}
				services[svcPort.String()] = svcInfo
			}
		}
	}
}

func processTlsBackends(tlsRoute *gwv1alpha2.TLSRoute, services map[string]serviceInfo) {
	// DO nothing for now
}

func processTcpBackends(tcpRoute *gwv1alpha2.TCPRoute, services map[string]serviceInfo) {
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if svcPort := backendRefToServicePortName(backend, tcpRoute.Namespace); svcPort != nil {
				services[svcPort.String()] = serviceInfo{
					svcPortName: *svcPort,
				}
			}
		}
	}
}

func (c *GatewayCache) serviceConfigs(services map[string]serviceInfo) map[string]route.ServiceConfig {
	configs := make(map[string]route.ServiceConfig)

	for svcPortName, svcInfo := range services {
		svcKey := svcInfo.svcPortName.NamespacedName
		if svc, exists, err := c.informers.GetByKey(informers.InformerKeyService, svcKey.String()); exists && err == nil {
			svc := svc.(*corev1.Service)

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

			svcCfg := route.ServiceConfig{
				Filters:   svcInfo.filters,
				Endpoints: make(map[string]route.Endpoint),
			}

			for ep := range endpointSet {
				hostport := fmt.Sprintf("%s:%d", ep.address, ep.port)
				svcCfg.Endpoints[hostport] = route.Endpoint{
					Weight: 1,
				}
			}

			configs[svcPortName] = svcCfg
		}
	}

	return configs
}

func chains() route.Chains {
	return route.Chains{
		HTTPRoute: []string{
			"common/access-control.js",
			"common/ratelimit.js",
			"common/consumer.js",
			"http/codec.js",
			"http/auth.js",
			"http/route.js",
			"http/service.js",
			"http/metrics.js",
			"http/tracing.js",
			"http/logging.js",
			"http/circuit-breaker.js",
			"http/throttle-domain.js",
			"http/throttle-route.js",
			"filter/request-redirect.js",
			"filter/header-modifier.js",
			"filter/url-rewrite.js",
			"http/forward.js",
			"http/default.js",
		},
		HTTPSRoute: []string{
			"common/access-control.js",
			"common/ratelimit.js",
			"common/tls-termination.js",
			"common/consumer.js",
			"http/codec.js",
			"http/auth.js",
			"http/route.js",
			"http/service.js",
			"http/metrics.js",
			"http/tracing.js",
			"http/logging.js",
			"http/circuit-breaker.js",
			"http/throttle-domain.js",
			"http/throttle-route.js",
			"filter/request-redirect.js",
			"filter/header-modifier.js",
			"filter/url-rewrite.js",
			"http/forward.js",
			"http/default.js",
		},
		TLSPassthrough: []string{
			"common/access-control.js",
			"common/ratelimit.js",
			"tls/passthrough.js",
			"common/consumer.js",
		},
		TLSTerminate: []string{
			"common/access-control.js",
			"common/ratelimit.js",
			"common/tls-termination.js",
			"common/consumer.js",
			"tls/forward.js",
		},
		TCPRoute: []string{
			"common/access-control.js",
			"common/ratelimit.js",
			"tcp/forward.js",
		},
	}
}
