// Package cache implements a cache of all the resources that are relevant to the gateway
package cache

import (
	"fmt"
	"sync"

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
