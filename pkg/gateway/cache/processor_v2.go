package cache

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/k8s"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/jinzhu/copier"
	corev1 "k8s.io/api/core/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/utils"
)

type GatewayProcessorV2 struct {
	cache       *GatewayCache
	gateway     *gwv1.Gateway
	secretFiles map[string]string
	services    map[string]serviceContextV2
	upstreams   calculateBackendTargetsFunc
}

func NewGatewayProcessorV2(cache *GatewayCache, gateway *gwv1.Gateway) *GatewayProcessorV2 {
	p := &GatewayProcessorV2{
		cache:       cache,
		gateway:     gateway,
		secretFiles: map[string]string{},
		services:    map[string]serviceContextV2{},
	}

	if cache.useEndpointSlices {
		p.upstreams = p.upstreamsByEndpointSlices
	} else {
		p.upstreams = p.upstreamsByEndpoints
	}

	return p
}

func (c *GatewayProcessorV2) build() *v2.Config {
	cfg := &v2.Config{
		Resources: c.processResources(),
		Secrets:   c.secretFiles,
	}
	cfg.Version = utils.SimpleHash(cfg)

	return cfg
}

func (c *GatewayProcessorV2) processResources() []interface{} {
	resources := make([]interface{}, 0)

	resources = append(resources, c.processGateway())
	resources = append(resources, c.processHTTPRoutes()...)
	resources = append(resources, c.processGRPCRoutes()...)
	resources = append(resources, c.processTLSRoutes()...)
	resources = append(resources, c.processTCPRoutes()...)
	resources = append(resources, c.processUDPRoutes()...)
	resources = append(resources, c.processBackends()...)

	return resources
}

func (c *GatewayProcessorV2) processGateway() *v2.Gateway {
	g2 := &v2.Gateway{}

	err := copier.CopyWithOption(g2, c.gateway, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	if err != nil {
		log.Error().Msgf("Failed to copy gateway: %v", err)
		return nil
	}

	// replace listeners with valid listeners
	g2.Spec.Listeners = make([]v2.Listener, 0)
	for _, l := range gwutils.GetValidListenersForGateway(c.gateway) {
		v2l := &v2.Listener{
			Name:     l.Name,
			Hostname: l.Hostname,
			Port:     l.Port,
			Protocol: l.Protocol,
		}

		if l.TLS != nil {
			v2l.TLS = &v2.GatewayTLSConfig{
				Mode:         l.TLS.Mode,
				Certificates: []map[string]string{},
				Options:      l.TLS.Options,
			}

			if l.TLS.FrontendValidation != nil {
				v2l.TLS.FrontendValidation = &v2.FrontendTLSValidation{CACertificates: []map[string]string{}}
			}
		}

		// get certificates and CA certificates
		if c.tls(l) && v2l.TLS != nil {
			c.processCertificates(l, v2l)
			c.processCACerts(l, v2l)
		}

		g2.Spec.Listeners = append(g2.Spec.Listeners, *v2l)
	}

	return g2
}

func (c *GatewayProcessorV2) tls(l gwtypes.Listener) bool {
	switch l.Protocol {
	case gwv1.HTTPSProtocolType:
		// Terminate
		if l.TLS != nil {
			if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
				return true
			}
		}
	case gwv1.TLSProtocolType:
		// Terminate & Passthrough
		if l.TLS != nil {
			if l.TLS.Mode == nil {
				return true
			}

			switch *l.TLS.Mode {
			case gwv1.TLSModeTerminate:
				return true
			}
		}
	}

	return false
}

func (c *GatewayProcessorV2) processCertificates(l gwtypes.Listener, v2l *v2.Listener) {
	for index, ref := range l.TLS.CertificateRefs {
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

		certName := fmt.Sprintf("tls-%d-%d.crt", l.Port, index)
		keyName := fmt.Sprintf("tls-%d-%d.key", l.Port, index)

		v2l.TLS.Certificates = append(v2l.TLS.Certificates, map[string]string{
			corev1.TLSCertKey:       certName,
			corev1.TLSPrivateKeyKey: keyName,
		})

		c.secretFiles[certName] = string(secret.Data[corev1.TLSCertKey])
		c.secretFiles[keyName] = string(secret.Data[corev1.TLSPrivateKeyKey])
	}
}

func (c *GatewayProcessorV2) processCACerts(l gwtypes.Listener, v2l *v2.Listener) {
	if l.TLS.FrontendValidation != nil && len(l.TLS.FrontendValidation.CACertificateRefs) > 0 {
		for index, ref := range l.TLS.FrontendValidation.CACertificateRefs {
			ca, err := c.objectRefToCACertificate(c.gateway, ref)

			if err != nil {
				log.Error().Msgf("Failed to resolve CA Certificate: %s", err)
				continue
			}

			caName := fmt.Sprintf("ca-%d-%d.crt", l.Port, index)

			v2l.TLS.FrontendValidation.CACertificates = append(v2l.TLS.FrontendValidation.CACertificates, map[string]string{
				corev1.ServiceAccountRootCAKey: caName,
			})

			c.secretFiles[caName] = string(ca)
		}
	}
}

func (c *GatewayProcessorV2) processBackends() []interface{} {
	//configs := make(map[string]fgw.ServiceConfig)
	backends := make([]interface{}, 0)
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

		// don't create Backend resource if there are no endpoints
		targets := c.calculateEndpoints(svc, svcInfo.svcPortName.Port)
		if len(targets) == 0 {
			continue
		}

		//for _, enricher := range c.getServicePolicyEnrichers(svc) {
		//    enricher.Enrich(svcPortName, svcCfg)
		//}

		backends = append(backends, &v2.Backend{
			Kind: "Backend",
			ObjectMeta: v2.ObjectMeta{
				Name: svcPortName,
			},
			Spec: v2.BackendSpec{
				Targets: targets,
			},
		})
	}

	return backends
}

func (c *GatewayProcessorV2) calculateEndpoints(svc *corev1.Service, port *int32) []v2.BackendTarget {
	// If the Service is headless, use the Endpoints to get the list of backends
	if k8s.IsHeadlessService(*svc) {
		return c.upstreamsByEndpoints(svc, port)
	}

	return c.upstreams(svc, port)
}

func (c *GatewayProcessorV2) upstreamsByEndpoints(svc *corev1.Service, port *int32) []v2.BackendTarget {
	eps := &corev1.Endpoints{}
	if err := c.cache.client.Get(context.TODO(), client.ObjectKeyFromObject(svc), eps); err != nil {
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

	return toFGWBackendTargets(endpointSet)
}

func (c *GatewayProcessorV2) upstreamsByEndpointSlices(svc *corev1.Service, port *int32) []v2.BackendTarget {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			discoveryv1.LabelServiceName: svc.Name,
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to convert LabelSelector to Selector: %s", err)
		return nil
	}

	endpointSliceList := &discoveryv1.EndpointSliceList{}
	if err := c.cache.client.List(context.TODO(), endpointSliceList, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error().Msgf("Failed to list EndpointSlice of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(endpointSliceList.Items) == 0 {
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

	return toFGWBackendTargets(endpointSet)
}

func (c *GatewayProcessorV2) backendRefToServicePortName(referer client.Object, ref gwv1.BackendObjectReference) *v2.ServicePortName {
	if !gwutils.IsValidBackendRefToGroupKindOfService(ref) {
		log.Error().Msgf("Unsupported backend group %s and kind %s for service", *ref.Group, *ref.Kind)
		return nil
	}

	if ref.Port == nil {
		log.Warn().Msgf("Port is not specified in the backend reference %s/%s when the referent is a Kubernetes Service", gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()), ref.Name)
		return nil
	}

	gvk := referer.GetObjectKind().GroupVersionKind()
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     gvk.Group,
			Kind:      gvk.Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(*ref.Group),
			Kind:      string(*ref.Kind),
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
		gwutils.GetServiceRefGrants(c.cache.client),
	) {
		log.Error().Msgf("Cross-namespace reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			gvk.Kind, gvk.Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
		return nil
	}

	return &v2.ServicePortName{
		NamespacedName: types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		},
		Port: ptr.To(int32(*ref.Port)),
	}
}

func (c *GatewayProcessorV2) secretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error) {
	if !gwutils.IsValidRefToGroupKindOfSecret(ref) {
		return nil, fmt.Errorf("unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind)
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
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
		gwutils.GetSecretRefGrants(c.cache.client),
	) {
		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
	}

	return c.cache.getSecretFromCache(client.ObjectKey{
		Namespace: gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
		Name:      string(ref.Name),
	})
}

func (c *GatewayProcessorV2) objectRefToCACertificate(referer client.Object, ref gwv1.ObjectReference) ([]byte, error) {
	if !gwutils.IsValidRefToGroupKindOfCA(ref) {
		return nil, fmt.Errorf("unsupported group %s and kind %s for secret", ref.Group, ref.Kind)
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !gwutils.ValidCrossNamespaceRef(
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
		gwutils.GetCARefGrants(c.cache.client),
	) {
		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(ref.Kind), string(ref.Group), string(*ref.Namespace), ref.Name)
	}

	ca := make([]byte, 0)

	switch ref.Kind {
	case constants.KubernetesSecretKind:
		secret, err := c.cache.getSecretFromCache(client.ObjectKey{
			Namespace: gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
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
			Namespace: gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
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
		return nil, fmt.Errorf("no CA certificate found in %s %s/%s", ref.Kind, gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()), ref.Name)
	}

	return ca, nil
}
