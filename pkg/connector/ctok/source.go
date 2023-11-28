// Package c2k implements a syncer from cloud to k8s.
package ctok

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/flomesh-io/fsm/pkg/connector/provider"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// Source is the source for the sync that watches cloud services and
// updates a Sink whenever the set of services to register changes.
type Source struct {
	DiscClient  provider.ServiceDiscoveryClient
	Domain      string // DNS domain
	Sink        *Sink  // Sink is the sink to update with services
	Prefix      string // Prefix is a prefix to prepend to services
	FilterTag   string // The tag value for services registered
	PrefixTag   string
	SuffixTag   string
	PassingOnly bool
}

// Run is the long-running loop for watching cloud services and
// updating the Sink.
func (s *Source) Run(ctx context.Context) {
	opts := (&provider.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   5 * time.Second,
	}).WithContext(ctx)
	for {
		// Get all services with tags.
		var servicesMap map[string][]string
		err := backoff.Retry(func() error {
			var err error
			servicesMap, err = s.DiscClient.CatalogServices(opts)
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
		for service, tags := range servicesMap {
			k8s := false
			if len(s.FilterTag) > 0 {
				for _, t := range tags {
					if t == s.FilterTag {
						k8s = true
						break
					}
				}
			} else {
				k8s = true
			}

			if k8s {
				services[MicroSvcName(s.Prefix+service)] = MicroSvcDomainName(fmt.Sprintf("%s.service.%s", service, s.Domain))
			}
		}
		log.Trace().Msgf("received services from cloud, count:%d", len(services))
		s.Sink.SetServices(services)
		time.Sleep(opts.WaitTime)
	}
}

// Aggregate micro services
//
//lint:ignore U1000 ignore unused
func (s *Source) Aggregate(svcName MicroSvcName, svcDomainName MicroSvcDomainName) (map[MicroSvcName]*MicroSvcMeta, string) {
	serviceEntries, err := s.DiscClient.HealthService(string(svcName), s.FilterTag, nil, s.PassingOnly)
	if err != nil {
		return nil, s.DiscClient.MicroServiceProvider()
	}

	if len(serviceEntries) == 0 {
		return nil, s.DiscClient.MicroServiceProvider()
	}

	svcMetaMap := make(map[MicroSvcName]*MicroSvcMeta)

	for _, svc := range serviceEntries {
		httpPort := svc.Port
		grpcPort := 0
		svcNames := []MicroSvcName{MicroSvcName(svc.Service)}
		if len(svc.Tags) > 0 {
			grpcPort, svcNames = s.aggregateTag(svcName, svc, grpcPort, svcNames)
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
		}
	}
	return svcMetaMap, s.DiscClient.MicroServiceProvider()
}

func (s *Source) aggregateTag(svcName MicroSvcName, svc *provider.AgentService, grpcPort int, svcNames []MicroSvcName) (int, []MicroSvcName) {
	svcPrefix := ""
	svcSuffix := ""
	for _, tag := range svc.Tags {
		if len(s.PrefixTag) > 0 {
			if strings.HasPrefix(tag, fmt.Sprintf("%s=", s.PrefixTag)) {
				if segs := strings.Split(tag, "="); len(segs) == 2 {
					svcPrefix = segs[1]
				}
			}
		}
		if len(s.SuffixTag) > 0 {
			if strings.HasPrefix(tag, fmt.Sprintf("%s=", s.SuffixTag)) {
				if segs := strings.Split(tag, "="); len(segs) == 2 {
					svcSuffix = segs[1]
				}
			}
		}
		if strings.HasPrefix(tag, "gRPC.port=") {
			if segs := strings.Split(tag, "="); len(segs) == 2 {
				if port, convErr := strconv.Atoi(segs[1]); convErr == nil {
					grpcPort = port
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
	return grpcPort, svcNames
}

func (s *Source) aggregateMetadata(svcName MicroSvcName, svc *provider.AgentService, svcNames []MicroSvcName) []MicroSvcName {
	svcPrefix := ""
	svcSuffix := ""
	for tag, v := range svc.Meta {
		if len(s.PrefixTag) > 0 {
			if strings.EqualFold(tag, s.PrefixTag) {
				if tagval, ok := v.(string); ok {
					svcPrefix = tagval
				}
			}
		}
		if len(s.SuffixTag) > 0 {
			if strings.EqualFold(tag, s.SuffixTag) {
				if tagval, ok := v.(string); ok {
					svcSuffix = tagval
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
