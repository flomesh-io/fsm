package v2

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/configurator"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"github.com/flomesh-io/fsm/pkg/utils"
)

type ConfigGenerator struct {
	client              cache.Cache
	processor           processor.Processor
	cfg                 configurator.Configurator
	gateway             *gwv1.Gateway
	secretFiles         map[string]string
	backends            map[string]*fgwv2.Backend
	filters             map[extv1alpha1.FilterProtocol]map[extv1alpha1.FilterType]string
	upstreams           calculateBackendTargetsFunc
	backendTLSPolicies  map[string]*fgwv2.BackendTLSPolicy
	backendLBPolicies   map[string]*fgwv2.BackendLBPolicy
	healthCheckPolicies map[string]*fgwv2.HealthCheckPolicy
}

func NewGatewayConfigGenerator(gateway *gwv1.Gateway, processor processor.Processor, client cache.Cache, mc configurator.Configurator) processor.Generator {
	p := &ConfigGenerator{
		client:              client,
		processor:           processor,
		cfg:                 mc,
		gateway:             gateway,
		secretFiles:         map[string]string{},
		backends:            map[string]*fgwv2.Backend{},
		filters:             map[extv1alpha1.FilterProtocol]map[extv1alpha1.FilterType]string{},
		backendTLSPolicies:  map[string]*fgwv2.BackendTLSPolicy{},
		backendLBPolicies:   map[string]*fgwv2.BackendLBPolicy{},
		healthCheckPolicies: map[string]*fgwv2.HealthCheckPolicy{},
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

func (c *ConfigGenerator) processResources() []fgwv2.Resource {
	resources := make([]fgwv2.Resource, 0)

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

	return resources
}

func (c *ConfigGenerator) backendRefToServicePortName(route client.Object, backendRef gwv1.BackendObjectReference) *fgwv2.ServicePortName {
	return gwutils.BackendRefToServicePortName(c.client, route, backendRef, func(gwv1.RouteConditionReason, string) {})
}

func (c *ConfigGenerator) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj := &corev1.Service{}
	if err := c.client.Get(context.TODO(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}
