// Package eureka implements a syncer from eureka to k8s.
package eureka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hudl/fargo"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("cloud-eureka")
)

// Source is the source for the sync that watches Consul services and
// updates a Sink whenever the set of services to register changes.
type Source struct {
	EurekaClient *fargo.EurekaConnection // Eureka API client
	Domain       string                  // Eureka DNS domain
	Sink         *connector.Sink         // Sink is the sink to update with services
	Prefix       string                  // Prefix is a prefix to prepend to services
	FilterTag    string                  // The tag value for services registered
	PrefixTag    string
	SuffixTag    string
	PassingOnly  bool
}

// Run is the long-running loop for watching Consul services and
// updating the Sink.
func (s *Source) Run(ctx context.Context) {
	for {
		// Get all services with tags.
		var apps map[string]*fargo.Application
		err := backoff.Retry(func() error {
			var err error
			apps, err = s.EurekaClient.GetApps()
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
		services := make(map[connector.MicroSvcName]connector.MicroSvcDomainName, len(apps))
		for name, app := range apps {
			name = strings.ToLower(name)
			k8s := false
			if len(s.FilterTag) > 0 {
				for _, instance := range app.Instances {
					for metaName := range instance.Metadata.GetMap() {
						if metaName == s.FilterTag {
							k8s = true
							break
						}
					}
				}
			} else {
				k8s = true
			}

			if k8s {
				services[connector.MicroSvcName(s.Prefix+name)] = connector.MicroSvcDomainName(fmt.Sprintf("%s.service.%s", name, s.Domain))
			}
		}
		log.Info().Msgf("received services from Eureka, count:%d", len(services))
		s.Sink.SetServices(services)
		time.Sleep(time.Second)
	}
}

// Aggregate micro services
//
//lint:ignore U1000 ignore unused
func (s *Source) Aggregate(svcName connector.MicroSvcName, svcDomainName connector.MicroSvcDomainName) (map[connector.MicroSvcName]*connector.MicroSvcMeta, string) {
	app, err := s.EurekaClient.GetApp(strings.ToUpper(string(svcName)))
	if err != nil {
		log.Err(err).Msgf("can't retrieve eureka service, name:%s", string(svcName))
		return nil, connector.EurekaDiscoveryService
	}
	serviceEntries := app.Instances
	log.Info().Msgf("PassingOnly:%v FilterTag:%v len(serviceEntries):%d", s.PassingOnly, s.FilterTag, len(serviceEntries))
	if len(serviceEntries) == 0 {
		return nil, connector.EurekaDiscoveryService
	}

	svcMetaMap := make(map[connector.MicroSvcName]*connector.MicroSvcMeta)

	for _, svc := range serviceEntries {
		httpPort := svc.Port
		svcNames := []connector.MicroSvcName{connector.MicroSvcName(svc.VipAddress), connector.MicroSvcName(svc.InstanceId)}
		for _, serviceName := range svcNames {
			svcMeta, exists := svcMetaMap[serviceName]
			if !exists {
				svcMeta = new(connector.MicroSvcMeta)
				svcMeta.Ports = make(map[connector.MicroSvcPort]connector.MicroSvcAppProtocol)
				svcMeta.Addresses = make(map[connector.MicroEndpointAddr]int)
				svcMetaMap[serviceName] = svcMeta
			}
			svcMeta.Ports[connector.MicroSvcPort(httpPort)] = connector.MicroSvcAppProtocol(constants.ProtocolHTTP)
			svcMeta.Addresses[connector.MicroEndpointAddr(svc.IPAddr)] = httpPort
		}
	}
	return svcMetaMap, connector.EurekaDiscoveryService
}
