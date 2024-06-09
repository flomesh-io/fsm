package cache

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"

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
	cache          *GatewayCache
	gateway        *gwv1.Gateway
	validListeners []gwtypes.Listener
	resources      []interface{}
	secretFiles    map[string]string
}

func NewGatewayProcessorV2(cache *GatewayCache, gateway *gwv1.Gateway) *GatewayProcessorV2 {
	return &GatewayProcessorV2{
		cache:          cache,
		gateway:        gateway,
		validListeners: gwutils.GetValidListenersForGateway(gateway),
		resources:      []interface{}{},
		secretFiles:    map[string]string{},
	}
}

func (c *GatewayProcessorV2) build() *v2.Config {
	gateway := c.processGateway()
	resources := c.processResources()

	cfg := &v2.Config{
		Gateway:     gateway,
		Resources:   resources,
		SecretFiles: c.secretFiles,
	}
	cfg.Version = utils.SimpleHash(cfg)

	return cfg
}

func (c *GatewayProcessorV2) processGateway() *v2.Gateway {
	g2 := &v2.Gateway{}

	err := copier.CopyWithOption(g2, c.gateway, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	if err != nil {
		log.Error().Msgf("Failed to copy gateway: %v", err)
		return &v2.Gateway{}
	}

	// replace listeners with valid listeners
	g2.Spec.Listeners = make([]v2.Listener, 0)
	for _, l := range c.validListeners {
		v2l := &v2.Listener{
			Name:          l.Name,
			Hostname:      l.Hostname,
			Port:          l.Port,
			Protocol:      l.Protocol,
			AllowedRoutes: l.AllowedRoutes,
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

func (c *GatewayProcessorV2) processResources() []interface{} {
	c.processHTTPRoutes()
	c.processGRPCRoutes()
	c.processTLSRoutes()
	c.processTCPRoutes()
	c.processUDPRoutes()

	return c.resources
}

func (c *GatewayProcessorV2) backendRefToServicePortName(referer client.Object, ref gwv1.BackendObjectReference) *fgw.ServicePortName {
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

	return &fgw.ServicePortName{
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
