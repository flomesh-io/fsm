package provider

import (
	"strings"

	consul "github.com/hashicorp/consul/api"

	"github.com/flomesh-io/fsm/pkg/connector"
)

type ConsulDiscoveryClient struct {
	consulClient *consul.Client
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
	services, _, err := dc.consulClient.Health().Service(service, tag, passingOnly, opts)
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
		agentService := new(AgentService)
		agentService.fromConsul(svc.Service)
		agentServices = append(agentServices, agentService)
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
	_, err := dc.consulClient.Catalog().Register(reg.toConsul(), nil)
	return err
}

const (
	WildcardNamespace = "*"
	DefaultNamespace  = "default"
)

// EnsureNamespaceExists ensures a Consul namespace with name ns exists. If it doesn't,
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

func GetConsulDiscoveryClient(address string) (*ConsulDiscoveryClient, error) {
	cfg := consul.DefaultConfig()
	cfg.Address = address
	consulClient, err := consul.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	consulDiscoveryClient := new(ConsulDiscoveryClient)
	consulDiscoveryClient.consulClient = consulClient
	return consulDiscoveryClient, nil
}
