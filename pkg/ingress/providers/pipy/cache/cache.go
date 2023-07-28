/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package cache

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/flomesh-io/fsm/pkg/configurator"
	repocfg "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/route"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/utils"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"strings"
	"sync"
	"time"
)

type Cache struct {
	kubeClient kubernetes.Interface
	recorder   events.EventRecorder
	cfg        configurator.Configurator
	informers  *fsminformers.InformerCollection

	serviceChanges       *ServiceChangeTracker
	endpointsChanges     *EndpointChangeTracker
	ingressChanges       *IngressChangeTracker
	serviceImportChanges *ServiceImportChangeTracker

	serviceMap               ServiceMap
	endpointsMap             EndpointsMap
	ingressMap               IngressMap
	serviceImportMap         ServiceImportMap
	multiClusterEndpointsMap MultiClusterEndpointsMap

	mu sync.Mutex

	//endpointsSynced      bool
	//servicesSynced       bool
	//ingressesSynced      bool
	//ingressClassesSynced bool
	//serviceImportSynced  bool
	//initialized          int32

	repoClient  *repo.PipyRepoClient
	broadcaster events.EventBroadcaster

	ingressRoutesVersion string
	serviceRoutesVersion string
}

var (
	log = logger.New("fsm-ingress-cache")
)

func NewCache(kubeClient kubernetes.Interface, informers *fsminformers.InformerCollection, cfg configurator.Configurator) *Cache {
	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: kubeClient.EventsV1()})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, "fsm-cluster-connector-local")

	c := &Cache{
		kubeClient:               kubeClient,
		recorder:                 recorder,
		cfg:                      cfg,
		informers:                informers,
		serviceMap:               make(ServiceMap),
		serviceImportMap:         make(ServiceImportMap),
		endpointsMap:             make(EndpointsMap),
		ingressMap:               make(IngressMap),
		multiClusterEndpointsMap: make(MultiClusterEndpointsMap),
		repoClient:               repo.NewRepoClient(cfg.GetRepoServerIPAddr(), uint16(cfg.GetProxyServerPort())),
		broadcaster:              eventBroadcaster,
	}

	c.serviceChanges = NewServiceChangeTracker(enrichServiceInfo, recorder, kubeClient, informers)
	c.serviceImportChanges = NewServiceImportChangeTracker(enrichServiceImportInfo, nil, recorder, informers)
	c.endpointsChanges = NewEndpointChangeTracker(nil, recorder)
	c.ingressChanges = NewIngressChangeTracker(kubeClient, informers, recorder)

	return c
}

//
//func (c *Cache) GetBroadcaster() events.EventBroadcaster {
//	return c.broadcaster
//}
//
//func (c *Cache) GetRecorder() events.EventRecorder {
//	return c.recorder
//}

//func (c *Cache) setInitialized(value bool) {
//	var initialized int32
//	if value {
//		initialized = 1
//	}
//	atomic.StoreInt32(&c.initialized, initialized)
//}
//
//func (c *Cache) isInitialized() bool {
//	return atomic.LoadInt32(&c.initialized) > 0
//}

func (c *Cache) SyncRoutes() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.serviceMap.Update(c.serviceChanges)
	log.Info().Msgf("Service Map: %v", c.serviceMap)

	c.serviceImportMap.Update(c.serviceImportChanges)
	log.Info().Msgf("ServiceImport Map: %v", c.serviceImportMap)

	c.multiClusterEndpointsMap.Update(c.serviceImportChanges)
	log.Info().Msgf("MultiCluster Endpoints Map: %v", c.multiClusterEndpointsMap)

	c.endpointsMap.Update(c.endpointsChanges)
	log.Info().Msgf("Endpoints Map: %v", c.endpointsMap)

	c.ingressMap.Update(c.ingressChanges)
	log.Info().Msgf("Ingress Map: %v", c.ingressMap)

	log.Info().Msgf("Start syncing rules ...")

	mc := c.cfg

	serviceRoutes := c.buildServiceRoutes()
	log.Info().Msgf("Service Routes:\n %v", serviceRoutes)

	exists := c.repoClient.CodebaseExists(utils.GetDefaultServicesPath())
	if !exists {
		c.serviceRoutesVersion = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	if c.serviceRoutesVersion != serviceRoutes.Hash && exists {
		log.Info().Msgf("Service Routes changed, old hash=%q, new hash=%q", c.serviceRoutesVersion, serviceRoutes.Hash)
		batches := serviceBatches(serviceRoutes, mc)
		if batches != nil {
			go func() {
				hash := utils.Hash([]byte(serviceRoutes.Hash))
				if _, err := c.repoClient.Batch(fmt.Sprintf("%d", hash), batches); err != nil {
					log.Error().Msgf("Sync service routes to repo failed: %s", err)
					return
				}

				log.Info().Msgf("Updating service routes version ...")
				c.serviceRoutesVersion = serviceRoutes.Hash
			}()
		}

		// If services changed, try to fully rebuild the ingress map
		c.refreshIngress()
	}

	ingressRoutes := c.buildIngressConfig()
	log.Info().Msgf("Ingress Routes:\n %v", ingressRoutes)
	exists = c.repoClient.CodebaseExists(utils.GetDefaultIngressPath())
	if !exists {
		c.ingressRoutesVersion = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	if c.ingressRoutesVersion != ingressRoutes.Hash && exists {
		log.Info().Msgf("Ingress Routes changed, old hash=%q, new hash=%q", c.ingressRoutesVersion, ingressRoutes.Hash)
		batches := c.ingressBatches(ingressRoutes, mc)
		if batches != nil {
			go func() {
				hash := utils.Hash([]byte(ingressRoutes.Hash))
				if _, err := c.repoClient.Batch(fmt.Sprintf("%d", hash), batches); err != nil {
					log.Error().Msgf("Sync ingress routes to repo failed: %s", err)
					return
				}

				log.Info().Msgf("Updating ingress routes version ...")
				c.ingressRoutesVersion = ingressRoutes.Hash
			}()
		}
	}
}

func (c *Cache) refreshIngress() {
	log.Info().Msgf("Refreshing Ingress Map ...")

	ingresses, err := c.informers.GetListers().K8sIngress.
		Ingresses(corev1.NamespaceAll).
		List(labels.Everything())
	if err != nil {
		log.Error().Msgf("Failed to list all ingresses: %s", err)
	}

	for _, ing := range ingresses {
		if !ingresspipy.IsValidPipyIngress(ing) {
			continue
		}

		c.ingressChanges.Update(nil, ing)
	}

	c.ingressMap.Update(c.ingressChanges)
}

func (c *Cache) buildIngressConfig() repocfg.IngressData {
	ingressConfig := repocfg.IngressData{
		Routes: []repocfg.IngressRouteSpec{},
	}

	for _, route := range c.ingressMap {
		svcName := route.Backend()

		ir := repocfg.IngressRouteSpec{
			RouterSpec: repocfg.RouterSpec{
				Host:    route.Host(),
				Path:    route.Path(),
				Service: svcName.String(),
				Rewrite: route.Rewrite(),
			},
			BalancerSpec: repocfg.BalancerSpec{
				Sticky:   route.SessionSticky(),
				Balancer: route.LBType(),
				Upstream: &repocfg.UpstreamSpec{
					Protocol:  strings.ToUpper(route.Protocol()),
					SSLName:   route.UpstreamSSLName(),
					SSLVerify: route.UpstreamSSLVerify(),
					SSLCert:   route.UpstreamSSLCert(),
					Endpoints: []repocfg.UpstreamEndpoint{},
				},
			},
			TLSSpec: repocfg.TLSSpec{
				IsTLS:          route.IsTLS(), // IsTLS=true, Certificate=nil, will use default cert
				VerifyDepth:    route.VerifyDepth(),
				VerifyClient:   route.VerifyClient(),
				Certificate:    route.Certificate(),
				IsWildcardHost: route.IsWildcardHost(),
				TrustedCA:      route.TrustedCA(),
			},
		}

		for _, e := range c.endpointsMap[svcName] {
			ep, ok := e.(*BaseEndpointInfo)
			if !ok {
				log.Error().Msgf("Failed to cast BaseEndpointInfo, endpoint: %s", e.String())
				continue
			}

			epIP := ep.IP()
			epPort, err := ep.Port()
			// Error parsing this endpoint has been logged. Skip to next endpoint.
			if epIP == "" || err != nil {
				continue
			}
			entry := repocfg.UpstreamEndpoint{
				IP:   epIP,
				Port: epPort,
				//Protocol: protocol,
			}
			ir.Upstream.Endpoints = append(ir.Upstream.Endpoints, entry)
		}

		if len(ir.Upstream.Endpoints) > 0 {
			ingressConfig.Routes = append(ingressConfig.Routes, ir)
		}
	}

	ingressConfig.Hash = utils.SimpleHash(ingressConfig)

	return ingressConfig
}

func (c *Cache) ingressBatches(ingressData repocfg.IngressData, mc configurator.Configurator) []repo.Batch {
	batch := repo.Batch{
		Basepath: utils.GetDefaultIngressPath(),
		Items:    []repo.BatchItem{},
	}

	// Generate router.json
	router := repocfg.RouterConfig{Routes: map[string]repocfg.RouterSpec{}}
	// Generate balancer.json
	balancer := repocfg.BalancerConfig{Services: map[string]repocfg.BalancerSpec{}}
	// Generate certificates.json
	certificates := repocfg.TLSConfig{Certificates: map[string]repocfg.TLSSpec{}}

	trustedCAMap := make(map[string]bool, 0)

	for _, r := range ingressData.Routes {
		// router
		router.Routes[routerKey(r)] = r.RouterSpec

		// balancer
		balancer.Services[r.Service] = r.BalancerSpec

		// certificates
		if r.Host != "" && r.IsTLS {
			_, ok := certificates.Certificates[r.Host]
			if ok {
				continue
			}

			certificates.Certificates[r.Host] = r.TLSSpec
		}

		if r.TrustedCA != nil && r.TrustedCA.CA != "" {
			trustedCAMap[r.TrustedCA.CA] = true
		}

		if r.Certificate != nil && r.Certificate.CA != "" {
			trustedCAMap[r.Certificate.CA] = true
		}
	}

	ingressConfig := repocfg.IngressConfig{
		TrustedCAs:     getTrustedCAs(trustedCAMap),
		TLSConfig:      certificates,
		RouterConfig:   router,
		BalancerConfig: balancer,
	}

	batch.Items = append(batch.Items, ingressBatchItems(ingressConfig)...)
	if len(batch.Items) > 0 {
		return []repo.Batch{batch}
	}

	return nil
}

func getTrustedCAs(caMap map[string]bool) []string {
	trustedCAs := make([]string, 0)

	for ca := range caMap {
		trustedCAs = append(trustedCAs, ca)
	}

	return trustedCAs
}

func (c *Cache) buildServiceRoutes() repocfg.ServiceRoute {
	// Build  rules for each service.
	serviceRoutes := repocfg.ServiceRoute{
		Routes: []repocfg.ServiceRouteEntry{},
	}

	svcNames := mapset.NewSet[ServicePortName]()
	for svcName := range c.serviceMap {
		svcNames.Add(svcName)
	}
	for svcName := range c.serviceImportMap {
		svcNames.Add(svcName)
	}

	for _, svcName := range svcNames.ToSlice() {
		svc, exists := c.serviceMap[svcName]
		if exists {
			svcInfo, ok := svc.(*serviceInfo)
			if ok {
				sr := repocfg.ServiceRouteEntry{
					Name:      svcInfo.svcName.Name,
					Namespace: svcInfo.svcName.Namespace,
					Targets:   make([]repocfg.Target, 0),
					PortName:  svcInfo.portName,
				}

				switch svcInfo.Type {
				case corev1.ServiceTypeClusterIP:
					for _, ep := range c.endpointsMap[svcName] {
						sr.Targets = append(sr.Targets, repocfg.Target{
							Address: ep.String(),
							Tags: map[string]string{
								"Node": ep.NodeName(),
								"Host": ep.HostName(),
							}},
						)
					}
					serviceRoutes.Routes = append(serviceRoutes.Routes, sr)
				case corev1.ServiceTypeExternalName:
					sr.Targets = append(sr.Targets, repocfg.Target{
						Address: svcInfo.Address(),
						Tags:    map[string]string{}},
					)
					serviceRoutes.Routes = append(serviceRoutes.Routes, sr)
				}
			} else {
				log.Error().Msgf("Failed to cast serviceInfo, svcName: %s", svcName.String())
			}
		}

		svcImp, exists := c.serviceImportMap[svcName]
		if exists {
			svcImpInfo, ok := svcImp.(*serviceImportInfo)
			if ok {
				sr := repocfg.ServiceRouteEntry{
					Name:      svcImpInfo.svcName.Name,
					Namespace: svcImpInfo.svcName.Namespace,
					Targets:   make([]repocfg.Target, 0),
					PortName:  svcImpInfo.portName,
				}

				for _, ep := range c.multiClusterEndpointsMap[svcName] {
					sr.Targets = append(sr.Targets, repocfg.Target{
						Address: ep.String(),
						Tags: map[string]string{
							"Cluster": ep.ClusterInfo(),
						}},
					)
				}

				serviceRoutes.Routes = append(serviceRoutes.Routes, sr)
			}
		}
	}
	serviceRoutes.Hash = utils.SimpleHash(serviceRoutes)

	return serviceRoutes
}

func serviceBatches(serviceRoutes repocfg.ServiceRoute, mc configurator.Configurator) []repo.Batch {
	registry := repocfg.ServiceRegistry{Services: repocfg.ServiceRegistryEntry{}}

	for _, route := range serviceRoutes.Routes {
		addrs := addresses(route)
		if len(addrs) > 0 {
			serviceName := servicePortName(route)
			registry.Services[serviceName] = append(registry.Services[serviceName], addrs...)
		}
	}

	batch := repo.Batch{
		Basepath: utils.GetDefaultServicesPath(),
		Items:    []repo.BatchItem{},
	}

	item := repo.BatchItem{
		Path:     "/config",
		Filename: "registry.json",
		Content:  registry,
	}

	batch.Items = append(batch.Items, item)
	if len(batch.Items) > 0 {
		return []repo.Batch{batch}
	}

	return nil
}

func routerKey(r repocfg.IngressRouteSpec) string {
	return fmt.Sprintf("%s%s", r.Host, r.Path)
}

func ingressBatchItems(ingressConfig repocfg.IngressConfig) []repo.BatchItem {
	return []repo.BatchItem{
		{
			Path:     "/config",
			Filename: "ingress.json",
			Content:  ingressConfig,
		},
	}
}

func servicePortName(route repocfg.ServiceRouteEntry) string {
	return fmt.Sprintf("%s/%s%s", route.Namespace, route.Name, fmtPortName(route.PortName))
}

func addresses(route repocfg.ServiceRouteEntry) []string {
	result := make([]string, 0)
	for _, target := range route.Targets {
		result = append(result, target.Address)
	}

	return result
}
