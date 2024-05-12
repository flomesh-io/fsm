// Package cache implements a cache of all the resources that are relevant to the gateway
package cache

import (
	"fmt"
	"sync"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"k8s.io/client-go/kubernetes"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

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
	triggers     map[informers.ResourceType]Trigger
	gatewayclass *gwv1.GatewayClass
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

		triggers: map[informers.ResourceType]Trigger{
			informers.EndpointsResourceType:               &EndpointsTrigger{},
			informers.ServicesResourceType:                &ServicesTrigger{},
			informers.ServiceImportsResourceType:          &ServiceImportsTrigger{},
			informers.EndpointSlicesResourceType:          &EndpointSlicesTrigger{},
			informers.SecretsResourceType:                 &SecretTrigger{},
			informers.ConfigMapsResourceType:              &ConfigMapTrigger{},
			informers.GatewayClassesResourceType:          &GatewayClassesTrigger{},
			informers.GatewaysResourceType:                &GatewaysTrigger{},
			informers.HTTPRoutesResourceType:              &HTTPRoutesTrigger{},
			informers.GRPCRoutesResourceType:              &GRPCRoutesTrigger{},
			informers.TCPRoutesResourceType:               &TCPRoutesTrigger{},
			informers.TLSRoutesResourceType:               &TLSRoutesTrigger{},
			informers.UDPRoutesResourceType:               &UDPRoutesTrigger{},
			informers.ReferenceGrantResourceType:          &ReferenceGrantTrigger{},
			informers.RateLimitPoliciesResourceType:       &RateLimitPoliciesTrigger{},
			informers.SessionStickyPoliciesResourceType:   &SessionStickyPoliciesTrigger{},
			informers.LoadBalancerPoliciesResourceType:    &LoadBalancerPoliciesTrigger{},
			informers.CircuitBreakingPoliciesResourceType: &CircuitBreakingPoliciesTrigger{},
			informers.AccessControlPoliciesResourceType:   &AccessControlPoliciesTrigger{},
			informers.HealthCheckPoliciesResourceType:     &HealthCheckPoliciesTrigger{},
			informers.FaultInjectionPoliciesResourceType:  &FaultInjectionPoliciesTrigger{},
			informers.UpstreamTLSPoliciesResourceType:     &UpstreamTLSPoliciesTrigger{},
			informers.RetryPoliciesResourceType:           &RetryPoliciesTrigger{},
			informers.GatewayTLSPoliciesResourceType:      &GatewayTLSPoliciesTrigger{},
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
	case *gwpav1alpha1.RateLimitPolicy:
		return c.triggers[informers.RateLimitPoliciesResourceType]
	case *gwpav1alpha1.SessionStickyPolicy:
		return c.triggers[informers.SessionStickyPoliciesResourceType]
	case *gwpav1alpha1.LoadBalancerPolicy:
		return c.triggers[informers.LoadBalancerPoliciesResourceType]
	case *gwpav1alpha1.CircuitBreakingPolicy:
		return c.triggers[informers.CircuitBreakingPoliciesResourceType]
	case *gwpav1alpha1.AccessControlPolicy:
		return c.triggers[informers.AccessControlPoliciesResourceType]
	case *gwpav1alpha1.HealthCheckPolicy:
		return c.triggers[informers.HealthCheckPoliciesResourceType]
	case *gwpav1alpha1.FaultInjectionPolicy:
		return c.triggers[informers.FaultInjectionPoliciesResourceType]
	case *gwpav1alpha1.UpstreamTLSPolicy:
		return c.triggers[informers.UpstreamTLSPoliciesResourceType]
	case *gwpav1alpha1.RetryPolicy:
		return c.triggers[informers.RetryPoliciesResourceType]
	case *gwpav1alpha1.GatewayTLSPolicy:
		return c.triggers[informers.GatewayTLSPoliciesResourceType]
	}

	return nil
}
