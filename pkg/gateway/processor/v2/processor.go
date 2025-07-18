// Package v2 implements a processor of all the resources that are relevant to the gateway
package v2

import (
	"fmt"

	"github.com/flomesh-io/fsm/pkg/utils"

	"sync"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	extensiontrigger "github.com/flomesh-io/fsm/pkg/gateway/processor/triggers/extension"
	gatewaytrigger "github.com/flomesh-io/fsm/pkg/gateway/processor/triggers/gateway"
	k8strigger "github.com/flomesh-io/fsm/pkg/gateway/processor/triggers/k8s"
	policytriggerv2 "github.com/flomesh-io/fsm/pkg/gateway/processor/triggers/policy/v2"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	cctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/version"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// GatewayProcessor is a processor of all the resources that are relevant to the gateway
type GatewayProcessor struct {
	repoClient        *repo.PipyRepoClient
	client            cache.Cache
	cfg               configurator.Configurator
	triggers          map[informers.ResourceType]processor.Trigger
	mutex             *sync.RWMutex
	useEndpointSlices bool
	gatewayFilesHash  map[string]map[string]string
}

// NewGatewayProcessor creates a new gateway processor
//
//gocyclo:ignore
func NewGatewayProcessor(ctx *cctx.ControllerContext) *GatewayProcessor {
	cfg := ctx.Configurator
	repoBaseURL := fmt.Sprintf("%s://%s:%d", "http", cfg.GetRepoServerIPAddr(), cfg.GetProxyServerPort())
	useEndpointSlices := cfg.GetFeatureFlags().UseEndpointSlicesForGateway && version.IsEndpointSliceEnabled(ctx.KubeClient)
	return &GatewayProcessor{
		repoClient: repo.NewRepoClient(repoBaseURL, cfg.GetFSMLogLevel()),
		client:     ctx.Manager.GetCache(),
		cfg:        cfg,

		triggers: map[informers.ResourceType]processor.Trigger{
			informers.EndpointsResourceType:               &k8strigger.EndpointsTrigger{},
			informers.ServicesResourceType:                &k8strigger.ServicesTrigger{},
			informers.ServiceImportsResourceType:          &k8strigger.ServiceImportsTrigger{},
			informers.EndpointSlicesResourceType:          &k8strigger.EndpointSlicesTrigger{},
			informers.SecretsResourceType:                 &k8strigger.SecretTrigger{},
			informers.ConfigMapsResourceType:              &k8strigger.ConfigMapTrigger{},
			informers.GatewayClassesResourceType:          &gatewaytrigger.GatewayClassesTrigger{},
			informers.GatewaysResourceType:                &gatewaytrigger.GatewaysTrigger{},
			informers.HTTPRoutesResourceType:              &gatewaytrigger.HTTPRoutesTrigger{},
			informers.GRPCRoutesResourceType:              &gatewaytrigger.GRPCRoutesTrigger{},
			informers.TCPRoutesResourceType:               &gatewaytrigger.TCPRoutesTrigger{},
			informers.TLSRoutesResourceType:               &gatewaytrigger.TLSRoutesTrigger{},
			informers.UDPRoutesResourceType:               &gatewaytrigger.UDPRoutesTrigger{},
			informers.ReferenceGrantResourceType:          &gatewaytrigger.ReferenceGrantTrigger{},
			informers.HealthCheckPoliciesResourceType:     &policytriggerv2.HealthCheckPoliciesTrigger{},
			informers.BackendLBPoliciesResourceType:       &policytriggerv2.BackendLBPoliciesTrigger{},
			informers.BackendTLSPoliciesResourceType:      &policytriggerv2.BackendTLSPoliciesTrigger{},
			informers.RouteRuleFilterPoliciesResourceType: &policytriggerv2.RouteRuleFilterPoliciesTrigger{},
			informers.FiltersResourceType:                 &extensiontrigger.FilterTrigger{},
			informers.ListenerFiltersResourceType:         &extensiontrigger.ListenerFilterTrigger{},
			informers.FilterDefinitionsResourceType:       &extensiontrigger.FilterDefinitionTrigger{},
			informers.CircuitBreakersResourceType:         &extensiontrigger.CircuitBreakerTrigger{},
			informers.FaultInjectionsResourceType:         &extensiontrigger.FaultInjectionTrigger{},
			informers.RateLimitsResourceType:              &extensiontrigger.RateLimitTrigger{},
			informers.HTTPLogsResourceType:                &extensiontrigger.HTTPLogTrigger{},
			informers.MetricsResourceType:                 &extensiontrigger.MetricsTrigger{},
			informers.ZipkinResourceType:                  &extensiontrigger.ZipkinTrigger{},
			informers.FilterConfigsResourceType:           &extensiontrigger.FilterConfigTrigger{},
			informers.ProxyTagResourceType:                &extensiontrigger.ProxyTagTrigger{},
			informers.IPRestrictionResourceType:           &extensiontrigger.IPRestrictionTrigger{},
			informers.ExternalRateLimitResourceType:       &extensiontrigger.ExternalRateLimitTrigger{},
			informers.RequestTerminationResourceType:      &extensiontrigger.RequestTerminationTrigger{},
			informers.ConcurrencyLimitResourceType:        &extensiontrigger.ConcurrencyLimitTrigger{},
			informers.DNSModifierResourceType:             &extensiontrigger.DNSModifierTrigger{},
		},

		mutex:             new(sync.RWMutex),
		useEndpointSlices: useEndpointSlices,
		gatewayFilesHash:  make(map[string]map[string]string),
	}
}

// Insert inserts an object into the processor
func (c *GatewayProcessor) Insert(obj interface{}) bool {
	p := c.getTrigger(obj)
	if p != nil {
		return p.Insert(obj, c)
	}

	return false
}

// Delete deletes an object from the processor
func (c *GatewayProcessor) Delete(obj interface{}) bool {
	p := c.getTrigger(obj)
	if p != nil {
		return p.Delete(obj, c)
	}

	return false
}

//gocyclo:ignore
func (c *GatewayProcessor) getTrigger(obj interface{}) processor.Trigger {
	switch obj.(type) {
	case *corev1.Endpoints:
		return c.triggers[informers.EndpointsResourceType]
	case *corev1.Service:
		return c.triggers[informers.ServicesResourceType]
	case *mcsv1alpha1.ServiceImport:
		return c.triggers[informers.ServiceImportsResourceType]
	case *discoveryv1.EndpointSlice:
		return c.triggers[informers.EndpointSlicesResourceType]
	case *corev1.Secret:
		return c.triggers[informers.SecretsResourceType]
	case *corev1.ConfigMap:
		return c.triggers[informers.ConfigMapsResourceType]
	case *gwv1.GatewayClass:
		return c.triggers[informers.GatewayClassesResourceType]
	case *gwv1.Gateway:
		return c.triggers[informers.GatewaysResourceType]
	case *gwv1.HTTPRoute:
		return c.triggers[informers.HTTPRoutesResourceType]
	case *gwv1.GRPCRoute:
		return c.triggers[informers.GRPCRoutesResourceType]
	case *gwv1alpha2.TCPRoute:
		return c.triggers[informers.TCPRoutesResourceType]
	case *gwv1alpha2.TLSRoute:
		return c.triggers[informers.TLSRoutesResourceType]
	case *gwv1alpha2.UDPRoute:
		return c.triggers[informers.UDPRoutesResourceType]
	case *gwv1beta1.ReferenceGrant:
		return c.triggers[informers.ReferenceGrantResourceType]
	case *gwpav1alpha2.HealthCheckPolicy:
		return c.triggers[informers.HealthCheckPoliciesResourceType]
	case *gwpav1alpha2.BackendLBPolicy:
		return c.triggers[informers.BackendLBPoliciesResourceType]
	case *gwv1alpha3.BackendTLSPolicy:
		return c.triggers[informers.BackendTLSPoliciesResourceType]
	case *gwpav1alpha2.RouteRuleFilterPolicy:
		return c.triggers[informers.RouteRuleFilterPoliciesResourceType]
	case *extv1alpha1.Filter:
		return c.triggers[informers.FiltersResourceType]
	case *extv1alpha1.ListenerFilter:
		return c.triggers[informers.ListenerFiltersResourceType]
	case *extv1alpha1.FilterDefinition:
		return c.triggers[informers.FilterDefinitionsResourceType]
	case *extv1alpha1.CircuitBreaker:
		return c.triggers[informers.CircuitBreakersResourceType]
	case *extv1alpha1.FaultInjection:
		return c.triggers[informers.FaultInjectionsResourceType]
	case *extv1alpha1.RateLimit:
		return c.triggers[informers.RateLimitsResourceType]
	case *extv1alpha1.HTTPLog:
		return c.triggers[informers.HTTPLogsResourceType]
	case *extv1alpha1.Metrics:
		return c.triggers[informers.MetricsResourceType]
	case *extv1alpha1.Zipkin:
		return c.triggers[informers.ZipkinResourceType]
	case *extv1alpha1.FilterConfig:
		return c.triggers[informers.FilterConfigsResourceType]
	case *extv1alpha1.ProxyTag:
		return c.triggers[informers.ProxyTagResourceType]
	case *extv1alpha1.IPRestriction:
		return c.triggers[informers.IPRestrictionResourceType]
	case *extv1alpha1.ExternalRateLimit:
		return c.triggers[informers.ExternalRateLimitResourceType]
	case *extv1alpha1.RequestTermination:
		return c.triggers[informers.RequestTerminationResourceType]
	case *extv1alpha1.ConcurrencyLimit:
		return c.triggers[informers.ConcurrencyLimitResourceType]
	case *extv1alpha1.DNSModifier:
		return c.triggers[informers.DNSModifierResourceType]
	}

	return nil
}

func (c *GatewayProcessor) UseEndpointSlices() bool {
	return c.useEndpointSlices
}

func (c *GatewayProcessor) OnDeleteGateway(gateway *gwv1.Gateway) bool {
	gatewayPath := utils.GatewayCodebasePath(gateway.Namespace, gateway.Name)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Info().Msgf("Cleaning up files hash of Gateway %s", gatewayPath)
	delete(c.gatewayFilesHash, gatewayPath)

	return true
}
