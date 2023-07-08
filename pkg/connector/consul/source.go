// Package consul implements a syncer from consul to k8s.
package consul

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/consul/api"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("cloud-consul")
)

// Source is the source for the sync that watches Consul services and
// updates a Sink whenever the set of services to register changes.
type Source struct {
	ConsulClient *api.Client     // Consul API client
	Domain       string          // Consul DNS domain
	Sink         *connector.Sink // Sink is the sink to update with services
	Prefix       string          // Prefix is a prefix to prepend to services
	FilterTag    string          // The tag value for services registered
	PrefixTag    string
	SuffixTag    string
	PassingOnly  bool
}

// Run is the long-running runloop for watching Consul services and
// updating the Sink.
func (s *Source) Run(ctx context.Context) {
	opts := (&api.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   5 * time.Second,
	}).WithContext(ctx)
	for {
		// Get all services with tags.
		var serviceMap map[string][]string
		var meta *api.QueryMeta
		err := backoff.Retry(func() error {
			var err error
			serviceMap, meta, err = s.ConsulClient.Catalog().Services(opts)
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

		// Update our blocking index
		opts.WaitIndex = meta.LastIndex

		// Setup the services
		services := make(map[connector.MicroSvcName]connector.MicroSvcDomainName, len(serviceMap))
		for name, tags := range serviceMap {
			if strings.EqualFold(name, "consul") {
				continue
			}

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
				services[connector.MicroSvcName(s.Prefix+name)] = connector.MicroSvcDomainName(fmt.Sprintf("%s.service.%s", name, s.Domain))
			}
		}
		log.Info().Msgf("received services from Consul, count:%d", len(services))
		s.Sink.SetServices(services)
	}
}

// Aggregate micro services
func (s *Source) Aggregate(svcName connector.MicroSvcName, svcDomainName connector.MicroSvcDomainName) ([]connector.MicroSvcName, []connector.MicroSvcPort, []connector.MicroEndpointAddr) {
	serviceEntries, _, err := s.ConsulClient.Health().Service(string(svcName), s.FilterTag, s.PassingOnly, nil)
	if err != nil {
		log.Err(err).Msgf("can't retrieve consul service, name:%s", string(svcName))
		return nil, nil, nil
	}
	log.Info().Msgf("PassingOnly:%v FilterTag:%v len(serviceEntries):%d", s.PassingOnly, s.FilterTag, len(serviceEntries))
	if len(serviceEntries) == 0 {
		return nil, nil, nil
	}

	svcNames := make([]connector.MicroSvcName, 0)
	svcNames = append(svcNames, svcName)

	svcPorts := make([]connector.MicroSvcPort, 0)
	svcAddrs := make([]connector.MicroEndpointAddr, 0)

	for _, svc := range serviceEntries {
		svcPorts = append(svcPorts, connector.MicroSvcPort{
			Port:        svc.Service.Port,
			AppProtocol: constants.ProtocolHTTP,
		})
		if len(svc.Service.Tags) > 0 {
			svcPrefix := ""
			svcSuffix := ""
			for _, tag := range svc.Service.Tags {
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
						if grpcPort, convErr := strconv.Atoi(segs[1]); convErr == nil {
							svcPorts = append(svcPorts, connector.MicroSvcPort{
								Port:        grpcPort,
								AppProtocol: constants.ProtocolGRPC,
							})
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
				svcNames = append(svcNames, connector.MicroSvcName(extSvcName))
			}
		}
		svcAddrs = append(svcAddrs, connector.MicroEndpointAddr(svc.Service.Address))
	}
	return svcNames, svcPorts, svcAddrs
}
