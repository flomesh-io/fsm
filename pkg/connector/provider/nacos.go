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
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/env"

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
		dc.clientConfig.Password = password
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

func (dc *NacosDiscoveryClient) selectServices() ([]string, error) {
	var services []string
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
		} else {
			return nil, err
		}
	}
	return services, nil
}

func (dc *NacosDiscoveryClient) selectInstances(svc string) ([]model.Instance, error) {
	result, err := dc.connectController.CacheCatalogInstances(svc, func() (interface{}, error) {
		var instances []model.Instance
		for _, group := range dc.connectController.GetNacos2KGroupSet() {
			if groupInstances, err := dc.nacosClient().SelectInstances(vo.SelectInstancesParam{
				ServiceName: svc,
				GroupName:   group,
				Clusters:    dc.connectController.GetNacos2KClusterSet(),
				HealthyOnly: dc.connectController.GetPassingOnly(),
			}); err == nil {
				instances = append(instances, groupInstances...)
			} else {
				return nil, err
			}
		}
		return instances, nil
	})
	if result != nil {
		return result.([]model.Instance), err
	}
	return nil, err
}

func (dc *NacosDiscoveryClient) IsInternalServices() bool {
	return dc.connectController.AsInternalServices()
}

func (dc *NacosDiscoveryClient) CatalogInstances(service string, _ *connector.QueryOptions) ([]*connector.AgentService, error) {
	instances, err := dc.selectInstances(service)
	if err != nil {
		return nil, err
	}
	agentServices := make([]*connector.AgentService, 0)
	if len(instances) > 0 {
		for _, ins := range instances {
			ins := ins
			if clusterSet, clusterSetExist := ins.Metadata[connector.ClusterSetKey]; clusterSetExist {
				if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
					continue
				}
			}
			if filterMetadatas := dc.connectController.GetC2KFilterMetadatas(); len(filterMetadatas) > 0 {
				matched := true
				for _, meta := range filterMetadatas {
					if metaSet, metaExist := ins.Metadata[meta.Key]; metaExist {
						if strings.EqualFold(metaSet, meta.Value) {
							continue
						}
					} else if len(meta.Value) == 0 {
						continue
					}
					matched = false
					break
				}
				if !matched {
					continue
				}
			}
			if excludeMetadatas := dc.connectController.GetC2KExcludeMetadatas(); len(excludeMetadatas) > 0 {
				matched := false
				for _, meta := range excludeMetadatas {
					if metaSet, metaExist := ins.Metadata[meta.Key]; metaExist {
						if strings.EqualFold(metaSet, meta.Value) {
							matched = true
							break
						}
					}
				}
				if matched {
					continue
				}
			}
			if filterIPRanges := dc.connectController.GetC2KFilterIPRanges(); len(filterIPRanges) > 0 {
				include := false
				for _, cidr := range filterIPRanges {
					if cidr.Contains(ins.Ip) {
						include = true
						break
					}
				}
				if !include {
					continue
				}
			}
			if excludeIPRanges := dc.connectController.GetC2KExcludeIPRanges(); len(excludeIPRanges) > 0 {
				exclude := false
				for _, cidr := range excludeIPRanges {
					if cidr.Contains(ins.Ip) {
						exclude = true
						break
					}
				}
				if exclude {
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

func (dc *NacosDiscoveryClient) CatalogServices(*connector.QueryOptions) ([]connector.MicroService, error) {
	serviceList, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	var catalogServices []connector.MicroService
	if len(serviceList) > 0 {
		for _, svc := range serviceList {
			if errMsgs := validation.IsDNS1035Label(svc); len(errMsgs) > 0 {
				log.Info().Msgf("invalid format, ignore service: %s, errors:%s", svc, strings.Join(errMsgs, "; "))
				continue
			}
			instances, _ := dc.selectInstances(svc)
			if len(instances) == 0 {
				continue
			}
			for _, svcIns := range instances {
				if clusterSet, clusterSetExist := svcIns.Metadata[connector.ClusterSetKey]; clusterSetExist {
					if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
						continue
					}
				}
				if filterMetadatas := dc.connectController.GetC2KFilterMetadatas(); len(filterMetadatas) > 0 {
					matched := true
					for _, meta := range filterMetadatas {
						if metaSet, metaExist := svcIns.Metadata[meta.Key]; metaExist {
							if strings.EqualFold(metaSet, meta.Value) {
								continue
							}
						} else if len(meta.Value) == 0 {
							continue
						}
						matched = false
						break
					}
					if !matched {
						continue
					}
				}
				if excludeMetadatas := dc.connectController.GetC2KExcludeMetadatas(); len(excludeMetadatas) > 0 {
					matched := false
					for _, meta := range excludeMetadatas {
						if metaSet, metaExist := svcIns.Metadata[meta.Key]; metaExist {
							if strings.EqualFold(metaSet, meta.Value) {
								matched = true
								break
							}
						}
					}
					if matched {
						continue
					}
				}
				if filterIPRanges := dc.connectController.GetC2KFilterIPRanges(); len(filterIPRanges) > 0 {
					include := false
					for _, cidr := range filterIPRanges {
						if cidr.Contains(svcIns.Ip) {
							include = true
							break
						}
					}
					if !include {
						continue
					}
				}
				if excludeIPRanges := dc.connectController.GetC2KExcludeIPRanges(); len(excludeIPRanges) > 0 {
					exclude := false
					for _, cidr := range excludeIPRanges {
						if cidr.Contains(svcIns.Ip) {
							exclude = true
							break
						}
					}
					if exclude {
						continue
					}
				}
				catalogServices = append(catalogServices, connector.MicroService{Service: svc})
				break
			}
		}
	}
	return catalogServices, nil
}

// RegisteredInstances is used to query catalog entries for a given service
func (dc *NacosDiscoveryClient) RegisteredInstances(service string, _ *connector.QueryOptions) ([]*connector.CatalogService, error) {
	instances, err := dc.selectInstances(service)
	if err != nil {
		return nil, err
	}
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

func (dc *NacosDiscoveryClient) RegisteredServices(*connector.QueryOptions) ([]connector.MicroService, error) {
	serviceList, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	var registeredServices []connector.MicroService
	if len(serviceList) > 0 {
		for _, svc := range serviceList {
			svc := strings.ToLower(svc)
			if strings.Contains(svc, "_") {
				log.Info().Msgf("invalid format, ignore service: %s", svc)
				continue
			}
			instances, _ := dc.selectInstances(svc)
			if len(instances) == 0 {
				continue
			}
			for _, instance := range instances {
				instance := instance
				if len(instance.Metadata) > 0 {
					if connectUID, connectUIDExist := instance.Metadata[connector.ConnectUIDKey]; connectUIDExist {
						if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
							registeredServices = append(registeredServices, connector.MicroService{Service: svc})
							break
						}
					}
				}
			}
		}
	}
	return registeredServices, nil
}

func (dc *NacosDiscoveryClient) Deregister(dereg *connector.CatalogDeregistration) error {
	ins := dereg.ToNacos()
	if ins == nil {
		return nil
	}
	port, _ := strconv.Atoi(fmt.Sprintf("%d", ins.Port))
	return dc.connectController.CacheDeregisterInstance(dc.getServiceInstanceID(ins.ServiceName, ins.Ip, port, 0), func() error {
		_, err := dc.nacosClient().DeregisterInstance(*ins)
		return err
	})
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
	port, _ := strconv.Atoi(fmt.Sprintf("%d", ins.Port))
	return dc.connectController.CacheRegisterInstance(dc.getServiceInstanceID(ins.ServiceName, ins.Ip, port, 0), ins, func() error {
		_, err := dc.nacosClient().RegisterInstance(*ins)
		return err
	})
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

func (dc *NacosDiscoveryClient) getServiceInstanceID(name, addr string, httpPort, _ int) string {
	k2cGroupId := dc.connectController.GetNacosGroupId()
	if len(k2cGroupId) == 0 {
		k2cGroupId = constant.DEFAULT_GROUP
	}

	k2cClusterId := dc.connectController.GetNacosClusterId()
	if len(k2cClusterId) == 0 {
		k2cClusterId = connector.NACOS_DEFAULT_CLUSTER
	}

	return fmt.Sprintf("%s#%d#%s#%s@@%s",
		addr, httpPort, k2cClusterId, k2cGroupId, name)
}

func (dc *NacosDiscoveryClient) Close() {
}

func GetNacosDiscoveryClient(connectController connector.ConnectController) (*NacosDiscoveryClient, error) {
	level := env.GetString("LOG_LEVEL", "warn")
	nacosDiscoveryClient := new(NacosDiscoveryClient)
	nacosDiscoveryClient.connectController = connectController
	nacosDiscoveryClient.clientConfig = constant.ClientConfig{
		TimeoutMs:            60000,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
		DisableUseSnapShot:   false,
		LogDir:               "/tmp/nacos/log",
		CacheDir:             "/tmp/nacos/cache",
		LogLevel:             level,
	}

	nacosDiscoveryClient.connectController.SetServiceInstanceIDFunc(nacosDiscoveryClient.getServiceInstanceID)
	return nacosDiscoveryClient, nil
}
