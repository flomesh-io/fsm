// Package cache implements a cache of all the resources that are relevant to the gateway
package cache

import (
	"fmt"
	"sync"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

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

	processors map[ProcessorType]Processor

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

		processors: map[ProcessorType]Processor{
			//EndpointsProcessorType:      &EndpointsProcessor{},
			ServicesProcessorType:                &ServicesProcessor{},
			ServiceImportsProcessorType:          &ServiceImportsProcessor{},
			EndpointSlicesProcessorType:          &EndpointSlicesProcessor{},
			SecretsProcessorType:                 &SecretProcessor{},
			GatewayClassesProcessorType:          &GatewayClassesProcessor{},
			GatewaysProcessorType:                &GatewaysProcessor{},
			HTTPRoutesProcessorType:              &HTTPRoutesProcessor{},
			GRPCRoutesProcessorType:              &GRPCRoutesProcessor{},
			TCPRoutesProcessorType:               &TCPRoutesProcessor{},
			TLSRoutesProcessorType:               &TLSRoutesProcessor{},
			RateLimitPoliciesProcessorType:       &RateLimitPoliciesProcessor{},
			SessionStickyPoliciesProcessorType:   &SessionStickyPoliciesProcessor{},
			LoadBalancerPoliciesProcessorType:    &LoadBalancerPoliciesProcessor{},
			CircuitBreakingPoliciesProcessorType: &CircuitBreakingPoliciesProcessor{},
			AccessControlPoliciesProcessorType:   &AccessControlPoliciesProcessor{},
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
	//	return c.processors[EndpointsProcessorType]
	case *corev1.Service:
		return c.processors[ServicesProcessorType]
	case *mcsv1alpha1.ServiceImport:
		return c.processors[ServiceImportsProcessorType]
	case *discoveryv1.EndpointSlice:
		return c.processors[EndpointSlicesProcessorType]
	case *corev1.Secret:
		return c.processors[SecretsProcessorType]
	case *gwv1beta1.GatewayClass:
		return c.processors[GatewayClassesProcessorType]
	case *gwv1beta1.Gateway:
		return c.processors[GatewaysProcessorType]
	case *gwv1beta1.HTTPRoute:
		return c.processors[HTTPRoutesProcessorType]
	case *gwv1alpha2.GRPCRoute:
		return c.processors[GRPCRoutesProcessorType]
	case *gwv1alpha2.TCPRoute:
		return c.processors[TCPRoutesProcessorType]
	case *gwv1alpha2.TLSRoute:
		return c.processors[TLSRoutesProcessorType]
	case *gwpav1alpha1.RateLimitPolicy:
		return c.processors[RateLimitPoliciesProcessorType]
	case *gwpav1alpha1.SessionStickyPolicy:
		return c.processors[SessionStickyPoliciesProcessorType]
	case *gwpav1alpha1.LoadBalancerPolicy:
		return c.processors[LoadBalancerPoliciesProcessorType]
	case *gwpav1alpha1.CircuitBreakingPolicy:
		return c.processors[CircuitBreakingPoliciesProcessorType]
	case *gwpav1alpha1.AccessControlPolicy:
		return c.processors[AccessControlPoliciesProcessorType]
	}

	return nil
}
