package provider

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	mapset "github.com/deckarep/golang-set"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

type NacosDiscoveryClient struct {
	connectController connector.ConnectController
	namingClient      naming_client.INamingClient
	serverConfig      constant.ServerConfig
	clientConfig      constant.ClientConfig
	lock              sync.Mutex
}

func (dc *NacosDiscoveryClient) nacosClient() naming_client.INamingClient {
	dc.lock.Lock()
	defer dc.lock.Unlock()

	namespaceId := dc.connectController.GetAuthNacosNamespaceId()
	if len(namespaceId) == 0 {
		namespaceId = constant.DEFAULT_NAMESPACE_ID
	}
	if !strings.EqualFold(dc.clientConfig.NamespaceId, namespaceId) {
		dc.clientConfig.NamespaceId = namespaceId
		dc.namingClient = nil
	}
	if username := dc.connectController.GetAuthNacosUsername(); !strings.EqualFold(dc.clientConfig.Username, username) {
		dc.clientConfig.Username = username
		dc.namingClient = nil
	}
	if password := dc.connectController.GetAuthNacosPassword(); !strings.EqualFold(dc.clientConfig.Password, password) {
		dc.clientConfig.Username = password
		dc.namingClient = nil
	}
	if accessKey := dc.connectController.GetAuthNacosAccessKey(); !strings.EqualFold(dc.clientConfig.AccessKey, accessKey) {
		dc.clientConfig.AccessKey = accessKey
		dc.namingClient = nil
	}
	if secretKey := dc.connectController.GetAuthNacosSecretKey(); !strings.EqualFold(dc.clientConfig.SecretKey, secretKey) {
		dc.clientConfig.SecretKey = secretKey
		dc.namingClient = nil
	}

	address := dc.connectController.GetHTTPAddr()
	segs := strings.Split(address, ":")
	ipAddr := segs[0]
	port, _ := strconv.ParseUint(segs[1], 10, 64)

	if !strings.EqualFold(dc.serverConfig.IpAddr, ipAddr) {
		dc.serverConfig.IpAddr = ipAddr
		dc.namingClient = nil
	}

	if dc.serverConfig.Port != port {
		dc.serverConfig.Port = port
		dc.namingClient = nil
	}

	if dc.namingClient == nil {
		dc.namingClient, _ = clients.CreateNamingClient(map[string]interface{}{
			"serverConfigs": []constant.ServerConfig{dc.serverConfig},
			"clientConfig":  dc.clientConfig,
		})
	}

	dc.connectController.WaitLimiter()
	return dc.namingClient
}

func (dc *NacosDiscoveryClient) getAllServicesInfo() []string {
	services := make([]string, 0)
	serviceSet := mapset.NewSet()
	for _, group := range dc.connectController.GetNacos2KGroupSet() {
		namespaceId := dc.connectController.GetAuthNacosNamespaceId()
		if len(namespaceId) == 0 {
			namespaceId = constant.DEFAULT_NAMESPACE_ID
		}
		if serviceList, err := dc.nacosClient().GetAllServicesInfo(vo.GetAllServiceInfoParam{
			NameSpace: namespaceId,
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
	for _, group := range dc.connectController.GetNacos2KGroupSet() {
		if groupInstances, err := dc.nacosClient().SelectAllInstances(vo.SelectAllInstancesParam{
			ServiceName: svc,
			GroupName:   group,
			Clusters:    dc.connectController.GetNacos2KClusterSet(),
		}); err == nil {
			instances = append(instances, groupInstances...)
		}
	}
	return instances
}

func (dc *NacosDiscoveryClient) selectInstances(svc string, passingOnly bool) []model.Instance {
	instances := make([]model.Instance, 0)
	for _, group := range dc.connectController.GetNacos2KGroupSet() {
		if groupInstances, err := dc.nacosClient().SelectInstances(vo.SelectInstancesParam{
			ServiceName: svc,
			GroupName:   group,
			Clusters:    dc.connectController.GetNacos2KClusterSet(),
			HealthyOnly: passingOnly,
		}); err == nil {
			instances = append(instances, groupInstances...)
		}
	}
	return instances
}

func (dc *NacosDiscoveryClient) IsInternalServices() bool {
	return dc.connectController.AsInternalServices()
}

func (dc *NacosDiscoveryClient) CatalogServices(q *connector.QueryOptions) (map[string][]string, error) {
	serviceList := dc.getAllServicesInfo()
	catalogServices := make(map[string][]string)
	if len(serviceList) > 0 {
		for _, svc := range serviceList {
			instances := dc.selectAllInstances(svc)
			if len(instances) == 0 {
				continue
			}
			for _, svcIns := range instances {
				if clusterSet, clusterSetExist := svcIns.Metadata[connector.ClusterSetKey]; clusterSetExist {
					if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
						continue
					}
				}
				if filterMetadatas := dc.connectController.GetFilterMetadatas(); len(filterMetadatas) > 0 {
					matched := true
					for _, meta := range filterMetadatas {
						if metaSet, metaExist := svcIns.Metadata[meta.Key]; metaExist {
							if strings.EqualFold(metaSet, meta.Value) {
								continue
							}
						}
						matched = false
						break
					}
					if !matched {
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

// RegisteredInstances is used to query catalog entries for a given service
func (dc *NacosDiscoveryClient) RegisteredInstances(service string, q *connector.QueryOptions) ([]*connector.CatalogService, error) {
	instances := dc.selectAllInstances(service)
	catalogServices := make([]*connector.CatalogService, 0)
	if len(instances) > 0 {
		for _, instance := range instances {
			instance := instance
			if connectUID, connectUIDExist := instance.Metadata[connector.ConnectUIDKey]; connectUIDExist {
				if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
					catalogService := new(connector.CatalogService)
					catalogService.FromNacos(&instance)
					catalogServices = append(catalogServices, catalogService)
				}
			}
		}
	}
	return catalogServices, nil
}

func (dc *NacosDiscoveryClient) CatalogInstances(service string, q *connector.QueryOptions) ([]*connector.AgentService, error) {
	instances := dc.selectInstances(service, dc.connectController.GetPassingOnly())
	agentServices := make([]*connector.AgentService, 0)
	if len(instances) > 0 {
		for _, ins := range instances {
			ins := ins
			if clusterSet, clusterSetExist := ins.Metadata[connector.ClusterSetKey]; clusterSetExist {
				if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
					continue
				}
			}
			if filterMetadatas := dc.connectController.GetFilterMetadatas(); len(filterMetadatas) > 0 {
				matched := true
				for _, meta := range filterMetadatas {
					if metaSet, metaExist := ins.Metadata[meta.Key]; metaExist {
						if strings.EqualFold(metaSet, meta.Value) {
							continue
						}
					}
					matched = false
					break
				}
				if !matched {
					continue
				}
			}
			agentService := new(connector.AgentService)
			agentService.FromNacos(&ins)
			agentService.ClusterId = dc.connectController.GetClusterId()
			agentServices = append(agentServices, agentService)
		}
	}
	return agentServices, nil
}

func (dc *NacosDiscoveryClient) RegisteredServices(q *connector.QueryOptions) (*connector.RegisteredServiceList, error) {
	serviceList := dc.getAllServicesInfo()
	registeredServices := make([]*model.Instance, 0)
	if len(serviceList) > 0 {
		for _, svc := range serviceList {
			svc := strings.ToLower(svc)
			if strings.Contains(svc, "_") {
				log.Info().Msgf("invalid format, ignore service: %s", svc)
				continue
			}
			//instances := dc.selectAllInstances(svc)
			instances := dc.selectInstances(svc, true)
			if len(instances) == 0 {
				continue
			}
			for _, instance := range instances {
				instance := instance
				if len(instance.Metadata) > 0 {
					if connectUID, connectUIDExist := instance.Metadata[connector.ConnectUIDKey]; connectUIDExist {
						if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
							registeredServices = append(registeredServices, &instance)
						}
					}
				}
			}
		}
	}
	registeredServiceList := new(connector.RegisteredServiceList)
	registeredServiceList.FromNacos(registeredServices)
	return registeredServiceList, nil
}

func (dc *NacosDiscoveryClient) Deregister(dereg *connector.CatalogDeregistration) error {
	deregIns := *dereg.ToNacos()
	_, err := dc.nacosClient().DeregisterInstance(deregIns)
	return err
}

func (dc *NacosDiscoveryClient) Register(reg *connector.CatalogRegistration) error {
	k2cGroupId := dc.connectController.GetNacosGroupId()
	if len(k2cGroupId) == 0 {
		k2cGroupId = constant.DEFAULT_GROUP
	}

	k2cClusterId := dc.connectController.GetNacosClusterId()
	if len(k2cClusterId) == 0 {
		k2cClusterId = connector.NACOS_DEFAULT_CLUSTER
	}
	ins := reg.ToNacos(k2cClusterId, k2cGroupId, float64(1))
	appendMetadataSet := dc.connectController.GetAppendMetadataSet().ToSlice()
	if len(appendMetadataSet) > 0 {
		rMetadata := ins.Metadata
		for _, item := range appendMetadataSet {
			metadata := item.(ctv1.Metadata)
			rMetadata[metadata.Key] = metadata.Value
		}
	}
	_, err := dc.nacosClient().RegisterInstance(*ins)
	return err
}

func (dc *NacosDiscoveryClient) EnableNamespaces() bool {
	return false
}

// EnsureNamespaceExists ensures a namespace with name ns exists.
func (dc *NacosDiscoveryClient) EnsureNamespaceExists(ns string) (bool, error) {
	return false, nil
}

// RegisteredNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func (dc *NacosDiscoveryClient) RegisteredNamespace(kubeNS string) string {
	return ""
}

func (dc *NacosDiscoveryClient) MicroServiceProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.NacosDiscoveryService
}

func GetNacosDiscoveryClient(connectController connector.ConnectController) (*NacosDiscoveryClient, error) {
	nacosDiscoveryClient := new(NacosDiscoveryClient)
	nacosDiscoveryClient.connectController = connectController
	nacosDiscoveryClient.clientConfig = constant.ClientConfig{
		TimeoutMs:            60000,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
		DisableUseSnapShot:   false,
		LogDir:               "/tmp/nacos/log",
		CacheDir:             "/tmp/nacos/cache",
		LogLevel:             "warn",
	}

	nacosDiscoveryClient.connectController.SetServiceInstanceIDFunc(
		func(name, addr string, httpPort, grpcPort int) string {
			k2cGroupId := connectController.GetNacosGroupId()
			if len(k2cGroupId) == 0 {
				k2cGroupId = constant.DEFAULT_GROUP
			}

			k2cClusterId := connectController.GetNacosClusterId()
			if len(k2cClusterId) == 0 {
				k2cClusterId = connector.NACOS_DEFAULT_CLUSTER
			}

			return fmt.Sprintf("%s#%d#%s#%s@@%s",
				addr, httpPort, k2cClusterId, k2cGroupId, name)
		})
	return nacosDiscoveryClient, nil
}
