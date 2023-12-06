package provider

import (
	"fmt"
	"strings"

	"github.com/hudl/fargo"
	"github.com/op/go-logging"

	"github.com/flomesh-io/fsm/pkg/connector"
)

type EurekaDiscoveryClient struct {
	eurekaClient       *fargo.EurekaConnection
	isInternalServices bool
}

func (dc *EurekaDiscoveryClient) IsInternalServices() bool {
	return dc.isInternalServices
}

func (dc *EurekaDiscoveryClient) CatalogServices(q *QueryOptions) (map[string][]string, error) {
	servicesMap, err := dc.eurekaClient.GetApps()
	if err != nil {
		return nil, err
	}

	catalogServices := make(map[string][]string)
	if len(servicesMap) > 0 {
		for svc, svcApp := range servicesMap {
			for _, svcIns := range svcApp.Instances {
				if serviceSource, serviceSourceErr := svcIns.Metadata.GetString(connector.ServiceSourceKey); serviceSourceErr == nil {
					if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
						continue
					}
				}
				svcTagArray, exists := catalogServices[svc]
				if !exists {
					svcTagArray = make([]string, 0)
				}
				metadata := svcIns.Metadata.GetMap()
				for k, v := range metadata {
					svcTagArray = append(svcTagArray, fmt.Sprintf("%s=%v", k, v))
				}
				catalogServices[svc] = svcTagArray
			}
		}
	}
	return catalogServices, nil
}

// CatalogService is used to query catalog entries for a given service
func (dc *EurekaDiscoveryClient) CatalogService(service, tag string, q *QueryOptions) ([]*CatalogService, error) {
	services, err := dc.eurekaClient.GetApp(service)
	if err != nil {
		return nil, err
	}
	catalogServices := make([]*CatalogService, 0)
	for _, ins := range services.Instances {
		if serviceSource, serviceSourceErr := ins.Metadata.GetString(connector.ServiceSourceKey); serviceSourceErr == nil {
			if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
				catalogService := new(CatalogService)
				catalogService.fromEureka(ins)
				catalogServices = append(catalogServices, catalogService)
			}
		}
	}
	return catalogServices, nil
}

// HealthService is used to query catalog entries for a given service
func (dc *EurekaDiscoveryClient) HealthService(service, tag string, q *QueryOptions, passingOnly bool) ([]*AgentService, error) {
	services, err := dc.eurekaClient.GetApp(service)
	if err != nil {
		return nil, err
	}

	agentServices := make([]*AgentService, 0)
	for _, ins := range services.Instances {
		if serviceSource, serviceSourceErr := ins.Metadata.GetString(connector.ServiceSourceKey); serviceSourceErr == nil {
			if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
				continue
			}
		}
		agentService := new(AgentService)
		agentService.fromEureka(ins)
		agentServices = append(agentServices, agentService)
	}
	return agentServices, nil
}

func (dc *EurekaDiscoveryClient) NodeServiceList(node string, q *QueryOptions) (*CatalogNodeServiceList, error) {
	return nil, nil
}

func (dc *EurekaDiscoveryClient) Deregister(dereg *CatalogDeregistration) error {
	return dc.eurekaClient.DeregisterInstance(dereg.toEureka())
}

func (dc *EurekaDiscoveryClient) Register(reg *CatalogRegistration) error {
	return dc.eurekaClient.RegisterInstance(reg.toEureka())
}

// EnsureNamespaceExists ensures a Consul namespace with name ns exists. If it doesn't,
// it will create it and set crossNSACLPolicy as a policy default.
// Boolean return value indicates if the namespace was created by this call.
func (dc *EurekaDiscoveryClient) EnsureNamespaceExists(ns string, crossNSAClPolicy string) (bool, error) {
	return false, nil
}

func (dc *EurekaDiscoveryClient) MicroServiceProvider() string {
	return connector.EurekaDiscoveryService
}

func GetEurekaDiscoveryClient(address string, isInternalServices bool) (*EurekaDiscoveryClient, error) {
	eurekaClient := fargo.NewConn(address)
	eurekaDiscoveryClient := new(EurekaDiscoveryClient)
	eurekaDiscoveryClient.eurekaClient = &eurekaClient
	eurekaDiscoveryClient.isInternalServices = isInternalServices
	logging.SetLevel(logging.WARNING, "fargo")
	return eurekaDiscoveryClient, nil
}
