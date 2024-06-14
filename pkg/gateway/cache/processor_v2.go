package cache

import (
	"fmt"

	"github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

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
	policies    []interface{}
}

func NewGatewayProcessorV2(cache *GatewayCache, gateway *gwv1.Gateway) Processor {
	p := &GatewayProcessorV2{
		cache:       cache,
		gateway:     gateway,
		secretFiles: map[string]string{},
		services:    map[string]serviceContextV2{},
		policies:    []interface{}{},
	}

	if cache.useEndpointSlices {
		p.upstreams = p.upstreamsByEndpointSlices
	} else {
		p.upstreams = p.upstreamsByEndpoints
	}

	return p
}

func (c *GatewayProcessorV2) Build() fgw.Config {
	cfg := &v2.ConfigSpec{
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
	resources = append(resources, c.policies...)

	return resources
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
