// Package cache implements a cache of all the resources that are relevant to the gateway
package cache

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"
)

var (
	httpRouteGVK = schema.FromAPIVersionAndKind(gwv1beta1.GroupVersion.String(), "HTTPRoute")
	tlsRouteGVK  = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), "TLSRoute")
	tcpRouteGVK  = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), "TCPRoute")
	grpcRouteGVK = schema.FromAPIVersionAndKind(gwv1alpha2.GroupVersion.String(), "GRPCRoute")
)

// GatewayCache is a cache of all the resources that are relevant to the gateway
type GatewayCache struct {
	repoClient *repo.PipyRepoClient
	informers  *informers.InformerCollection
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator

	processors map[ProcessorType]Processor

	gatewayclass   *gwv1beta1.GatewayClass
	gateways       map[string]client.ObjectKey // ns -> gateway
	services       map[client.ObjectKey]struct{}
	serviceimports map[client.ObjectKey]struct{}
	endpoints      map[client.ObjectKey]struct{}
	endpointslices map[client.ObjectKey]map[client.ObjectKey]struct{} // svc -> endpointslices
	secrets        map[client.ObjectKey]struct{}
	httproutes     map[client.ObjectKey]struct{}
	grpcroutes     map[client.ObjectKey]struct{}
	tcproutes      map[client.ObjectKey]struct{}
	tlsroutes      map[client.ObjectKey]struct{}
	ratelimits     map[client.ObjectKey]struct{}

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
			ServicesProcessorType:       &ServicesProcessor{},
			ServiceImportsProcessorType: &ServiceImportsProcessor{},
			EndpointSlicesProcessorType: &EndpointSlicesProcessor{},
			//EndpointsProcessorType:      &EndpointsProcessor{},
			SecretsProcessorType:           &SecretProcessor{},
			GatewayClassesProcessorType:    &GatewayClassesProcessor{},
			GatewaysProcessorType:          &GatewaysProcessor{},
			HTTPRoutesProcessorType:        &HTTPRoutesProcessor{},
			GRPCRoutesProcessorType:        &GRPCRoutesProcessor{},
			TCPRoutesProcessorType:         &TCPRoutesProcessor{},
			TLSRoutesProcessorType:         &TLSRoutesProcessor{},
			RateLimitPoliciesProcessorType: &RateLimitPoliciesProcessor{},
		},

		gateways:       make(map[string]client.ObjectKey),
		services:       make(map[client.ObjectKey]struct{}),
		serviceimports: make(map[client.ObjectKey]struct{}),
		endpointslices: make(map[client.ObjectKey]map[client.ObjectKey]struct{}),
		//endpoints:      make(map[client.ObjectKey]struct{}),
		secrets:    make(map[client.ObjectKey]struct{}),
		httproutes: make(map[client.ObjectKey]struct{}),
		grpcroutes: make(map[client.ObjectKey]struct{}),
		tcproutes:  make(map[client.ObjectKey]struct{}),
		tlsroutes:  make(map[client.ObjectKey]struct{}),
		ratelimits: make(map[client.ObjectKey]struct{}),

		mutex: new(sync.RWMutex),
	}
}

//func processHTTPRouteBackendFilters(httpRoute *gwv1beta1.HTTPRoute, services map[string]serviceInfo) {
//	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
//	for _, rule := range httpRoute.Spec.Rules {
//		ruleLevelFilters := make(map[gwv1beta1.HTTPRouteFilterType]routecfg.Filter)
//
//		for _, ruleFilter := range rule.Filters {
//			ruleLevelFilters[ruleFilter.Type] = ruleFilter
//		}
//
//		for _, backend := range rule.BackendRefs {
//			if svcPort := backendRefToServicePortName(backend.BackendRef, httpRoute.Namespace); svcPort != nil {
//				svcFilters := copyMap(ruleLevelFilters)
//				for _, svcFilter := range backend.Filters {
//					svcFilters[svcFilter.Type] = svcFilter
//				}
//
//				svcInfo := serviceInfo{
//					svcPortName: *svcPort,
//					filters:     make([]routecfg.Filter, 0),
//				}
//				for _, f := range svcFilters {
//					svcInfo.filters = append(svcInfo.filters, f)
//				}
//				services[svcPort.String()] = svcInfo
//			}
//		}
//	}
//}

//func processGRPCRouteBackendFilters(grpcRoute *gwv1alpha2.GRPCRoute, services map[string]serviceInfo) {
//	// For now, ONLY supports unique filter types, cannot specify one type filter multiple times
//	for _, rule := range grpcRoute.Spec.Rules {
//		ruleLevelFilters := make(map[gwv1alpha2.GRPCRouteFilterType]routecfg.Filter)
//
//		for _, ruleFilter := range rule.Filters {
//			ruleLevelFilters[ruleFilter.Type] = ruleFilter
//		}
//
//		for _, backend := range rule.BackendRefs {
//			if svcPort := backendRefToServicePortName(backend.BackendRef, grpcRoute.Namespace); svcPort != nil {
//				svcFilters := copyMap(ruleLevelFilters)
//				for _, svcFilter := range backend.Filters {
//					svcFilters[svcFilter.Type] = svcFilter
//				}
//
//				svcInfo := serviceInfo{
//					svcPortName: *svcPort,
//					filters:     make([]routecfg.Filter, 0),
//				}
//				for _, f := range svcFilters {
//					svcInfo.filters = append(svcInfo.filters, f)
//				}
//				services[svcPort.String()] = svcInfo
//			}
//		}
//	}
//}
