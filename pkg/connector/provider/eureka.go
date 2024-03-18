package provider

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hudl/fargo"
	"github.com/op/go-logging"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

type EurekaDiscoveryClient struct {
	connectController connector.ConnectController
	namingClient      *fargo.EurekaConnection
	lock              sync.Mutex
}

func (dc *EurekaDiscoveryClient) eurekaClient() *fargo.EurekaConnection {
	dc.lock.Lock()
	defer dc.lock.Unlock()

	if dc.namingClient != nil {
		eurekaAddr := dc.connectController.GetHTTPAddr()
		match := false
		for _, addr := range dc.namingClient.ServiceUrls {
			if strings.EqualFold(addr, eurekaAddr) {
				match = true
				break
			}
		}
		if !match {
			dc.namingClient = nil
		}
	}

	if dc.namingClient == nil {
		eurekaConnection := fargo.NewConn(dc.connectController.GetHTTPAddr())
		eurekaConnection.Timeout = time.Duration(60) * time.Second
		eurekaConnection.PollInterval = time.Duration(10) * time.Second
		eurekaConnection.Retries = 2
		eurekaConnection.DNSDiscovery = false
		dc.namingClient = &eurekaConnection
	}

	dc.connectController.WaitLimiter()

	return dc.namingClient
}

func (dc *EurekaDiscoveryClient) IsInternalServices() bool {
	return dc.connectController.AsInternalServices()
}

func (dc *EurekaDiscoveryClient) CatalogInstances(service string, q *connector.QueryOptions) ([]*connector.AgentService, error) {
	services, err := dc.eurekaClient().GetApp(strings.ToUpper(service))
	if err != nil {
		return nil, err
	}
	//servicesMap, err := dc.eurekaClient().GetApps()
	//if err != nil {
	//	return nil, err
	//}
	//services := servicesMap[strings.ToUpper(service)]
	agentServices := make([]*connector.AgentService, 0)
	if services != nil && len(services.Instances) > 0 {
		for _, ins := range services.Instances {
			if clusterSet, clusterSetErr := ins.Metadata.GetString(connector.ClusterSetKey); clusterSetErr == nil {
				if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
					continue
				}
			}
			if filterMetadatas := dc.connectController.GetFilterMetadatas(); len(filterMetadatas) > 0 {
				matched := true
				for _, meta := range filterMetadatas {
					if metaSet, metaErr := ins.Metadata.GetString(meta.Key); metaErr == nil {
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
			agentService.FromEureka(ins)
			agentService.ClusterId = dc.connectController.GetClusterId()
			agentServices = append(agentServices, agentService)
		}
	}
	return agentServices, nil
}

func (dc *EurekaDiscoveryClient) CatalogServices(q *connector.QueryOptions) (map[string][]string, error) {
	servicesMap, err := dc.eurekaClient().GetApps()
	if err != nil {
		return nil, err
	}
	catalogServices := make(map[string][]string)
	if len(servicesMap) > 0 {
		for svc, svcApp := range servicesMap {
			svc := strings.ToLower(svc)
			if strings.Contains(svc, "_") {
				log.Info().Msgf("invalid format, ignore service: %s", svc)
				continue
			}
			if len(svcApp.Instances) == 0 {
				continue
			}
			for _, svcIns := range svcApp.Instances {
				if clusterSet, clusterSetErr := svcIns.Metadata.GetString(connector.ClusterSetKey); clusterSetErr == nil {
					if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
						continue
					}
				}
				if filterMetadatas := dc.connectController.GetFilterMetadatas(); len(filterMetadatas) > 0 {
					matched := true
					for _, meta := range filterMetadatas {
						if metaSet, metaErr := svcIns.Metadata.GetString(meta.Key); metaErr == nil {
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
				svcIns.App = strings.ToLower(svcIns.App)
				svcIns.VipAddress = strings.ToLower(svcIns.VipAddress)
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

func (dc *EurekaDiscoveryClient) RegisteredServices(q *connector.QueryOptions) (*connector.RegisteredServiceList, error) {
	servicesMap, err := dc.eurekaClient().GetApps()
	if err != nil {
		return nil, err
	}
	registeredIServices := make([]*fargo.Instance, 0)
	if len(servicesMap) > 0 {
		for svc, svcApp := range servicesMap {
			svc := strings.ToLower(svc)
			if strings.Contains(svc, "_") {
				log.Warn().Msgf("invalid format, ignore service: %s", svc)
				continue
			}
			instances := svcApp.Instances
			if len(instances) == 0 {
				continue
			}
			for _, instance := range instances {
				instance := instance
				if connectUID, connectUIDErr := instance.Metadata.GetString(connector.ConnectUIDKey); connectUIDErr == nil {
					if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
						registeredIServices = append(registeredIServices, instance)
					}
				}
			}
		}
	}
	registeredServiceList := new(connector.RegisteredServiceList)
	registeredServiceList.FromEureka(registeredIServices)
	return registeredServiceList, nil
}

// RegisteredInstances is used to query catalog entries for a given service
func (dc *EurekaDiscoveryClient) RegisteredInstances(service string, q *connector.QueryOptions) ([]*connector.CatalogService, error) {
	services, err := dc.eurekaClient().GetApp(strings.ToUpper(service))
	if err != nil {
		return nil, err
	}
	//servicesMap, err := dc.eurekaClient().GetApps()
	//if err != nil {
	//	return nil, err
	//}
	//services = servicesMap[strings.ToUpper(service)]
	catalogServices := make([]*connector.CatalogService, 0)
	if services != nil && len(services.Instances) > 0 {
		for _, instance := range services.Instances {
			if connectUID, connectUIDErr := instance.Metadata.GetString(connector.ConnectUIDKey); connectUIDErr == nil {
				if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
					catalogService := new(connector.CatalogService)
					catalogService.FromEureka(instance)
					catalogServices = append(catalogServices, catalogService)
				}
			}
		}
	}
	return catalogServices, nil
}

func (dc *EurekaDiscoveryClient) Deregister(dereg *connector.CatalogDeregistration) error {
	err := dc.eurekaClient().DeregisterInstance(dereg.ToEureka())
	if err != nil {
		if code, present := fargo.HTTPResponseStatusCode(err); present {
			if code == 404 {
				return nil
			}
		}
	}
	return err
}

func (dc *EurekaDiscoveryClient) Register(reg *connector.CatalogRegistration) error {
	ins := reg.ToEureka()
	metadataSet := dc.connectController.GetAppendMetadataSet().ToSlice()
	if len(metadataSet) > 0 {
		rMetadata := ins.Metadata.GetMap()
		for _, item := range metadataSet {
			metadata := item.(ctv1.Metadata)
			rMetadata[metadata.Key] = metadata.Value
		}
	}
	return dc.eurekaClient().RegisterInstance(ins)
}

func (dc *EurekaDiscoveryClient) EnableNamespaces() bool {
	return false
}

// EnsureNamespaceExists ensures a namespace with name ns exists.
func (dc *EurekaDiscoveryClient) EnsureNamespaceExists(ns string) (bool, error) {
	return false, nil
}

// RegisteredNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func (dc *EurekaDiscoveryClient) RegisteredNamespace(kubeNS string) string {
	return ""
}

func (dc *EurekaDiscoveryClient) MicroServiceProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.EurekaDiscoveryService
}

func GetEurekaDiscoveryClient(connectController connector.ConnectController) (*EurekaDiscoveryClient, error) {
	eurekaDiscoveryClient := new(EurekaDiscoveryClient)
	eurekaDiscoveryClient.connectController = connectController
	logging.SetLevel(logging.CRITICAL, "fargo")
	return eurekaDiscoveryClient, nil
}
