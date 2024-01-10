package provider

import (
	"fmt"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	NACOS_METADATA_GRPC_PORT = "gRPC_port"
	NACOS_DEFAULT_CLUSTER    = "DEFAULT"
)

type NacosDiscoveryClient struct {
	nacosClient        naming_client.INamingClient
	namespaceId        string
	clusterId          string
	groupId            string
	clusterSet         []string
	groupSet           []string
	isInternalServices bool
}

func (dc *NacosDiscoveryClient) getAllServicesInfo() []string {
	services := make([]string, 0)
	serviceSet := mapset.NewSet()
	for _, group := range dc.groupSet {
		if serviceList, err := dc.nacosClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
			NameSpace: dc.namespaceId,
			GroupName: group,
			PageNo:    1,
			PageSize:  1024 * 1024,
		}); err == nil {
			if serviceList.Count > 0 {
				for _, svc := range serviceList.Doms {
					if !serviceSet.Contains(svc) {
						serviceSet.Add(svc)
						services = append(services, svc)
					}
				}
			}
		}
	}
	return services
}

func (dc *NacosDiscoveryClient) selectAllInstances(svc string) []model.Instance {
	instances := make([]model.Instance, 0)
	for _, group := range dc.groupSet {
		if groupInstances, err := dc.nacosClient.SelectAllInstances(vo.SelectAllInstancesParam{
			ServiceName: svc,
			GroupName:   group,
			Clusters:    dc.clusterSet,
		}); err == nil {
			instances = append(instances, groupInstances...)
		}
	}
	return instances
}

func (dc *NacosDiscoveryClient) selectInstances(svc string, passingOnly bool) []model.Instance {
	instances := make([]model.Instance, 0)
	for _, group := range dc.groupSet {
		if groupInstances, err := dc.nacosClient.SelectInstances(vo.SelectInstancesParam{
			ServiceName: svc,
			GroupName:   group,
			Clusters:    dc.clusterSet,
			HealthyOnly: passingOnly,
		}); err == nil {
			instances = append(instances, groupInstances...)
		}
	}
	return instances
}

func (dc *NacosDiscoveryClient) IsInternalServices() bool {
	return dc.isInternalServices
}

func (dc *NacosDiscoveryClient) CatalogServices(q *QueryOptions) (map[string][]string, error) {
	serviceList := dc.getAllServicesInfo()
	catalogServices := make(map[string][]string)
	if len(serviceList) > 0 {
		for _, svc := range serviceList {
			instances := dc.selectAllInstances(svc)
			if len(instances) == 0 {
				continue
			}
			for _, svcIns := range instances {
				if serviceSource, serviceSourceExist := svcIns.Metadata[connector.ServiceSourceKey]; serviceSourceExist {
					if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
						continue
					}
				}
				svcTagArray, exists := catalogServices[svc]
				if !exists {
					svcTagArray = make([]string, 0)
				}
				for k, v := range svcIns.Metadata {
					svcTagArray = append(svcTagArray, fmt.Sprintf("%s=%v", k, v))
				}
				catalogServices[svc] = svcTagArray
			}
		}
	}
	return catalogServices, nil
}

// CatalogService is used to query catalog entries for a given service
func (dc *NacosDiscoveryClient) CatalogService(service, tag string, q *QueryOptions) ([]*CatalogService, error) {
	instances := dc.selectInstances(service, false)
	catalogServices := make([]*CatalogService, 0)
	if len(instances) > 0 {
		for _, ins := range instances {
			if serviceSource, serviceSourceExist := ins.Metadata[connector.ServiceSourceKey]; serviceSourceExist {
				if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
					catalogService := new(CatalogService)
					catalogService.fromNacos(&ins)
					catalogServices = append(catalogServices, catalogService)
				}
			}
		}
	}
	return catalogServices, nil
}

// HealthService is used to query catalog entries for a given service
func (dc *NacosDiscoveryClient) HealthService(service, tag string, q *QueryOptions, passingOnly bool) ([]*AgentService, error) {
	instances := dc.selectInstances(service, passingOnly)
	agentServices := make([]*AgentService, 0)
	if len(instances) > 0 {
		for _, ins := range instances {
			if serviceSource, serviceSourceExist := ins.Metadata[connector.ServiceSourceKey]; serviceSourceExist {
				if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
					continue
				}
			}
			agentService := new(AgentService)
			agentService.fromNacos(&ins)
			agentServices = append(agentServices, agentService)
		}
	}
	return agentServices, nil
}

func (dc *NacosDiscoveryClient) NodeServiceList(node string, q *QueryOptions) (*CatalogNodeServiceList, error) {
	return nil, nil
}

func (dc *NacosDiscoveryClient) Deregister(dereg *CatalogDeregistration) error {
	_, err := dc.nacosClient.DeregisterInstance(*dereg.toNacos())
	return err
}

func (dc *NacosDiscoveryClient) Register(reg *CatalogRegistration) error {
	_, err := dc.nacosClient.RegisterInstance(*reg.toNacos(dc.clusterId, dc.groupId, float64(1)))
	return err
}

// EnsureNamespaceExists ensures a namespace with name ns exists. If it doesn't,
// it will create it and set crossNSACLPolicy as a policy default.
// Boolean return value indicates if the namespace was created by this call.
func (dc *NacosDiscoveryClient) EnsureNamespaceExists(ns string, crossNSAClPolicy string) (bool, error) {
	return false, nil
}

func (dc *NacosDiscoveryClient) MicroServiceProvider() string {
	return connector.NacosDiscoveryService
}

func GetNacosDiscoveryClient(address, namespaceId, clusterId, groupId string,
	clusterSet, groupSet []string,
	isInternalServices bool) (*NacosDiscoveryClient, error) {
	nacosDiscoveryClient := new(NacosDiscoveryClient)
	nacosDiscoveryClient.isInternalServices = isInternalServices
	segs := strings.Split(address, ":")
	ipAddr := segs[0]
	port, err := strconv.ParseUint(segs[1], 10, 64)
	if err != nil {
		return nil, err
	}

	if len(namespaceId) == 0 {
		namespaceId = constant.DEFAULT_NAMESPACE_ID
	}
	nacosDiscoveryClient.namespaceId = namespaceId

	if len(clusterId) == 0 {
		clusterId = NACOS_DEFAULT_CLUSTER
	}
	nacosDiscoveryClient.clusterId = clusterId

	if len(groupId) == 0 {
		groupId = constant.DEFAULT_GROUP
	}
	nacosDiscoveryClient.groupId = groupId

	if len(clusterSet) > 0 {
		nacosDiscoveryClient.clusterSet = append(nacosDiscoveryClient.clusterSet, clusterSet...)
	} else {
		nacosDiscoveryClient.clusterSet = append(nacosDiscoveryClient.clusterSet, NACOS_DEFAULT_CLUSTER)
	}

	if len(groupSet) > 0 {
		nacosDiscoveryClient.groupSet = append(nacosDiscoveryClient.groupSet, groupSet...)
	} else {
		nacosDiscoveryClient.groupSet = append(nacosDiscoveryClient.groupSet, constant.DEFAULT_GROUP)
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: ipAddr,
			Port:   port,
		},
	}
	clientConfig := constant.ClientConfig{
		NamespaceId:          namespaceId,
		TimeoutMs:            3000,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
		LogDir:               "/tmp/nacos/log",
		CacheDir:             "/tmp/nacos/cache",
		LogLevel:             "warn",
	}

	nacosClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		return nil, err
	}
	nacosDiscoveryClient.nacosClient = nacosClient
	return nacosDiscoveryClient, nil
}
