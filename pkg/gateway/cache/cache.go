// Package cache implements a cache of all the resources that are relevant to the gateway
package cache

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"k8s.io/client-go/kubernetes"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// GatewayCache is a cache of all the resources that are relevant to the gateway
type GatewayCache struct {
	repoClient   *repo.PipyRepoClient
	informers    *informers.InformerCollection
	kubeClient   kubernetes.Interface
	cfg          configurator.Configurator
	triggers     map[ResourceType]Trigger
	gatewayclass *gwv1beta1.GatewayClass
	mutex        *sync.RWMutex
}

// NewGatewayCache creates a new gateway cache
func NewGatewayCache(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, cfg configurator.Configurator) *GatewayCache {
	repoBaseURL := fmt.Sprintf("%s://%s:%d", "http", cfg.GetRepoServerIPAddr(), cfg.GetProxyServerPort())
	return &GatewayCache{
		repoClient: repo.NewRepoClient(repoBaseURL, cfg.GetFSMLogLevel()),
		informers:  informerCollection,
		kubeClient: kubeClient,
		cfg:        cfg,

		triggers: map[ResourceType]Trigger{
			EndpointsResourceType:               &EndpointsTrigger{},
			ServicesResourceType:                &ServicesTrigger{},
			ServiceImportsResourceType:          &ServiceImportsTrigger{},
			EndpointSlicesResourceType:          &EndpointSlicesTrigger{},
			SecretsResourceType:                 &SecretTrigger{},
			GatewayClassesResourceType:          &GatewayClassesTrigger{},
			GatewaysResourceType:                &GatewaysTrigger{},
			HTTPRoutesResourceType:              &HTTPRoutesTrigger{},
			GRPCRoutesResourceType:              &GRPCRoutesTrigger{},
			TCPRoutesResourceType:               &TCPRoutesTrigger{},
			TLSRoutesResourceType:               &TLSRoutesTrigger{},
			UDPRoutesResourceType:               &UDPRoutesTrigger{},
			RateLimitPoliciesResourceType:       &RateLimitPoliciesTrigger{},
			SessionStickyPoliciesResourceType:   &SessionStickyPoliciesTrigger{},
			LoadBalancerPoliciesResourceType:    &LoadBalancerPoliciesTrigger{},
			CircuitBreakingPoliciesResourceType: &CircuitBreakingPoliciesTrigger{},
			AccessControlPoliciesResourceType:   &AccessControlPoliciesTrigger{},
			HealthCheckPoliciesResourceType:     &HealthCheckPoliciesTrigger{},
			FaultInjectionPoliciesResourceType:  &FaultInjectionPoliciesTrigger{},
			UpstreamTLSPoliciesResourceType:     &UpstreamTLSPoliciesTrigger{},
			RetryPoliciesResourceType:           &RetryPoliciesTrigger{},
			GatewayTLSPoliciesResourceType:      &GatewayTLSPoliciesTrigger{},
		},

		mutex: new(sync.RWMutex),
	}
}

// Insert inserts an object into the cache
func (c *GatewayCache) Insert(obj interface{}) bool {
	p := c.getTrigger(obj)
	if p != nil {
		return p.Insert(obj, c)
	}

	return false
}

// Delete deletes an object from the cache
func (c *GatewayCache) Delete(obj interface{}) bool {
	p := c.getTrigger(obj)
	if p != nil {
		return p.Delete(obj, c)
	}

	return false
}

func (c *GatewayCache) getTrigger(obj interface{}) Trigger {
	switch obj.(type) {
	case *corev1.Endpoints:
		return c.triggers[EndpointsResourceType]
	case *corev1.Service:
		return c.triggers[ServicesResourceType]
	case *mcsv1alpha1.ServiceImport:
		return c.triggers[ServiceImportsResourceType]
	case *discoveryv1.EndpointSlice:
		return c.triggers[EndpointSlicesResourceType]
	case *corev1.Secret:
		return c.triggers[SecretsResourceType]
	case *gwv1beta1.GatewayClass:
		return c.triggers[GatewayClassesResourceType]
	case *gwv1beta1.Gateway:
		return c.triggers[GatewaysResourceType]
	case *gwv1beta1.HTTPRoute:
		return c.triggers[HTTPRoutesResourceType]
	case *gwv1alpha2.GRPCRoute:
		return c.triggers[GRPCRoutesResourceType]
	case *gwv1alpha2.TCPRoute:
		return c.triggers[TCPRoutesResourceType]
	case *gwv1alpha2.TLSRoute:
		return c.triggers[TLSRoutesResourceType]
	case *gwv1alpha2.UDPRoute:
		return c.triggers[UDPRoutesResourceType]
	case *gwpav1alpha1.RateLimitPolicy:
		return c.triggers[RateLimitPoliciesResourceType]
	case *gwpav1alpha1.SessionStickyPolicy:
		return c.triggers[SessionStickyPoliciesResourceType]
	case *gwpav1alpha1.LoadBalancerPolicy:
		return c.triggers[LoadBalancerPoliciesResourceType]
	case *gwpav1alpha1.CircuitBreakingPolicy:
		return c.triggers[CircuitBreakingPoliciesResourceType]
	case *gwpav1alpha1.AccessControlPolicy:
		return c.triggers[AccessControlPoliciesResourceType]
	case *gwpav1alpha1.HealthCheckPolicy:
		return c.triggers[HealthCheckPoliciesResourceType]
	case *gwpav1alpha1.FaultInjectionPolicy:
		return c.triggers[FaultInjectionPoliciesResourceType]
	case *gwpav1alpha1.UpstreamTLSPolicy:
		return c.triggers[UpstreamTLSPoliciesResourceType]
	case *gwpav1alpha1.RetryPolicy:
		return c.triggers[RetryPoliciesResourceType]
	case *gwpav1alpha1.GatewayTLSPolicy:
		return c.triggers[GatewayTLSPoliciesResourceType]
	}

	return nil
}
