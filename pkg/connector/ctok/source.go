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
	opts := (&connector.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   s.controller.GetSyncPeriod(),
	}).WithContext(ctx)
	for {
		// Get all services.
		var catalogServices []connector.NamespacedService

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

		// Setup the services
		services := make(map[connector.MicroSvcName]connector.MicroSvcDomainName, len(catalogServices))
		for _, svc := range catalogServices {
			services[connector.MicroSvcName(s.controller.GetPrefix()+svc.Service)] = connector.MicroSvcDomainName(fmt.Sprintf("%s.service.%s", svc.Service, s.domain))
		}
		log.Trace().Msgf("received services from cloud, count:%d", len(services))
		s.syncer.SetServices(services)
		time.Sleep(opts.WaitTime)
	}
}

// Aggregate micro services
func (s *CtoKSource) Aggregate(ctx context.Context, svcName connector.MicroSvcName) map[connector.MicroSvcName]*connector.MicroSvcMeta {
	cloudSvcName, exists := s.syncer.controller.GetC2KContext().RawServices[string(svcName)]
	if !exists {
		return nil
	}

	opts := (&connector.QueryOptions{
		AllowStale: true,
		WaitIndex:  1,
		WaitTime:   5 * time.Second,
	}).WithContext(ctx)

	instanceEntries, err := s.discClient.CatalogInstances(cloudSvcName, opts)
	if err != nil {
		return nil
	}

	if len(instanceEntries) == 0 {
		return nil
	}

	svcMetaMap := make(map[connector.MicroSvcName]*connector.MicroSvcMeta)

	for _, instance := range instanceEntries {
		instance.MicroService.Service = strings.ToLower(instance.MicroService.Service)
		svcNames := []connector.MicroSvcName{connector.MicroSvcName(instance.MicroService.Service)}
		if len(instance.Tags) > 0 {
			svcNames = s.aggregateTag(svcName, instance, svcNames)
		}
		if len(instance.Meta) > 0 {
			svcNames = s.aggregateMetadata(svcName, instance, svcNames)
		}
		for _, serviceName := range svcNames {
			s.aggregateMeta(svcMetaMap, serviceName, instance)
		}
	}
	return svcMetaMap
}

func (s *CtoKSource) aggregateMeta(svcMetaMap map[connector.MicroSvcName]*connector.MicroSvcMeta, serviceName connector.MicroSvcName, instance *connector.AgentService) {
	port := instance.MicroService.EndpointPort()
	protocol := instance.MicroService.Protocol()
	svcMeta, exists := svcMetaMap[serviceName]
	if !exists {
		svcMeta = new(connector.MicroSvcMeta)
		svcMeta.Ports = make(map[connector.MicroServicePort]connector.MicroServiceProtocol)
		svcMeta.Endpoints = make(map[connector.MicroServiceAddress]*connector.MicroEndpointMeta)
		svcMetaMap[serviceName] = svcMeta
	}
	svcMeta.HealthCheck = instance.HealthCheck

	endpointMeta := new(connector.MicroEndpointMeta)
	endpointMeta.Ports = make(map[connector.MicroServicePort]connector.MicroServiceProtocol)
	if *port > 0 {
		svcMeta.Ports[*port] = *protocol
		endpointMeta.Ports[*port] = *protocol
	}
	if *protocol == connector.ProtocolGRPC {
		if len(instance.GRPCInterface) > 0 && len(instance.GRPCMethods) > 0 {
			if svcMeta.GRPCMeta == nil {
				svcMeta.GRPCMeta = new(connector.GRPCMeta)
			}
			svcMeta.GRPCMeta.Interface = instance.GRPCInterface
			if svcMeta.GRPCMeta.Methods == nil {
				svcMeta.GRPCMeta.Methods = make(map[string][]string)
			}
			for _, method := range instance.GRPCMethods {
				eps, exists := svcMeta.GRPCMeta.Methods[method]
				if !exists {
					eps = make([]string, 0)
				}
				eps = append(eps, instance.MicroService.EndpointAddress().Get())
				svcMeta.GRPCMeta.Methods[method] = eps
			}
			endpointMeta.GRPCMeta = instance.Meta
		}
	}
	endpointMeta.Address = *instance.MicroService.EndpointAddress()
	endpointMeta.Native.ClusterId = instance.ClusterId
	endpointMeta.Native.ViaGatewayMode = ctv1.Forward
	if viaGatewayModeIf, ok := instance.Meta[connector.CloudViaGatewayMode]; ok {
		if viaGatewayMode, str := viaGatewayModeIf.(string); str {
			if len(viaGatewayMode) > 0 {
				endpointMeta.Native.ViaGatewayMode = ctv1.WithGatewayMode(viaGatewayMode)
			}
		}
	}
	if httpViaGatewayIf, ok := instance.Meta[connector.CloudHTTPViaGateway]; ok {
		if httpViaGateway, str := httpViaGatewayIf.(string); str {
			if len(httpViaGateway) > 0 {
				endpointMeta.Native.ViaGatewayHTTP = httpViaGateway
			}
		}
	}
	if grpcViaGatewayIf, ok := instance.Meta[connector.CloudGRPCViaGateway]; ok {
		if grpcViaGateway, str := grpcViaGatewayIf.(string); str {
			if len(grpcViaGateway) > 0 {
				endpointMeta.Native.ViaGatewayGRPC = grpcViaGateway
			}
		}
	}
	if clusterSetIf, ok := instance.Meta[connector.ClusterSetKey]; ok {
		if clusterSet, str := clusterSetIf.(string); str {
			if len(clusterSet) > 0 {
				endpointMeta.Native.ClusterSet = clusterSet
				endpointMeta.Native.ClusterId = clusterSet
			}
		}
	}
	if len(endpointMeta.Native.ClusterSet) == 0 || len(endpointMeta.Native.ClusterId) > 0 {
		endpointMeta.Native.ClusterSet = endpointMeta.Native.ClusterId
	}
	svcMeta.Endpoints[*instance.MicroService.EndpointAddress()] = endpointMeta
}

func (s *CtoKSource) aggregateTag(svcName connector.MicroSvcName, svc *connector.AgentService, svcNames []connector.MicroSvcName) []connector.MicroSvcName {
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
		svcNames = append(svcNames, connector.MicroSvcName(extSvcName))
	}
	return svcNames
}

func (s *CtoKSource) aggregateMetadata(svcName connector.MicroSvcName, svc *connector.AgentService, svcNames []connector.MicroSvcName) []connector.MicroSvcName {
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
		svcNames = append(svcNames, connector.MicroSvcName(extSvcName))
	}
	return svcNames
}
