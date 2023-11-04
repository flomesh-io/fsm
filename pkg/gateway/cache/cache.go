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
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"
)

// GatewayCache is a cache of all the resources that are relevant to the gateway
type GatewayCache struct {
	repoClient *repo.PipyRepoClient
	informers  *informers.InformerCollection
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator

	processors map[TriggerType]Processor

	gatewayclass     *gwv1beta1.GatewayClass
	gateways         map[string]client.ObjectKey // ns -> gateway
	services         map[client.ObjectKey]struct{}
	serviceimports   map[client.ObjectKey]struct{}
	endpoints        map[client.ObjectKey]struct{}
	endpointslices   map[client.ObjectKey]map[client.ObjectKey]struct{} // svc -> endpointslices
	secrets          map[client.ObjectKey]struct{}
	httproutes       map[client.ObjectKey]struct{}
	grpcroutes       map[client.ObjectKey]struct{}
	tcproutes        map[client.ObjectKey]struct{}
	tlsroutes        map[client.ObjectKey]struct{}
	ratelimits       map[client.ObjectKey]struct{}
	sessionstickies  map[client.ObjectKey]struct{}
	loadbalancers    map[client.ObjectKey]struct{}
	circuitbreakings map[client.ObjectKey]struct{}
	accesscontrols   map[client.ObjectKey]struct{}
	healthchecks     map[client.ObjectKey]struct{}
	faultinjections  map[client.ObjectKey]struct{}

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

		processors: map[TriggerType]Processor{
			//EndpointsTriggerType:      &EndpointsTrigger{},
			ServicesTriggerType:                &ServicesTrigger{},
			ServiceImportsTriggerType:          &ServiceImportsTrigger{},
			EndpointSlicesTriggerType:          &EndpointSlicesTrigger{},
			SecretsTriggerType:                 &SecretTrigger{},
			GatewayClassesTriggerType:          &GatewayClassesTrigger{},
			GatewaysTriggerType:                &GatewaysTrigger{},
			HTTPRoutesTriggerType:              &HTTPRoutesTrigger{},
			GRPCRoutesTriggerType:              &GRPCRoutesTrigger{},
			TCPRoutesTriggerType:               &TCPRoutesTrigger{},
			TLSRoutesTriggerType:               &TLSRoutesTrigger{},
			RateLimitPoliciesTriggerType:       &RateLimitPoliciesTrigger{},
			SessionStickyPoliciesTriggerType:   &SessionStickyPoliciesTrigger{},
			LoadBalancerPoliciesTriggerType:    &LoadBalancerPoliciesTrigger{},
			CircuitBreakingPoliciesTriggerType: &CircuitBreakingPoliciesTrigger{},
			AccessControlPoliciesTriggerType:   &AccessControlPoliciesTrigger{},
			HealthCheckPoliciesTriggerType:     &HealthCheckPoliciesTrigger{},
			FaultInjectionPoliciesTriggerType:  &FaultInjectionPoliciesTrigger{},
		},

		//endpoints:      make(map[client.ObjectKey]struct{}),
		gateways:         make(map[string]client.ObjectKey),
		services:         make(map[client.ObjectKey]struct{}),
		serviceimports:   make(map[client.ObjectKey]struct{}),
		endpointslices:   make(map[client.ObjectKey]map[client.ObjectKey]struct{}),
		secrets:          make(map[client.ObjectKey]struct{}),
		httproutes:       make(map[client.ObjectKey]struct{}),
		grpcroutes:       make(map[client.ObjectKey]struct{}),
		tcproutes:        make(map[client.ObjectKey]struct{}),
		tlsroutes:        make(map[client.ObjectKey]struct{}),
		ratelimits:       make(map[client.ObjectKey]struct{}),
		sessionstickies:  make(map[client.ObjectKey]struct{}),
		loadbalancers:    make(map[client.ObjectKey]struct{}),
		circuitbreakings: make(map[client.ObjectKey]struct{}),
		accesscontrols:   make(map[client.ObjectKey]struct{}),
		healthchecks:     make(map[client.ObjectKey]struct{}),
		faultinjections:  make(map[client.ObjectKey]struct{}),

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
	//case *corev1.Endpoints:
	//	return c.processors[EndpointsTriggerType]
	case *corev1.Service:
		return c.processors[ServicesTriggerType]
	case *mcsv1alpha1.ServiceImport:
		return c.processors[ServiceImportsTriggerType]
	case *discoveryv1.EndpointSlice:
		return c.processors[EndpointSlicesTriggerType]
	case *corev1.Secret:
		return c.processors[SecretsTriggerType]
	case *gwv1beta1.GatewayClass:
		return c.processors[GatewayClassesTriggerType]
	case *gwv1beta1.Gateway:
		return c.processors[GatewaysTriggerType]
	case *gwv1beta1.HTTPRoute:
		return c.processors[HTTPRoutesTriggerType]
	case *gwv1alpha2.GRPCRoute:
		return c.processors[GRPCRoutesTriggerType]
	case *gwv1alpha2.TCPRoute:
		return c.processors[TCPRoutesTriggerType]
	case *gwv1alpha2.TLSRoute:
		return c.processors[TLSRoutesTriggerType]
	case *gwpav1alpha1.RateLimitPolicy:
		return c.processors[RateLimitPoliciesTriggerType]
	case *gwpav1alpha1.SessionStickyPolicy:
		return c.processors[SessionStickyPoliciesTriggerType]
	case *gwpav1alpha1.LoadBalancerPolicy:
		return c.processors[LoadBalancerPoliciesTriggerType]
	case *gwpav1alpha1.CircuitBreakingPolicy:
		return c.processors[CircuitBreakingPoliciesTriggerType]
	case *gwpav1alpha1.AccessControlPolicy:
		return c.processors[AccessControlPoliciesTriggerType]
	case *gwpav1alpha1.HealthCheckPolicy:
		return c.processors[HealthCheckPoliciesTriggerType]
	case *gwpav1alpha1.FaultInjectionPolicy:
		return c.processors[FaultInjectionPoliciesTriggerType]
	}

	return nil
}
