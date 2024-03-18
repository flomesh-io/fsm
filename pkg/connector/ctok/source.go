// Package ctok implements a syncer from cloud to k8s.
package ctok

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector-c2k")
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
	// Register a controller for Endpoints
	go (&connector.CacheController{Resource: newEndpointsResource(s.controller, s.syncer)}).Run(ctx.Done())

	opts := (&connector.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   5 * time.Second,
	}).WithContext(ctx)
	for {
		// Get all services with tags.
		var servicesMap map[string][]string
		err := backoff.Retry(func() error {
			var err error
			servicesMap, err = s.discClient.CatalogServices(opts)
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

		// Setup the services
		services := make(map[MicroSvcName]MicroSvcDomainName, len(servicesMap))
		for service := range servicesMap {
			services[MicroSvcName(s.controller.GetPrefix()+service)] = MicroSvcDomainName(fmt.Sprintf("%s.service.%s", service, s.domain))
		}
		log.Trace().Msgf("received services from cloud, count:%d", len(services))
		s.syncer.SetServices(services)
		time.Sleep(opts.WaitTime)
	}
}

// Aggregate micro services
//
//lint:ignore U1000 ignore unused
func (s *CtoKSource) Aggregate(ctx context.Context, svcName MicroSvcName, svcDomainName MicroSvcDomainName) map[MicroSvcName]*MicroSvcMeta {
	if _, exists := s.syncer.controller.GetC2KContext().RawServices[string(svcName)]; !exists {
		return nil
	}

	opts := (&connector.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   5 * time.Second,
	}).WithContext(ctx)

	serviceEntries, err := s.discClient.CatalogInstances(string(svcName), opts)
	if err != nil {
		return nil
	}

	if len(serviceEntries) == 0 {
		return nil
	}

	svcMetaMap := make(map[MicroSvcName]*MicroSvcMeta)

	for _, svc := range serviceEntries {
		httpPort := svc.HTTPPort
		grpcPort := svc.GRPCPort
		svcNames := []MicroSvcName{MicroSvcName(svc.Service)}
		if len(svc.Tags) > 0 {
			svcNames = s.aggregateTag(svcName, svc, svcNames)
		}
		if len(svc.Meta) > 0 {
			svcNames = s.aggregateMetadata(svcName, svc, svcNames)
		}
		for _, serviceName := range svcNames {
			svcMeta, exists := svcMetaMap[serviceName]
			if !exists {
				svcMeta = new(MicroSvcMeta)
				svcMeta.Ports = make(map[MicroSvcPort]MicroSvcAppProtocol)
				svcMeta.Addresses = make(map[MicroEndpointAddr]int)
				svcMetaMap[serviceName] = svcMeta
			}
			svcMeta.Ports[MicroSvcPort(httpPort)] = constants.ProtocolHTTP
			if grpcPort > 0 {
				svcMeta.Ports[MicroSvcPort(grpcPort)] = constants.ProtocolGRPC
			}
			svcMeta.Addresses[MicroEndpointAddr(svc.Address)] = 1
			svcMeta.ClusterId = svc.ClusterId
			svcMeta.HealthCheck = svc.HealthCheck
		}
	}
	return svcMetaMap
}

func (s *CtoKSource) aggregateTag(svcName MicroSvcName, svc *connector.AgentService, svcNames []MicroSvcName) []MicroSvcName {
	svcPrefix := ""
	svcSuffix := ""
	for _, tag := range svc.Tags {
		if len(s.controller.GetPrefixTag()) > 0 {
			if strings.HasPrefix(tag, fmt.Sprintf("%s=", s.controller.GetPrefixTag())) {
				if segs := strings.Split(tag, "="); len(segs) == 2 {
					svcPrefix = segs[1]
				}
			}
		}
		if len(s.controller.GetSuffixTag()) > 0 {
			if strings.HasPrefix(tag, fmt.Sprintf("%s=", s.controller.GetSuffixTag())) {
				if segs := strings.Split(tag, "="); len(segs) == 2 {
					svcSuffix = segs[1]
				}
			}
		}
	}
	if len(svcPrefix) > 0 || len(svcSuffix) > 0 {
		extSvcName := string(svcName)
		if len(svcPrefix) > 0 {
			extSvcName = fmt.Sprintf("%s-%s", svcPrefix, extSvcName)
		}
		if len(svcSuffix) > 0 {
			extSvcName = fmt.Sprintf("%s-%s", extSvcName, svcSuffix)
		}
		svcNames = append(svcNames, MicroSvcName(extSvcName))
	}
	return svcNames
}

func (s *CtoKSource) aggregateMetadata(svcName MicroSvcName, svc *connector.AgentService, svcNames []MicroSvcName) []MicroSvcName {
	svcPrefix := ""
	svcSuffix := ""
	for metaName, metaVal := range svc.Meta {
		if len(s.controller.GetPrefixMetadata()) > 0 {
			if strings.EqualFold(metaName, s.controller.GetPrefixMetadata()) {
				if v, ok := metaVal.(string); ok {
					svcPrefix = v
				}
			}
		}
		if len(s.controller.GetSuffixMetadata()) > 0 {
			if strings.EqualFold(metaName, s.controller.GetSuffixMetadata()) {
				if v, ok := metaVal.(string); ok {
					svcSuffix = v
				}
			}
		}
	}
	if len(svcPrefix) > 0 || len(svcSuffix) > 0 {
		extSvcName := string(svcName)
		if len(svcPrefix) > 0 {
			extSvcName = fmt.Sprintf("%s-%s", svcPrefix, extSvcName)
		}
		if len(svcSuffix) > 0 {
			extSvcName = fmt.Sprintf("%s-%s", extSvcName, svcSuffix)
		}
		svcNames = append(svcNames, MicroSvcName(extSvcName))
	}
	return svcNames
}
