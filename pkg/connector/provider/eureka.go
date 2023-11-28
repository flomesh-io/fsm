package provider

import (
	"fmt"

	"github.com/hudl/fargo"
	"github.com/op/go-logging"

	"github.com/flomesh-io/fsm/pkg/connector"
)

type EurekaDiscoveryClient struct {
	eurekaClient *fargo.EurekaConnection
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
	catalogServices := make([]*CatalogService, len(services.Instances))
	for idx, ins := range services.Instances {
		catalogServices[idx] = new(CatalogService)
		catalogServices[idx].fromEureka(ins)
	}
	return catalogServices, nil
}

// HealthService is used to query catalog entries for a given service
func (dc *EurekaDiscoveryClient) HealthService(service, tag string, q *QueryOptions, passingOnly bool) ([]*AgentService, error) {
	services, err := dc.eurekaClient.GetApp(service)
	if err != nil {
		return nil, err
	}

	agentServices := make([]*AgentService, len(services.Instances))
	for idx, ins := range services.Instances {
		agentServices[idx] = new(AgentService)
		agentServices[idx].fromEureka(ins)
	}
	return agentServices, nil
}

func (dc *EurekaDiscoveryClient) NodeServiceList(node string, q *QueryOptions) (*CatalogNodeServiceList, error) {
	return nil, nil
}

func (dc *EurekaDiscoveryClient) Deregister(dereg *CatalogDeregistration) error {
	err := dc.eurekaClient.DeregisterInstance(dereg.toEureka())
	return err
}

func (dc *EurekaDiscoveryClient) Register(reg *CatalogRegistration) error {
	err := dc.eurekaClient.RegisterInstance(reg.toEureka())
	return err
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

func GetEurekaDiscoveryClient(eurekaClient *fargo.EurekaConnection) *EurekaDiscoveryClient {
	eurekaDiscoveryClient := new(EurekaDiscoveryClient)
	eurekaDiscoveryClient.eurekaClient = eurekaClient
	logging.SetLevel(logging.WARNING, "fargo")
	return eurekaDiscoveryClient
}
