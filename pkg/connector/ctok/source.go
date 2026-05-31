// Package ctok implements a syncer from cloud to k8s.
package ctok

import (
	"context"
	"fmt"
	"strings"
	"time"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

// CtoKSource is the source for the sync that watches cloud services and
// updates a CtoKSyncer whenever the set of services to register changes.
type CtoKSource struct {
	controller connector.ConnectController
	syncer     *CtoKSyncer // syncer is the syncer to update with services
	discClient connector.ServiceDiscoveryClient

	domain string // DNS domain
}

func NewCtoKSource(controller connector.ConnectController,
	syncer *CtoKSyncer,
	discClient connector.ServiceDiscoveryClient,
	domain string) *CtoKSource {
	return &CtoKSource{
		controller: controller,
		syncer:     syncer,
		discClient: discClient,
		domain:     domain,
	}
}

// Run is the long-running loop for watching cloud services and
// updating the CtoKSyncer.
func (s *CtoKSource) Run(ctx context.Context) {
	opts := (&connector.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   s.controller.GetSyncPeriod(),
	}).WithContext(ctx)
	for {
		catalogServices := s.pollCatalog(opts)
		services := s.buildServicesMap(catalogServices)
		s.syncer.SetServices(services, catalogServices)
		time.Sleep(opts.WaitTime)
	}
}

// RunEventDriven uses Nacos Subscribe API for real-time service change
// notifications, with lightweight polling to discover new service names.
func (s *CtoKSource) RunEventDriven(ctx context.Context) {
	subClient, ok := s.discClient.(connector.SubscriptionClient)
	if !ok {
		log.Warn().Msg("subscription not supported by discovery client, falling back to polling")
		s.Run(ctx)
		return
	}

	groups := s.controller.GetNacos2KGroupSet()
	clusters := s.controller.GetNacos2KClusterSet()
	discoveryInterval := 30 * time.Second

	subscribed := make(map[string]func())
	defer func() {
		for _, unsub := range subscribed {
			unsub()
		}
	}()

	initialSync := func() []ctv1.NamespacedService {
		svcs := s.pollCatalog(nil)
		s.syncer.SetServices(s.buildServicesMap(svcs), svcs)
		return svcs
	}

	discoverAndSubscribe := func(prevServices []ctv1.NamespacedService) []ctv1.NamespacedService {
		catalogServices := s.pollCatalog(nil)
		currentSet := make(map[string]bool)
		for _, svc := range catalogServices {
			currentSet[svc.Service] = true
			if _, exists := subscribed[svc.Service]; !exists {
				svcName := svc.Service
				unsub, err := subClient.SubscribeToService(svcName, groups, clusters,
					func(instances interface{}, err error) {
						if err != nil {
							log.Warn().Err(err).Msgf("subscribe callback error for %s", svcName)
							return
						}
						services := s.buildServicesMap(catalogServices)
						s.syncer.SetServices(services, catalogServices)
					})
				if err != nil {
					log.Warn().Err(err).Msgf("failed to subscribe to %s", svcName)
					continue
				}
				subscribed[svcName] = unsub
			}
		}
		for key, unsub := range subscribed {
			if !currentSet[key] {
				unsub()
				delete(subscribed, key)
			}
		}
		return catalogServices
	}

	currServices := initialSync()
	discoveryTicker := time.NewTicker(discoveryInterval)
	defer discoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-discoveryTicker.C:
			currServices = discoverAndSubscribe(currServices)
		}
	}
}

func (s *CtoKSource) pollCatalog(opts *connector.QueryOptions) []ctv1.NamespacedService {
	var catalogServices []ctv1.NamespacedService
	if s.controller.Purge() {
		return catalogServices
	}
	var err error
	catalogServices, err = s.discClient.CatalogServices(opts)
	if err != nil {
		log.Warn().Err(err).Msgf("error querying services")
	}
	return catalogServices
}

func (s *CtoKSource) buildServicesMap(catalogServices []ctv1.NamespacedService) map[connector.KubeSvcName]connector.ServiceConversion {
	services := make(map[connector.KubeSvcName]connector.ServiceConversion, len(catalogServices))
	enableConversions := s.controller.EnableC2KConversions()
	var serviceConversions map[string]ctv1.ServiceConversion
	if enableConversions {
		serviceConversions = s.controller.GetC2KServiceConversions()
	}
	for _, svc := range catalogServices {
		if enableConversions {
			if len(serviceConversions) > 0 {
				if serviceConversion, exists := serviceConversions[fmt.Sprintf("%s/%s", svc.Namespace, svc.Service)]; exists {
					services[connector.KubeSvcName(serviceConversion.ConvertName)] = connector.ServiceConversion{
						Service: connector.CloudSvcName(svc.Service),
					}
				}
			}
		} else {
			services[connector.KubeSvcName(s.toLegalServiceName(svc.Service))] = connector.ServiceConversion{
				Service: connector.CloudSvcName(svc.Service),
			}
		}
	}
	return services
}

func (s *CtoKSource) toLegalServiceName(serviceName string) string {
	serviceName = strings.ReplaceAll(serviceName, "_", "-")
	serviceName = strings.ReplaceAll(serviceName, ".", "-")
	serviceName = strings.ReplaceAll(serviceName, " ", "-")
	serviceName = strings.ToLower(serviceName)
	return serviceName
}
