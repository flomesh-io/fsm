package cache

import (
	"fmt"
	"strconv"
	"strings"

	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/types"

	"github.com/flomesh-io/fsm/pkg/connector"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"github.com/flomesh-io/fsm/pkg/configurator"

	v1 "k8s.io/client-go/listers/core/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/gateway/fgw"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/utils"
)

type GatewayProcessor struct {
	cache           *GatewayCache
	gateway         *gwv1.Gateway
	policies        globalPolicyAttachments
	referenceGrants []client.Object
	validListeners  []gwtypes.Listener
	services        map[string]serviceContext
	rules           map[int32]fgw.RouteRule
	upstreams       calculateEndpointsFunc
}

func NewGatewayProcessor(cache *GatewayCache, gateway *gwv1.Gateway, policies globalPolicyAttachments, referenceGrants []client.Object) *GatewayProcessor {
	p := &GatewayProcessor{
		cache:           cache,
		gateway:         gateway,
		policies:        policies,
		referenceGrants: referenceGrants,
		validListeners:  gwutils.GetValidListenersForGateway(gateway),
		services:        make(map[string]serviceContext),
		rules:           make(map[int32]fgw.RouteRule),
	}

	if cache.useEndpointSlices {
		p.upstreams = p.upstreamsByEndpointSlices
	} else {
		p.upstreams = p.upstreamsByEndpoints
	}

	return p
}

func (c *GatewayProcessor) build() *fgw.ConfigSpec {
	// those three methods must run in order, as they depend on previous results
	listeners := c.listeners()
	rules := c.routeRules()
	services := c.serviceConfigs()

	configSpec := &fgw.ConfigSpec{
		Defaults:   c.defaults(),
		Listeners:  listeners,
		RouteRules: rules,
		Services:   services,
		Chains:     c.chains(),
	}
	configSpec.Version = utils.SimpleHash(configSpec)

	return configSpec
}

func (c *GatewayProcessor) getResourcesFromCache(resourceType informers.ResourceType, shouldSort bool) []client.Object {
	return c.cache.getResourcesFromCache(resourceType, shouldSort)
}

func (c *GatewayProcessor) getNamespaceLister() v1.NamespaceLister {
	return c.cache.informers.GetListers().Namespace
}

func (c *GatewayProcessor) getConfig() configurator.Configurator {
	return c.cache.cfg
}

func (c *GatewayProcessor) isDebugEnabled() bool {
	switch c.getConfig().GetFGWLogLevel() {
	case "debug", "trace":
		return true
	default:
		return false
	}
}

func (c *GatewayProcessor) listeners() []fgw.Listener {
	listeners := make([]fgw.Listener, 0)
	enrichers := c.getPortPolicyEnrichers()

	for _, l := range c.validListeners {
		listener := &fgw.Listener{
			Protocol: l.Protocol,
			Listen:   c.listenPort(l),
			Port:     l.Port,
		}

		if tls := c.tls(l); tls != nil {
			listener.TLS = tls
		}

		for _, enricher := range enrichers {
			enricher.Enrich(c.gateway, l.Port, listener)
		}

		listeners = append(listeners, *listener)
	}

	return listeners
}

func (c *GatewayProcessor) listenPort(l gwtypes.Listener) gwv1.PortNumber {
	if l.Port < 1024 {
		return l.Port + 60000
	}

	return l.Port
}

func (c *GatewayProcessor) tls(l gwtypes.Listener) *fgw.TLS {
	switch l.Protocol {
	case gwv1.HTTPSProtocolType:
		// Terminate
		if l.TLS != nil {
			if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
				return c.tlsTerminateCfg(l)
			}
		}
	case gwv1.TLSProtocolType:
		// Terminate & Passthrough
		if l.TLS != nil {
			if l.TLS.Mode == nil {
				return c.tlsTerminateCfg(l)
			}

			switch *l.TLS.Mode {
			case gwv1.TLSModeTerminate:
				return c.tlsTerminateCfg(l)
			case gwv1.TLSModePassthrough:
				return c.tlsPassthroughCfg()
			}
		}
	}

	return nil
}

func (c *GatewayProcessor) tlsTerminateCfg(l gwtypes.Listener) *fgw.TLS {
	cfg := &fgw.TLS{
		TLSModeType:  gwv1.TLSModeTerminate,
		Certificates: c.certificates(l),
		CACerts:      c.caCerts(l),
	}

	// keep it nil if not mTLS to reduce the size of the generated config
	if isMTLS(l) {
		cfg.MTLS = ptr.To(true)
	}

	return cfg
}

func (c *GatewayProcessor) tlsPassthroughCfg() *fgw.TLS {
	return &fgw.TLS{
		TLSModeType: gwv1.TLSModePassthrough,
		// set to false and protect it from being overwritten by the user
		MTLS: pointer.Bool(false),
	}
}

func (c *GatewayProcessor) certificates(l gwtypes.Listener) []fgw.Certificate {
	certs := make([]fgw.Certificate, 0)

	for _, ref := range l.TLS.CertificateRefs {
		secret, err := c.secretRefToSecret(c.gateway, ref)

		if err != nil {
			log.Error().Msgf("Failed to resolve Secret: %s", err)
			continue
		}

		if secret.Type != corev1.SecretTypeTLS {
			log.Warn().Msgf("Secret %s/%s is not of type %s, will be ignored for port %d of Gateway %s/%s",
				secret.Namespace, secret.Name, corev1.SecretTypeTLS,
				l.Port, c.gateway.Namespace, c.gateway.Name)
			continue
		}

		cert := fgw.Certificate{
			CertChain:  string(secret.Data[corev1.TLSCertKey]),
			PrivateKey: string(secret.Data[corev1.TLSPrivateKeyKey]),
		}

		certs = append(certs, cert)
	}

	return certs
}

func (c *GatewayProcessor) caCerts(l gwtypes.Listener) []string {
	certs := make([]string, 0)

	if l.TLS.FrontendValidation != nil && len(l.TLS.FrontendValidation.CACertificateRefs) > 0 {
		for _, ref := range l.TLS.FrontendValidation.CACertificateRefs {
			ca, err := c.objectRefToCACertificate(c.gateway, ref)

			if err != nil {
				log.Error().Msgf("Failed to resolve CA Certificate: %s", err)
				continue
			}

			certs = append(certs, string(ca))
		}
	}

	return certs
}

func (c *GatewayProcessor) routeRules() map[int32]fgw.RouteRule {
	for _, httpRoute := range c.getResourcesFromCache(informers.HTTPRoutesResourceType, true) {
		httpRoute := httpRoute.(*gwv1.HTTPRoute)
		c.processHTTPRoute(httpRoute)
	}

	for _, grpcRoute := range c.getResourcesFromCache(informers.GRPCRoutesResourceType, true) {
		grpcRoute := grpcRoute.(*gwv1.GRPCRoute)
		c.processGRPCRoute(grpcRoute)
	}

	for _, tlsRoute := range c.getResourcesFromCache(informers.TLSRoutesResourceType, true) {
		tlsRoute := tlsRoute.(*gwv1alpha2.TLSRoute)
		c.processTLSRoute(tlsRoute)
	}

	for _, tcpRoute := range c.getResourcesFromCache(informers.TCPRoutesResourceType, true) {
		tcpRoute := tcpRoute.(*gwv1alpha2.TCPRoute)
		c.processTCPRoute(tcpRoute)
	}

	for _, udpRoute := range c.getResourcesFromCache(informers.UDPRoutesResourceType, true) {
		udpRoute := udpRoute.(*gwv1alpha2.UDPRoute)
		c.processUDPRoute(udpRoute)
	}

	return c.rules
}

func (c *GatewayProcessor) serviceConfigs() map[string]fgw.ServiceConfig {
	configs := make(map[string]fgw.ServiceConfig)
	enrichers := c.getServicePolicyEnrichers()

	for svcPortName, svcInfo := range c.services {
		svcKey := svcInfo.svcPortName.NamespacedName
		svc, err := c.cache.getServiceFromCache(svcKey)

		if err != nil {
			log.Error().Msgf("Failed to get Service %s: %s", svcKey, err)
			continue
		}

		if svc.Spec.Type == corev1.ServiceTypeExternalName {
			log.Warn().Msgf("Type of Service %s is %s, will be ignored", svcKey, corev1.ServiceTypeExternalName)
			continue
		}

		svcCfg := &fgw.ServiceConfig{
			Endpoints: c.calculateEndpoints(svc, svcInfo.svcPortName.Port),
		}

		for _, enricher := range enrichers {
			enricher.Enrich(svcPortName, svcCfg)
		}

		configs[svcPortName] = *svcCfg
	}

	return configs
}

func (c *GatewayProcessor) calculateEndpoints(svc *corev1.Service, port *int32) map[string]fgw.Endpoint {
	// If the Service is headless, use the Endpoints to get the list of backends
	if isHeadlessServiceWithoutSelector(svc) {
		return c.upstreamsByEndpoints(svc, port)
	}

	return c.upstreams(svc, port)
}

func (c *GatewayProcessor) upstreamsByEndpoints(svc *corev1.Service, port *int32) map[string]fgw.Endpoint {
	eps, err := c.cache.informers.GetListers().Endpoints.Endpoints(svc.Namespace).Get(svc.Name)
	if err != nil {
		log.Error().Msgf("Failed to get Endpoints of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(eps.Subsets) == 0 {
		return nil
	}

	svcPort, err := getServicePort(svc, port)
	if err != nil {
		log.Error().Msgf("Failed to get ServicePort: %s", err)
		return nil
	}

	endpointSet := make(map[endpointContext]struct{})
	for _, subset := range eps.Subsets {
		if endpointPort := findEndpointPort(subset.Ports, svcPort); endpointPort > 0 && endpointPort <= 65535 {
			for _, address := range subset.Addresses {
				ep := endpointContext{address: address.IP, port: endpointPort}
				endpointSet[ep] = struct{}{}
			}
		}
	}

	return toFGWEndpoints(endpointSet)
}

func (c *GatewayProcessor) upstreamsByEndpointSlices(svc *corev1.Service, port *int32) map[string]fgw.Endpoint {
	if len(svc.Annotations) > 0 {
		if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
			svcMeta := connector.Decode(svc, v)
			found := false
			for portMeta := range svcMeta.Ports {
				if uint16(portMeta) == uint16(*port) {
					found = true
					break
				}
			}
			if found {
				endpointSet := make(map[endpointContext]struct{})
				for address, metadata := range svcMeta.Endpoints {
					if len(metadata.Native.ViaGatewayHTTP) == 0 {
						ep := endpointContext{address: string(address), port: *port}
						endpointSet[ep] = struct{}{}
					} else {
						if segs := strings.Split(metadata.Native.ViaGatewayHTTP, ":"); len(segs) == 2 {
							if portStr, convErr := strconv.Atoi(segs[1]); convErr == nil {
								viaPort := int32(portStr & 0xFFFF)
								viaAddr := segs[0]
								ep := endpointContext{address: viaAddr, port: viaPort}
								endpointSet[ep] = struct{}{}
							}
						}
					}
				}
				if len(endpointSet) > 0 {
					return toFGWEndpoints(endpointSet)
				} else {
					log.Error().Msgf("no valid endpoints found for Service %s/%s and port %v", svc.Namespace, svc.Name, *port)
					return nil
				}
			}
		}
	}

	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			discoveryv1.LabelServiceName: svc.Name,
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to convert LabelSelector to Selector: %s", err)
		return nil
	}

	endpointSliceList, err := c.cache.informers.GetListers().EndpointSlice.EndpointSlices(svc.Namespace).List(selector)
	if err != nil {
		log.Error().Msgf("Failed to list EndpointSlice of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(endpointSliceList) == 0 {
		return nil
	}

	svcPort, err := getServicePort(svc, port)
	if err != nil {
		log.Error().Msgf("Failed to get ServicePort: %s", err)
		return nil
	}

	filteredSlices := filterEndpointSliceList(endpointSliceList, svcPort)
	if len(filteredSlices) == 0 {
		log.Error().Msgf("no valid endpoints found for Service %s/%s and port %v", svc.Namespace, svc.Name, svcPort)
		return nil
	}

	endpointSet := make(map[endpointContext]struct{})
	for _, eps := range filteredSlices {
		for _, endpoint := range eps.Endpoints {
			if !isEndpointReady(endpoint) {
				continue
			}

			if endpointPort := findEndpointSlicePort(eps.Ports, svcPort); endpointPort > 0 && endpointPort <= 65535 {
				for _, address := range endpoint.Addresses {
					ep := endpointContext{address: address, port: endpointPort}
					endpointSet[ep] = struct{}{}
				}
			}
		}
	}

	return toFGWEndpoints(endpointSet)
}

func (c *GatewayProcessor) defaults() fgw.Defaults {
	cfg := c.getConfig()

	ret := fgw.Defaults{
		EnableDebug:                    c.isDebugEnabled(),
		DefaultPassthroughUpstreamPort: cfg.GetFGWSSLPassthroughUpstreamPort(),
		StripAnyHostPort:               cfg.IsFGWStripAnyHostPort(),
		ProxyPreserveHost:              cfg.IsFGWProxyPreserveHost(),
		HTTP1PerRequestLoadBalancing:   cfg.IsFGWHTTP1PerRequestLoadBalancingEnabled(),
		HTTP2PerRequestLoadBalancing:   cfg.IsFGWHTTP2PerRequestLoadBalancingEnabled(),
		SocketTimeout:                  pointer.Int32(60),
	}

	if cfg.GetFeatureFlags().EnableGatewayProxyTag {
		ret.ProxyTag = &fgw.ProxyTag{
			SrcHostHeader: cfg.GetFGWProxyTag().SrcHostHeader,
			DstHostHeader: cfg.GetFGWProxyTag().DstHostHeader,
		}
	}

	return ret
}

func (c *GatewayProcessor) chains() fgw.Chains {
	featureFlags := c.getConfig().GetFeatureFlags()

	if featureFlags.EnableGatewayAgentService {
		return fgw.Chains{
			HTTPRoute:      insertAgentServiceScript(defaultHTTPChains),
			HTTPSRoute:     insertAgentServiceScript(defaultHTTPSChains),
			TLSPassthrough: defaultTLSPassthroughChains,
			TLSTerminate:   defaultTLSTerminateChains,
			TCPRoute:       defaultTCPChains,
			UDPRoute:       defaultUDPChains,
		}
	}

	if featureFlags.EnableGatewayProxyTag {
		return fgw.Chains{
			HTTPRoute:      insertProxyTagScript(defaultHTTPChains),
			HTTPSRoute:     insertProxyTagScript(defaultHTTPSChains),
			TLSPassthrough: defaultTLSPassthroughChains,
			TLSTerminate:   defaultTLSTerminateChains,
			TCPRoute:       defaultTCPChains,
			UDPRoute:       defaultUDPChains,
		}
	}

	return fgw.Chains{
		HTTPRoute:      defaultHTTPChains,
		HTTPSRoute:     defaultHTTPSChains,
		TLSPassthrough: defaultTLSPassthroughChains,
		TLSTerminate:   defaultTLSTerminateChains,
		TCPRoute:       defaultTCPChains,
		UDPRoute:       defaultUDPChains,
	}
}

func (c *GatewayProcessor) backendRefToServicePortName(referer client.Object, ref gwv1.BackendObjectReference) *fgw.ServicePortName {
	if !isValidBackendRefToGroupKindOfService(ref) {
		log.Error().Msgf("Unsupported backend group %s and kind %s for service", *ref.Group, *ref.Kind)
		return nil
	}

	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		c.referenceGrants,
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(*ref.Group),
			Kind:      string(*ref.Kind),
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
		return nil
	}

	return &fgw.ServicePortName{
		NamespacedName: types.NamespacedName{
			Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		},
		Port: pointer.Int32(int32(*ref.Port)),
	}
}

func (c *GatewayProcessor) targetRefToServicePortName(referer client.Object, ref gwv1alpha2.NamespacedPolicyTargetReference, port int32) *fgw.ServicePortName {
	if !isValidTargetRefToGroupKindOfService(ref) {
		log.Error().Msgf("Unsupported target group %s and kind %s for service", ref.Group, ref.Kind)
		return nil
	}

	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		c.referenceGrants,
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(ref.Group),
			Kind:      string(ref.Kind),
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(ref.Kind), string(ref.Group), string(*ref.Namespace), ref.Name)
		return nil
	}

	return &fgw.ServicePortName{
		NamespacedName: types.NamespacedName{
			Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		},
		Port: pointer.Int32(port),
	}
}

func (c *GatewayProcessor) toFSMHTTPRouteFilter(referer client.Object, filter gwv1.HTTPRouteFilter) fgw.Filter {
	result := fgw.HTTPRouteFilter{Type: filter.Type}

	if filter.RequestHeaderModifier != nil {
		result.RequestHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Add),
			Remove: filter.RequestHeaderModifier.Remove,
		}
	}

	if filter.ResponseHeaderModifier != nil {
		result.ResponseHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Add),
			Remove: filter.ResponseHeaderModifier.Remove,
		}
	}

	if filter.RequestRedirect != nil {
		result.RequestRedirect = &fgw.HTTPRequestRedirectFilter{
			Scheme:     filter.RequestRedirect.Scheme,
			Hostname:   toFSMHostname(filter.RequestRedirect.Hostname),
			Path:       toFSMHTTPPathModifier(filter.RequestRedirect.Path),
			Port:       toFSMPortNumber(filter.RequestRedirect.Port),
			StatusCode: filter.RequestRedirect.StatusCode,
		}
	}

	if filter.URLRewrite != nil {
		result.URLRewrite = &fgw.HTTPURLRewriteFilter{
			Hostname: toFSMHostname(filter.URLRewrite.Hostname),
			Path:     toFSMHTTPPathModifier(filter.URLRewrite.Path),
		}
	}

	if filter.RequestMirror != nil {
		if svcPort := c.backendRefToServicePortName(referer, filter.RequestMirror.BackendRef); svcPort != nil {
			result.RequestMirror = &fgw.HTTPRequestMirrorFilter{
				BackendService: svcPort.String(),
			}

			c.services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	// TODO: implement it later
	if filter.ExtensionRef != nil {
		result.ExtensionRef = filter.ExtensionRef
	}

	return result
}

func (c *GatewayProcessor) toFSMGRPCRouteFilter(referer client.Object, filter gwv1.GRPCRouteFilter) fgw.Filter {
	result := fgw.GRPCRouteFilter{Type: filter.Type}

	if filter.RequestHeaderModifier != nil {
		result.RequestHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.RequestHeaderModifier.Add),
			Remove: filter.RequestHeaderModifier.Remove,
		}
	}

	if filter.ResponseHeaderModifier != nil {
		result.ResponseHeaderModifier = &fgw.HTTPHeaderFilter{
			Set:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Set),
			Add:    toFSMHTTPHeaders(filter.ResponseHeaderModifier.Add),
			Remove: filter.ResponseHeaderModifier.Remove,
		}
	}

	if filter.RequestMirror != nil {
		if svcPort := c.backendRefToServicePortName(referer, filter.RequestMirror.BackendRef); svcPort != nil {
			result.RequestMirror = &fgw.HTTPRequestMirrorFilter{
				BackendService: svcPort.String(),
			}

			c.services[svcPort.String()] = serviceContext{
				svcPortName: *svcPort,
			}
		}
	}

	// TODO: implement it later
	if filter.ExtensionRef != nil {
		result.ExtensionRef = filter.ExtensionRef
	}

	return result
}

func (c *GatewayProcessor) secretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error) {
	if !isValidRefToGroupKindOfSecret(ref) {
		return nil, fmt.Errorf("unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind)
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		c.referenceGrants,
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     corev1.GroupName,
			Kind:      constants.KubernetesSecretKind,
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
	}

	return c.cache.getSecretFromCache(client.ObjectKey{
		Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
		Name:      string(ref.Name),
	})
}

func (c *GatewayProcessor) objectRefToCACertificate(referer client.Object, ref gwv1.ObjectReference) ([]byte, error) {
	if !isValidRefToGroupKindOfCA(ref) {
		return nil, fmt.Errorf("unsupported group %s and kind %s for secret", ref.Group, ref.Kind)
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		c.referenceGrants,
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     corev1.GroupName,
			Kind:      constants.KubernetesSecretKind,
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
	) {
		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(ref.Kind), string(ref.Group), string(*ref.Namespace), ref.Name)
	}

	ca := make([]byte, 0)

	switch ref.Kind {
	case constants.KubernetesSecretKind:
		secret, err := c.cache.getSecretFromCache(client.ObjectKey{
			Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		})
		if err != nil {
			return nil, err
		}

		caBytes, ok := secret.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, caBytes...)
		}
	case constants.KubernetesConfigMapKind:
		cm, err := c.cache.getConfigMapFromCache(client.ObjectKey{
			Namespace: gwutils.Namespace(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		})
		if err != nil {
			return nil, err
		}

		caBytes, ok := cm.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, []byte(caBytes)...)
		}
	}

	if len(ca) == 0 {
		return nil, fmt.Errorf("no CA certificate found in %s %s/%s", ref.Kind, gwutils.Namespace(ref.Namespace, referer.GetNamespace()), ref.Name)
	}

	return ca, nil
}
