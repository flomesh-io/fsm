package v2

import (
	"context"
	"fmt"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
	"github.com/flomesh-io/fsm/pkg/gateway/status"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/utils"
)

type ConfigGenerator struct {
	client              cache.Cache
	processor           processor.Processor
	gateway             *gwv1.Gateway
	secretFiles         map[string]string
	services            map[string]serviceContext
	filters             map[extv1alpha1.FilterProtocol]map[string]string
	upstreams           calculateBackendTargetsFunc
	backendTLSPolicies  map[string]*fgwv2.BackendTLSPolicy
	backendLBPolicies   map[string]*fgwv2.BackendLBPolicy
	healthCheckPolicies map[string]*fgwv2.HealthCheckPolicy
	retryPolicies       map[string]*fgwv2.RetryPolicy
}

func NewGatewayConfigGenerator(gateway *gwv1.Gateway, processor processor.Processor, client cache.Cache) processor.Generator {
	p := &ConfigGenerator{
		client:              client,
		processor:           processor,
		gateway:             gateway,
		secretFiles:         map[string]string{},
		services:            map[string]serviceContext{},
		filters:             map[extv1alpha1.FilterProtocol]map[string]string{},
		backendTLSPolicies:  map[string]*fgwv2.BackendTLSPolicy{},
		backendLBPolicies:   map[string]*fgwv2.BackendLBPolicy{},
		healthCheckPolicies: map[string]*fgwv2.HealthCheckPolicy{},
		retryPolicies:       map[string]*fgwv2.RetryPolicy{},
	}

	if processor.UseEndpointSlices() {
		p.upstreams = p.upstreamsByEndpointSlices
	} else {
		p.upstreams = p.upstreamsByEndpoints
	}

	return p
}

func (c *ConfigGenerator) Generate() fgwv2.Config {
	cfg := &fgwv2.ConfigSpec{
		Resources: c.processResources(),
		Secrets:   c.secretFiles,
		Filters:   c.filters,
	}
	cfg.Version = utils.SimpleHash(cfg)

	return cfg
}

func (c *ConfigGenerator) processResources() []interface{} {
	resources := make([]interface{}, 0)

	resources = append(resources, c.processGateway())
	resources = append(resources, c.processHTTPRoutes()...)
	resources = append(resources, c.processGRPCRoutes()...)
	resources = append(resources, c.processTLSRoutes()...)
	resources = append(resources, c.processTCPRoutes()...)
	resources = append(resources, c.processUDPRoutes()...)
	resources = append(resources, c.processBackends()...)

	for _, policy := range c.backendTLSPolicies {
		resources = append(resources, policy)
	}

	for _, policy := range c.backendLBPolicies {
		resources = append(resources, policy)
	}

	for _, policy := range c.healthCheckPolicies {
		resources = append(resources, policy)
	}

	for _, policy := range c.retryPolicies {
		resources = append(resources, policy)
	}

	return resources
}

func (c *ConfigGenerator) backendRefToServicePortName(route client.Object, backendRef gwv1.BackendObjectReference, rps status.RouteConditionAccessor) *fgwv2.ServicePortName {
	return gwutils.BackendRefToServicePortName(c.client, route, backendRef, rps)
}

//func (c *ConfigGenerator) targetRefToServicePortName(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference, port int32) *v2.ServicePortName {
//	return gwutils.HasAccessToBackendTargetRef(c.client, policy, targetRef, port)
//}

func (c *ConfigGenerator) secretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error) {
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
		gwutils.GetSecretRefGrants(c.client),
	) {
		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
	}

	return c.getSecretFromCache(client.ObjectKey{
		Namespace: gwutils.NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
		Name:      string(ref.Name),
	})
}

func (c *ConfigGenerator) objectRefToCACertificate(referer client.Object, ref gwv1.ObjectReference) []byte {
	return gwutils.ObjectRefToCACertificate(c.client, referer, ref, nil)
}

func (c *ConfigGenerator) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj := &corev1.Secret{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

//func (c *ConfigGenerator) getConfigMapFromCache(key client.ObjectKey) (*corev1.ConfigMap, error) {
//	obj := &corev1.ConfigMap{}
//	if err := c.client.Get(context.TODO(), key, obj); err != nil {
//		return nil, err
//	}
//
//	return obj, nil
//}

func (c *ConfigGenerator) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj := &corev1.Service{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}
