package provider

import (
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set"
	consul "github.com/hashicorp/consul/api"

	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	CONSUL_METADATA_GRPC_PORT = "gRPC.port="
)

type ConsulDiscoveryClient struct {
	consulClient       *consul.Client
	isInternalServices bool
	clusterId          string
	appendTagSet       mapset.Set
}

func (dc *ConsulDiscoveryClient) IsInternalServices() bool {
	return dc.isInternalServices
}

func (dc *ConsulDiscoveryClient) CatalogServices(q *QueryOptions) (map[string][]string, error) {
	servicesMap, meta, err := dc.consulClient.Catalog().Services(q.toConsul())
	if err != nil {
		return nil, err
	}

	q.WaitIndex = meta.LastIndex

	catalogServices := make(map[string][]string)
	if len(servicesMap) > 0 {
		for svc, svcTags := range servicesMap {
			if strings.EqualFold(svc, "consul") {
				continue
			}
			svcTagArray, exists := catalogServices[svc]
			if !exists {
				svcTagArray = make([]string, 0)
			}
			svcTagArray = append(svcTagArray, svcTags...)
			catalogServices[svc] = svcTagArray
		}
	}
	return catalogServices, nil
}

// CatalogService is used to query catalog entries for a given service
func (dc *ConsulDiscoveryClient) CatalogService(service, tag string, q *QueryOptions) ([]*CatalogService, error) {
	// Only consider services that are tagged from k8s
	var opts *consul.QueryOptions = nil
	if q != nil {
		opts = q.toConsul()
	}
	services, _, err := dc.consulClient.Catalog().Service(service, tag, opts)
	if err != nil {
		return nil, err
	}

	catalogServices := make([]*CatalogService, 0)
	for _, svc := range services {
		if len(svc.ServiceMeta) > 0 {
			if serviceSource, serviceSourceExists := svc.ServiceMeta[connector.ServiceSourceKey]; serviceSourceExists {
				if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
					catalogService := new(CatalogService)
					catalogService.fromConsul(svc)
					catalogServices = append(catalogServices, catalogService)
				}
			}
		}
	}
	return catalogServices, nil
}

// HealthService is used to query catalog entries for a given service
func (dc *ConsulDiscoveryClient) HealthService(service, tag string, q *QueryOptions, passingOnly bool) ([]*AgentService, error) {
	var opts *consul.QueryOptions = nil
	if q != nil {
		opts = q.toConsul()
	}
	services, _, err := dc.consulClient.Health().Service(service, tag, false, opts)
	if err != nil {
		return nil, err
	}

	agentServices := make([]*AgentService, 0)
	for _, svc := range services {
		if len(svc.Service.Meta) > 0 {
			if serviceSource, serviceSourceExists := svc.Service.Meta[connector.ServiceSourceKey]; serviceSourceExists {
				if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
					continue
				}
			}
		}

		if !passingOnly {
			agentService := new(AgentService)
			agentService.fromConsul(svc.Service)
			agentService.ClusterId = dc.clusterId
			agentServices = append(agentServices, agentService)
			continue
		}

		healthPassing := false
		if len(svc.Checks) > 0 {
			for _, chk := range svc.Checks {
				if strings.EqualFold(chk.ServiceID, svc.Service.ID) {
					if strings.EqualFold(chk.Status, consul.HealthPassing) {
						healthPassing = true
					}
					break
				}
			}
		}

		if healthPassing {
			agentService := new(AgentService)
			agentService.fromConsul(svc.Service)
			agentService.ClusterId = dc.clusterId
			agentServices = append(agentServices, agentService)
		}

		checkService := new(AgentService)
		checkService.fromConsul(svc.Service)
		checkService.ClusterId = dc.clusterId
		checkService.Service = fmt.Sprintf("%s-check", svc.Service.Service)
		checkService.HealthCheck = true
		checkService.Tags = nil
		checkService.Meta = nil
		agentServices = append(agentServices, checkService)
	}
	return agentServices, nil
}

func (dc *ConsulDiscoveryClient) NodeServiceList(node string, q *QueryOptions) (*CatalogNodeServiceList, error) {
	var opts *consul.QueryOptions = nil
	if q != nil {
		opts = q.toConsul()
	}
	nodeServices, meta, err := dc.consulClient.Catalog().NodeServiceList(node, opts)
	if err != nil {
		return nil, err
	}

	// Update our blocking index
	q.WaitIndex = meta.LastIndex

	nodeServiceList := new(CatalogNodeServiceList)
	nodeServiceList.fromConsul(nodeServices)
	return nodeServiceList, nil
}

func (dc *ConsulDiscoveryClient) Deregister(dereg *CatalogDeregistration) error {
	_, err := dc.consulClient.Catalog().Deregister(dereg.toConsul(), nil)
	return err
}

func (dc *ConsulDiscoveryClient) Register(reg *CatalogRegistration) error {
	ins := reg.toConsul()
	appendTags := dc.appendTagSet.ToSlice()
	if len(appendTags) > 0 {
		for _, tag := range appendTags {
			ins.Service.Tags = append(ins.Service.Tags, tag.(string))
		}
	}
	_, err := dc.consulClient.Catalog().Register(ins, nil)
	return err
}

const (
	WildcardNamespace = "*"
	DefaultNamespace  = "default"
)

// EnsureNamespaceExists ensures a namespace with name ns exists. If it doesn't,
// it will create it and set crossNSACLPolicy as a policy default.
// Boolean return value indicates if the namespace was created by this call.
func (dc *ConsulDiscoveryClient) EnsureNamespaceExists(ns string, crossNSAClPolicy string) (bool, error) {
	if ns == WildcardNamespace || ns == DefaultNamespace {
		return false, nil
	}
	// Check if the Consul namespace exists.
	namespaceInfo, _, err := dc.consulClient.Namespaces().Read(ns, nil)
	if err != nil {
		return false, err
	}
	if namespaceInfo != nil {
		return false, nil
	}

	// If not, create it.
	var aclConfig consul.NamespaceACLConfig
	if crossNSAClPolicy != "" {
		// Create the ACLs config for the cross-Consul-namespace
		// default policy that needs to be attached
		aclConfig = consul.NamespaceACLConfig{
			PolicyDefaults: []consul.ACLLink{
				{Name: crossNSAClPolicy},
			},
		}
	}

	consulNamespace := consul.Namespace{
		Name:        ns,
		Description: "Auto-generated by consul-k8s",
		ACLs:        &aclConfig,
		Meta:        map[string]string{"external-source": "kubernetes"},
	}

	_, _, err = dc.consulClient.Namespaces().Create(&consulNamespace, nil)
	return true, err
}

func (dc *ConsulDiscoveryClient) MicroServiceProvider() string {
	return connector.ConsulDiscoveryService
}

func GetConsulDiscoveryClient(address string, isInternalServices bool, clusterId string,
	appendTagSet mapset.Set) (*ConsulDiscoveryClient, error) {
	cfg := consul.DefaultConfig()
	cfg.Address = address
	consulClient, err := consul.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	consulDiscoveryClient := new(ConsulDiscoveryClient)
	consulDiscoveryClient.consulClient = consulClient
	consulDiscoveryClient.isInternalServices = isInternalServices
	consulDiscoveryClient.clusterId = clusterId
	consulDiscoveryClient.appendTagSet = appendTagSet
	return consulDiscoveryClient, nil
}
