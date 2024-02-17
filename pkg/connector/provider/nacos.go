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
	nacosClient            naming_client.INamingClient
	namespaceId            string
	k2cClusterId           string
	k2cGroupId             string
	clusterSet             []string
	groupSet               []string
	isInternalServices     bool
	clusterId              string
	appendMetadataKeySet   mapset.Set
	appendMetadataValueSet mapset.Set
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
	instances := dc.selectAllInstances(service)
	catalogServices := make([]*CatalogService, 0)
	if len(instances) > 0 {
		for _, ins := range instances {
			ins := ins
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
			ins := ins
			if serviceSource, serviceSourceExist := ins.Metadata[connector.ServiceSourceKey]; serviceSourceExist {
				if strings.EqualFold(serviceSource, connector.ServiceSourceValue) {
					continue
				}
			}
			agentService := new(AgentService)
			agentService.fromNacos(&ins)
			agentService.ClusterId = dc.clusterId
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
	ins := reg.toNacos(dc.k2cClusterId, dc.k2cGroupId, float64(1))
	metaKeys := dc.appendMetadataKeySet.ToSlice()
	metaVals := dc.appendMetadataValueSet.ToSlice()
	if len(metaKeys) > 0 && len(metaVals) > 0 && len(metaKeys) == len(metaVals) {
		rMetadata := ins.Metadata
		for index, key := range metaKeys {
			metaKey := key.(string)
			metaVal := metaVals[index].(string)
			rMetadata[metaKey] = metaVal
		}
	}
	_, err := dc.nacosClient.RegisterInstance(*ins)
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

func GetNacosDiscoveryClient(address, username, password, namespaceId, clusterId, k2cClusterId, k2cGroupId string,
	clusterSet, groupSet []string,
	isInternalServices bool,
	appendMetadataKeySet, appendMetadataValueSet mapset.Set) (*NacosDiscoveryClient, error) {
	nacosDiscoveryClient := new(NacosDiscoveryClient)
	nacosDiscoveryClient.isInternalServices = isInternalServices
	nacosDiscoveryClient.clusterId = clusterId
	nacosDiscoveryClient.appendMetadataKeySet = appendMetadataKeySet
	nacosDiscoveryClient.appendMetadataValueSet = appendMetadataValueSet
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

	if len(k2cClusterId) == 0 {
		k2cClusterId = NACOS_DEFAULT_CLUSTER
	}
	nacosDiscoveryClient.k2cClusterId = k2cClusterId

	if len(k2cGroupId) == 0 {
		k2cGroupId = constant.DEFAULT_GROUP
	}
	nacosDiscoveryClient.k2cGroupId = k2cGroupId

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

	if len(username) > 0 && len(password) > 0 {
		clientConfig.Username = username
		clientConfig.Password = password
	}

	nacosClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		return nil, err
	}
	nacosDiscoveryClient.nacosClient = nacosClient

	connector.ServiceInstanceIDFunc = func(name, addr string, httpPort, grpcPort int) string {
		return fmt.Sprintf("%s#%d#%s#%s@@%s",
			addr, httpPort, k2cClusterId, k2cGroupId, name)
	}
	return nacosDiscoveryClient, nil
}
