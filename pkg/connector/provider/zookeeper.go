package provider

import (
	"strings"
	"sync"
	"time"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/zookeeper"
	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery"
	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery/dubbo"
	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery/k8s"
	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery/nebula"
)

const (
	NebulaAdaptor = CodecAdaptor(`nebula`)
	DubboAdaptor  = CodecAdaptor(`dubbo`)
	K8sAdaptor    = CodecAdaptor(`k8s`)
)

type CodecAdaptor string

type ZookeeperDiscoveryClient struct {
	connectController connector.ConnectController
	namingClient      *discovery.ServiceDiscovery
	zkAddr            string
	basePath          string
	category          string
	adaptor           string
	adaptorOps        discovery.FuncOps
	lock              sync.Mutex
}

func (dc *ZookeeperDiscoveryClient) zookeeperClient() *discovery.ServiceDiscovery {
	dc.lock.Lock()
	defer dc.lock.Unlock()

	zkAddr := dc.connectController.GetHTTPAddr()
	basePath := dc.connectController.GetZookeeperBasePath()
	category := dc.connectController.GetZookeeperCategory()
	adaptor := dc.connectController.GetZookeeperAdaptor()

	if !strings.EqualFold(dc.zkAddr, zkAddr) ||
		!strings.EqualFold(dc.basePath, basePath) ||
		!strings.EqualFold(dc.category, category) ||
		!strings.EqualFold(dc.adaptor, adaptor) {
		if dc.namingClient != nil {
			dc.namingClient.Close()
		}
		dc.namingClient = nil

		dc.zkAddr = zkAddr
		dc.basePath = basePath
		dc.category = category
		dc.adaptor = adaptor
	}

	if dc.namingClient == nil {
		client, err := zookeeper.NewClient(
			"fsm",
			[]string{dc.zkAddr},
			false,
			zookeeper.WithZkTimeOut(time.Second*15))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect zookeeper")
		}

		switch CodecAdaptor(dc.adaptor) {
		case NebulaAdaptor:
			dc.adaptorOps = nebula.NewAdaptor()
		case DubboAdaptor:
			dc.adaptorOps = dubbo.NewAdaptor()
		case K8sAdaptor:
			dc.adaptorOps = k8s.NewAdaptor()
		default:
			log.Fatal().Msgf("invalid zookeeper adaptor: %s", dc.adaptor)
		}

		dc.namingClient = discovery.NewServiceDiscovery(client, dc.basePath, dc.category, dc.adaptorOps)
	}

	dc.connectController.WaitLimiter()

	return dc.namingClient
}

func (dc *ZookeeperDiscoveryClient) selectServices() ([]string, error) {
	return dc.zookeeperClient().QueryForNames()
}

func (dc *ZookeeperDiscoveryClient) selectInstances(svc string) ([]discovery.ServiceInstance, error) {
	result, err := dc.connectController.CacheCatalogInstances(svc, func() (interface{}, error) {
		return dc.zookeeperClient().QueryForInstances(svc)
	})
	if result != nil {
		return result.([]discovery.ServiceInstance), err
	}
	return nil, err
}

func (dc *ZookeeperDiscoveryClient) IsInternalServices() bool {
	return dc.connectController.AsInternalServices()
}

func (dc *ZookeeperDiscoveryClient) CatalogInstances(service string, _ *connector.QueryOptions) ([]*connector.AgentService, error) {
	instances, err := dc.selectInstances(service)
	if err != nil {
		return nil, err
	}
	agentServices := make([]*connector.AgentService, 0)
	if len(instances) > 0 {
		for _, ins := range instances {
			ins := ins
			if clusterSet, clusterSetExist := ins.GetMetadata(connector.ClusterSetKey); clusterSetExist {
				if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
					continue
				}
			}
			if filterMetadatas := dc.connectController.GetC2KFilterMetadatas(); len(filterMetadatas) > 0 {
				matched := true
				for _, meta := range filterMetadatas {
					if metaSet, metaExist := ins.GetMetadata(meta.Key); metaExist {
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
					if metaSet, metaExist := ins.GetMetadata(meta.Key); metaExist {
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
					if cidr.Contains(ins.InstanceIP()) {
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
					if cidr.Contains(ins.InstanceIP()) {
						exclude = true
						break
					}
				}
				if exclude {
					continue
				}
			}
			agentService := new(connector.AgentService)
			agentService.FromZookeeper(ins)
			agentService.ClusterId = dc.connectController.GetClusterId()
			agentServices = append(agentServices, agentService)
		}
	}
	return agentServices, nil
}

func (dc *ZookeeperDiscoveryClient) CatalogServices(*connector.QueryOptions) ([]ctv1.NamespacedService, error) {
	serviceList, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	var catalogServices []ctv1.NamespacedService
	if len(serviceList) > 0 {
		for _, svc := range serviceList {
			instances, _ := dc.selectInstances(svc)
			if len(instances) == 0 {
				continue
			}
			for _, svcIns := range instances {
				if clusterSet, clusterSetExist := svcIns.GetMetadata(connector.ClusterSetKey); clusterSetExist {
					if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
						continue
					}
				}
				if filterMetadatas := dc.connectController.GetC2KFilterMetadatas(); len(filterMetadatas) > 0 {
					matched := true
					for _, meta := range filterMetadatas {
						if metaSet, metaExist := svcIns.GetMetadata(meta.Key); metaExist {
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
						if metaSet, metaExist := svcIns.GetMetadata(meta.Key); metaExist {
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
						if cidr.Contains(svcIns.InstanceIP()) {
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
						if cidr.Contains(svcIns.InstanceIP()) {
							exclude = true
							break
						}
					}
					if exclude {
						continue
					}
				}
				catalogServices = append(catalogServices, ctv1.NamespacedService{Service: svc})
				break
			}
		}
	}
	return catalogServices, nil
}

// RegisteredInstances is used to query catalog entries for a given service
func (dc *ZookeeperDiscoveryClient) RegisteredInstances(service string, _ *connector.QueryOptions) ([]*connector.CatalogService, error) {
	instances, err := dc.selectInstances(service)
	if err != nil {
		return nil, err
	}
	catalogServices := make([]*connector.CatalogService, 0)
	if len(instances) > 0 {
		for _, instance := range instances {
			instance := instance
			if connectUID, connectUIDExist := instance.GetMetadata(connector.ConnectUIDKey); connectUIDExist {
				if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
					catalogService := new(connector.CatalogService)
					catalogService.FromZookeeper(instance)
					catalogService.ServiceID = dc.connectController.GetServiceInstanceID(strings.ToLower(service), instance.InstanceIP(), connector.MicroServicePort(instance.InstancePort()), connector.ProtocolGRPC)
					catalogServices = append(catalogServices, catalogService)
				}
			}
		}
	}
	return catalogServices, nil
}

func (dc *ZookeeperDiscoveryClient) RegisteredServices(*connector.QueryOptions) ([]ctv1.NamespacedService, error) {
	serviceList, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	var registeredServices []ctv1.NamespacedService
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
				if connectUID, connectUIDExist := instance.GetMetadata(connector.ConnectUIDKey); connectUIDExist {
					if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
						registeredServices = append(registeredServices, ctv1.NamespacedService{Service: svc})
						break
					}
				}
			}
		}
	}
	return registeredServices, nil
}

func (dc *ZookeeperDiscoveryClient) Deregister(dereg *connector.CatalogDeregistration) error {
	ins := dereg.ToZookeeper(dc.adaptorOps)
	if ins == nil {
		return nil
	}
	return dc.connectController.CacheDeregisterInstance(dereg.ServiceID, func() error {
		return dc.zookeeperClient().UnregisterService(ins)
	})
}

func (dc *ZookeeperDiscoveryClient) Register(reg *connector.CatalogRegistration) error {
	ins, err := reg.ToZookeeper(dc.adaptorOps)
	if err != nil {
		return err
	}
	return dc.connectController.CacheRegisterInstance(reg.Service.ID, ins, func() error {
		return dc.zookeeperClient().RegisterService(ins)
	})
}

func (dc *ZookeeperDiscoveryClient) EnableNamespaces() bool {
	return false
}

// EnsureNamespaceExists ensures a namespace with name ns exists.
func (dc *ZookeeperDiscoveryClient) EnsureNamespaceExists(ns string) (bool, error) {
	return false, nil
}

// RegisteredNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func (dc *ZookeeperDiscoveryClient) RegisteredNamespace(kubeNS string) string {
	return ""
}

func (dc *ZookeeperDiscoveryClient) MicroServiceProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.ZookeeperDiscoveryService
}

func (dc *ZookeeperDiscoveryClient) Close() {
}

func GetZookeeperDiscoveryClient(connectController connector.ConnectController) (*ZookeeperDiscoveryClient, error) {
	zookeeperDiscoveryClient := new(ZookeeperDiscoveryClient)
	zookeeperDiscoveryClient.connectController = connectController
	zookeeperDiscoveryClient.adaptor = connectController.GetZookeeperAdaptor()
	switch CodecAdaptor(zookeeperDiscoveryClient.adaptor) {
	case NebulaAdaptor:
		zookeeperDiscoveryClient.adaptorOps = nebula.NewAdaptor()
	case DubboAdaptor:
		zookeeperDiscoveryClient.adaptorOps = dubbo.NewAdaptor()
	case K8sAdaptor:
		zookeeperDiscoveryClient.adaptorOps = k8s.NewAdaptor()
	default:
		log.Fatal().Msgf("invalid zookeeper adaptor: %s", zookeeperDiscoveryClient.adaptor)
	}
	return zookeeperDiscoveryClient, nil
}
