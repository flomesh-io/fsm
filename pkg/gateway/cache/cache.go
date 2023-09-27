// Package cache implements a cache of all the resources that are relevant to the gateway
package cache

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/routecfg"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

var (
	httpRouteGVK = schema.FromAPIVersionAndKind(gwv1beta1.GroupVersion.String(), "HTTPRoute")
	tlsRouteGVK  = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), "TLSRoute")
	tcpRouteGVK  = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), "TCPRoute")
	grpcRouteGVK = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), "GRPCRoute")
)

// GatewayCache is a cache of all the resources that are relevant to the gateway
type GatewayCache struct {
	repoClient *repo.PipyRepoClient
	informers  *informers.InformerCollection
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator

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

// NewGatewayCache creates a new gateway cache
func NewGatewayCache(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, cfg configurator.Configurator) *GatewayCache {
	repoBaseURL := fmt.Sprintf("%s://%s:%d", "http", cfg.GetRepoServerIPAddr(), cfg.GetProxyServerPort())
	return &GatewayCache{
		repoClient: repo.NewRepoClient(repoBaseURL, cfg.GetFSMLogLevel()),
		informers:  informerCollection,
		kubeClient: kubeClient,
		cfg:        cfg,

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

// Insert inserts an object into the cache
func (c *GatewayCache) Insert(obj interface{}) bool {
	p := c.getProcessor(obj)
	if p != nil {
		return p.Insert(obj, c)
	}

	return false
}

// Delete deletes an object from the cache
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
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIHTTPRoute, key.String()); exists && err == nil {
			r := r.(*gwv1beta1.HTTPRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}

					for _, filter := range backend.Filters {
						if filter.Type == gwv1beta1.HTTPRouteFilterRequestMirror {
							if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
								return true
							}
						}
					}
				}

				for _, filter := range rule.Filters {
					if filter.Type == gwv1beta1.HTTPRouteFilterRequestMirror {
						if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	for key := range c.grpcroutes {
		// Get GRPCRoute from client-go cache
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIGRPCRoute, key.String()); exists && err == nil {
			r := r.(*gwv1alpha2.GRPCRoute)
			for _, rule := range r.Spec.Rules {
				for _, backend := range rule.BackendRefs {
					if isRefToService(backend.BackendObjectReference, service, r.Namespace) {
						return true
					}

					for _, filter := range backend.Filters {
						if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
							if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
								return true
							}
						}
					}
				}

				for _, filter := range rule.Filters {
					if filter.Type == gwv1alpha2.GRPCRouteFilterRequestMirror {
						if isRefToService(filter.RequestMirror.BackendRef, service, r.Namespace) {
							return true
						}
					}
				}
			}
		}
	}

	for key := range c.tlsroutes {
		// Get TLSRoute from client-go cache
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPITLSRoute, key.String()); exists && err == nil {
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
		if r, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPITCPRoute, key.String()); exists && err == nil {
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
		case GroupCore, GroupFlomeshIo:
			log.Debug().Msgf("Ref group is %q", string(*ref.Group))
		default:
			return false
		}
	}

	if ref.Kind != nil {
		switch string(*ref.Kind) {
		case KindService, KindServiceImport:
			log.Debug().Msgf("Ref kind is %q", string(*ref.Kind))
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
		//	log.Error().Msgf("Failed to get Gateway %s: %s", key, err)
		//	continue
		//}
		obj, exists, err := c.informers.GetByKey(informers.InformerKeyGatewayAPIGateway, key.String())
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
			log.Debug().Msgf("Ref group is %q", string(*ref.Group))
		default:
			return false
		}
	}

	if ref.Kind != nil {
		switch string(*ref.Kind) {
		case KindSecret:
			log.Debug().Msgf("Ref kind is %q", string(*ref.Kind))
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

		listenerCfg := c.listeners(gw, validListeners)
		rules, referredServices := c.routeRules(gw, validListeners)
		svcConfigs := c.serviceConfigs(referredServices)

		configSpec := &routecfg.ConfigSpec{
			Defaults:   c.defaults(),
			Listeners:  listenerCfg,
			RouteRules: rules,
			Services:   svcConfigs,
			Chains:     chains(),
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
		DefaultPassthroughUpstreamPort: 443,  // TODO: enrich this from config
		StripAnyHostPort:               true, // TODO: enrich this from config
	}
}

func (c *GatewayCache) isDebugEnabled() bool {
	switch c.cfg.GetFSMGatewayLogLevel() {
	case "debug", "trace":
		return true
	default:
		return false
	}
}

func (c *GatewayCache) listeners(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener) []routecfg.Listener {
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

func (c *GatewayCache) routeRules(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener) (map[int32]routecfg.RouteRule, map[string]serviceInfo) {
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
		processHTTPRoute(gw, validListeners, httpRoute, rules, services)
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
		processGRPCRoute(gw, validListeners, grpcRoute, rules, services)
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

func processHTTPRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, httpRoute *gwv1beta1.HTTPRoute, rules map[int32]routecfg.RouteRule, services map[string]serviceInfo) {
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
				httpRule[hostname] = generateHTTPRouteConfig(httpRoute, services)
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

func processGRPCRoute(gw *gwv1beta1.Gateway, validListeners []gwtypes.Listener, grpcRoute *gwv1alpha2.GRPCRoute, rules map[int32]routecfg.RouteRule, services map[string]serviceInfo) {
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
				grpcRule[hostname] = generateGRPCRouteCfg(grpcRoute, services)
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

//func processHTTPRouteBackendFilters(httpRoute *gwv1beta1.HTTPRoute, services map[string]serviceInfo) {
//	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
//	for _, rule := range httpRoute.Spec.Rules {
//		ruleLevelFilters := make(map[gwv1beta1.HTTPRouteFilterType]routecfg.Filter)
//
//		for _, ruleFilter := range rule.Filters {
//			ruleLevelFilters[ruleFilter.Type] = ruleFilter
//		}
//
//		for _, backend := range rule.BackendRefs {
//			if svcPort := backendRefToServicePortName(backend.BackendRef, httpRoute.Namespace); svcPort != nil {
//				svcFilters := copyMap(ruleLevelFilters)
//				for _, svcFilter := range backend.Filters {
//					svcFilters[svcFilter.Type] = svcFilter
//				}
//
//				svcInfo := serviceInfo{
//					svcPortName: *svcPort,
//					filters:     make([]routecfg.Filter, 0),
//				}
//				for _, f := range svcFilters {
//					svcInfo.filters = append(svcInfo.filters, f)
//				}
//				services[svcPort.String()] = svcInfo
//			}
//		}
//	}
//}

//func processGRPCRouteBackendFilters(grpcRoute *gwv1alpha2.GRPCRoute, services map[string]serviceInfo) {
//	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
//	for _, rule := range grpcRoute.Spec.Rules {
//		ruleLevelFilters := make(map[gwv1alpha2.GRPCRouteFilterType]routecfg.Filter)
//
//		for _, ruleFilter := range rule.Filters {
//			ruleLevelFilters[ruleFilter.Type] = ruleFilter
//		}
//
//		for _, backend := range rule.BackendRefs {
//			if svcPort := backendRefToServicePortName(backend.BackendRef, grpcRoute.Namespace); svcPort != nil {
//				svcFilters := copyMap(ruleLevelFilters)
//				for _, svcFilter := range backend.Filters {
//					svcFilters[svcFilter.Type] = svcFilter
//				}
//
//				svcInfo := serviceInfo{
//					svcPortName: *svcPort,
//					filters:     make([]routecfg.Filter, 0),
//				}
//				for _, f := range svcFilters {
//					svcInfo.filters = append(svcInfo.filters, f)
//				}
//				services[svcPort.String()] = svcInfo
//			}
//		}
//	}
//}

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

func chains() routecfg.Chains {
	return routecfg.Chains{
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
