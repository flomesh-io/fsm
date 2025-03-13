// Package ctok implements a syncer from cloud to k8s.
package ctok

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

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
		// Get all services.
		var catalogServices []ctv1.NamespacedService

		if !s.controller.Purge() {
			err := backoff.Retry(func() error {
				var err error
				catalogServices, err = s.discClient.CatalogServices(opts)
				return err
			}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))

			// If the context is ended, then we end
			if ctx.Err() != nil {
				return
			}

			// If there was an error, handle that
			if err != nil {
				log.Warn().Msgf("error querying services, will retry, err:%s", err)
				continue
			}
		}

		var serviceConversions map[string]ctv1.ServiceConversion
		enableConversions := s.controller.EnableC2KConversions()
		if enableConversions {
			serviceConversions = s.controller.GetC2KServiceConversions()
		}

		services := make(map[connector.KubeSvcName]connector.ServiceConversion, len(catalogServices))
		for _, svc := range catalogServices {
			if enableConversions {
				if len(serviceConversions) > 0 {
					if serviceConversion, exists := serviceConversions[fmt.Sprintf("%s/%s", svc.Namespace, svc.Service)]; exists {
						services[connector.KubeSvcName(serviceConversion.ConvertName)] = connector.ServiceConversion{
							Service:      connector.CloudSvcName(svc.Service),
							ExternalName: connector.ExternalName(serviceConversion.ExternalName),
						}
					}
				}
			} else {
				services[connector.KubeSvcName(s.toLegalServiceName(svc.Service))] = connector.ServiceConversion{
					Service: connector.CloudSvcName(svc.Service),
				}
			}
		}

		log.Trace().Msgf("received services from cloud, count:%d", len(services))

		s.syncer.SetServices(services, catalogServices)

		time.Sleep(opts.WaitTime)
	}
}

func (s *CtoKSource) toLegalServiceName(serviceName string) string {
	serviceName = strings.ReplaceAll(serviceName, "_", "-")
	serviceName = strings.ReplaceAll(serviceName, ".", "-")
	serviceName = strings.ReplaceAll(serviceName, " ", "-")
	serviceName = strings.ToLower(serviceName)
	return serviceName
}
