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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	networkingv1 "k8s.io/api/networking/v1"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	cctx "github.com/flomesh-io/fsm/pkg/context"

	mapset "github.com/deckarep/golang-set/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"

	"github.com/flomesh-io/fsm/pkg/configurator"
	repocfg "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/route"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/utils"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// Cache is the type used to represent the cache for the ingress controller
type Cache struct {
	recorder events.EventRecorder
	cfg      configurator.Configurator
	client   cache.Cache

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

	repoClient  *repo.PipyRepoClient
	broadcaster events.EventBroadcaster

	ingressRoutesVersion string
	serviceRoutesVersion string
}

var (
	log = logger.New("fsm-ingress-cache")
)

// NewCache creates a new cache for the ingress controller
func NewCache(ctx *cctx.ControllerContext) *Cache {
	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: ctx.KubeClient.EventsV1()})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, "fsm-cluster-connector-local")
	cfg := ctx.Configurator
	repoBaseURL := fmt.Sprintf("%s://%s:%d", "http", cfg.GetRepoServerIPAddr(), cfg.GetProxyServerPort())

	c := &Cache{
		client:                   ctx.Manager.GetCache(),
		recorder:                 recorder,
		cfg:                      ctx.Configurator,
		serviceMap:               make(ServiceMap),
		serviceImportMap:         make(ServiceImportMap),
		endpointsMap:             make(EndpointsMap),
		ingressMap:               make(IngressMap),
		multiClusterEndpointsMap: make(MultiClusterEndpointsMap),
		repoClient:               repo.NewRepoClient(repoBaseURL, ctx.Configurator.GetFSMLogLevel()),
		broadcaster:              eventBroadcaster,
	}

	client := ctx.Manager.GetCache()
	c.serviceChanges = NewServiceChangeTracker(enrichServiceInfo, recorder, client)
	c.serviceImportChanges = NewServiceImportChangeTracker(enrichServiceImportInfo, nil, recorder, client)
	c.endpointsChanges = NewEndpointChangeTracker(nil, recorder)
	c.ingressChanges = NewIngressChangeTracker(client, recorder)

	return c
}

// SyncRoutes syncs the routes to the repo
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
				if err := c.repoClient.BatchFullUpdate(batches); err != nil {
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
				if err := c.repoClient.BatchFullUpdate(batches); err != nil {
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

	ingresses := &networkingv1.IngressList{}
	err := c.client.List(context.Background(), ingresses)
	if err != nil {
		log.Error().Msgf("Failed to list all ingresses: %s", err)
	}

	for _, ing := range ingresses.Items {
		ing := ing
		if !ingresspipy.IsValidPipyIngress(&ing) {
			continue
		}

		c.ingressChanges.Update(nil, &ing)
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
			ep, ok := e.(*baseEndpointInfo)
			if !ok {
				log.Error().Msgf("Failed to cast baseEndpointInfo, endpoint: %s", e.String())
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

func (c *Cache) ingressBatches(ingressData repocfg.IngressData, _ configurator.Configurator) []repo.Batch {
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

func serviceBatches(serviceRoutes repocfg.ServiceRoute, _ configurator.Configurator) []repo.Batch {
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
