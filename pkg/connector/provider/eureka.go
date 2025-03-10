package provider

import (
	"strings"
	"sync"
	"time"

	eureka "github.com/hudl/fargo"
	"github.com/op/go-logging"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/env"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

type heartbeat struct {
	stop     chan struct{}
	dc       *EurekaDiscoveryClient
	instance *eureka.Instance
}

func (h *heartbeat) run() {
	slidingTimer := time.NewTimer(h.dc.connectController.GetEurekaHeartBeatPeriod())
	defer slidingTimer.Stop()
	for {
		select {
		case <-h.stop:
			return
		case <-slidingTimer.C:
			if err := h.dc.eurekaClient().HeartBeatInstance(h.instance); err != nil {
				log.Error().Err(err).Msgf("%s/%s heart beat error", h.instance.App, h.instance.InstanceId)
			}
			slidingTimer.Reset(h.dc.connectController.GetEurekaHeartBeatPeriod())
		}
	}
}
func (h *heartbeat) close() {
	close(h.stop)
}

type EurekaDiscoveryClient struct {
	connectController connector.ConnectController
	namingClient      *eureka.EurekaConnection
	lock              sync.Mutex
	heartbeats        map[string]*heartbeat
	heartbeatLock     sync.Mutex
}

func (dc *EurekaDiscoveryClient) eurekaClient() *eureka.EurekaConnection {
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
		eurekaConnection := eureka.NewConn(dc.connectController.GetHTTPAddr())
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

func (dc *EurekaDiscoveryClient) selectServices() (map[string]*eureka.Application, error) {
	result, err := dc.connectController.CacheCatalogInstances("cache", func() (interface{}, error) {
		return dc.eurekaClient().GetApps()
	})
	if result != nil {
		return result.(map[string]*eureka.Application), err
	}
	return nil, err
}

func (dc *EurekaDiscoveryClient) CatalogInstances(service string, _ *connector.QueryOptions) ([]*connector.AgentService, error) {
	servicesMap, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	services := servicesMap[strings.ToUpper(service)]
	agentServices := make([]*connector.AgentService, 0)
	if services != nil && len(services.Instances) > 0 {
		for _, ins := range services.Instances {
			if clusterSet, clusterSetErr := ins.Metadata.GetString(connector.ClusterSetKey); clusterSetErr == nil {
				if strings.EqualFold(clusterSet, dc.connectController.GetClusterSet()) {
					continue
				}
			}
			if filterMetadatas := dc.connectController.GetC2KFilterMetadatas(); len(filterMetadatas) > 0 {
				matched := true
				for _, meta := range filterMetadatas {
					if metaSet, metaErr := ins.Metadata.GetString(meta.Key); metaErr == nil {
						if strings.EqualFold(metaSet, meta.Value) {
							continue
						} else if len(metaSet) == 0 && len(meta.Value) == 0 {
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
			if excludeMetadatas := dc.connectController.GetC2KExcludeMetadatas(); len(excludeMetadatas) > 0 {
				matched := false
				for _, meta := range excludeMetadatas {
					if metaSet, metaErr := ins.Metadata.GetString(meta.Key); metaErr == nil {
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
					if cidr.Contains(ins.IPAddr) {
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
					if cidr.Contains(ins.IPAddr) {
						exclude = true
						break
					}
				}
				if exclude {
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

func (dc *EurekaDiscoveryClient) CatalogServices(*connector.QueryOptions) ([]ctv1.NamespacedService, error) {
	servicesMap, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	var catalogServices []ctv1.NamespacedService
	if len(servicesMap) > 0 {
		for svc, svcApp := range servicesMap {
			svc := strings.ToLower(svc)
			if errMsgs := validation.IsDNS1035Label(svc); len(errMsgs) > 0 {
				log.Info().Msgf("invalid format, ignore service: %s, errors:%s", svc, strings.Join(errMsgs, "; "))
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
				if filterMetadatas := dc.connectController.GetC2KFilterMetadatas(); len(filterMetadatas) > 0 {
					matched := true
					for _, meta := range filterMetadatas {
						if metaSet, metaErr := svcIns.Metadata.GetString(meta.Key); metaErr == nil {
							if strings.EqualFold(metaSet, meta.Value) {
								continue
							} else if len(metaSet) == 0 && len(meta.Value) == 0 {
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
				if excludeMetadatas := dc.connectController.GetC2KExcludeMetadatas(); len(excludeMetadatas) > 0 {
					matched := false
					for _, meta := range excludeMetadatas {
						if metaSet, metaErr := svcIns.Metadata.GetString(meta.Key); metaErr == nil {
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
						if cidr.Contains(svcIns.IPAddr) {
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
						if cidr.Contains(svcIns.IPAddr) {
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

func (dc *EurekaDiscoveryClient) RegisteredServices(*connector.QueryOptions) ([]ctv1.NamespacedService, error) {
	servicesMap, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	var registeredServices []ctv1.NamespacedService
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
			hasLocalInstances := false
			for _, instance := range instances {
				instance := instance
				if connectUID, connectUIDErr := instance.Metadata.GetString(connector.ConnectUIDKey); connectUIDErr == nil {
					if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
						if dc.connectController.GetEurekaCheckServiceInstanceID() {
							agentService := new(connector.AgentService)
							agentService.FromEureka(instance)
							instanceId := dc.connectController.GetServiceInstanceID(svc,
								agentService.MicroService.EndpointAddress().Get(),
								*agentService.MicroService.EndpointPort(),
								*agentService.MicroService.Protocol())
							if !strings.EqualFold(instance.InstanceId, instanceId) {
								continue
							}
						}
						if !hasLocalInstances {
							hasLocalInstances = true
						}
						if dc.connectController.GetEurekaHeartBeatInstance() {
							dc.heartbeatLock.Lock()
							if _, exists := dc.heartbeats[instance.InstanceId]; !exists {
								h := &heartbeat{
									stop:     make(chan struct{}),
									dc:       dc,
									instance: instance,
								}
								go h.run()
								dc.heartbeats[instance.InstanceId] = h
							}
							dc.heartbeatLock.Unlock()
						}
					}
				}
			}
			if hasLocalInstances {
				registeredServices = append(registeredServices, ctv1.NamespacedService{Service: svc})
			}
		}
	}
	return registeredServices, nil
}

// RegisteredInstances is used to query catalog entries for a given service
func (dc *EurekaDiscoveryClient) RegisteredInstances(service string, _ *connector.QueryOptions) ([]*connector.CatalogService, error) {
	servicesMap, err := dc.selectServices()
	if err != nil {
		return nil, err
	}
	services := servicesMap[strings.ToUpper(service)]
	catalogServices := make([]*connector.CatalogService, 0)
	if services != nil && len(services.Instances) > 0 {
		for _, instance := range services.Instances {
			instance := instance
			if connectUID, connectUIDErr := instance.Metadata.GetString(connector.ConnectUIDKey); connectUIDErr == nil {
				if strings.EqualFold(connectUID, dc.connectController.GetConnectorUID()) {
					if dc.connectController.GetEurekaCheckServiceInstanceID() {
						agentService := new(connector.AgentService)
						agentService.FromEureka(instance)
						instanceId := dc.connectController.GetServiceInstanceID(strings.ToLower(service),
							agentService.MicroService.EndpointAddress().Get(),
							*agentService.MicroService.EndpointPort(),
							*agentService.MicroService.Protocol())
						if !strings.EqualFold(instance.InstanceId, instanceId) {
							continue
						}
					}
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
	ins := dereg.ToEureka()
	return dc.connectController.CacheDeregisterInstance(ins.InstanceId, func() error {
		dc.heartbeatLock.Lock()
		if h, exists := dc.heartbeats[ins.InstanceId]; exists {
			if h != nil {
				h.close()
			}
			delete(dc.heartbeats, ins.InstanceId)
		}
		dc.heartbeatLock.Unlock()
		err := dc.eurekaClient().DeregisterInstance(ins)
		if err != nil {
			if code, present := eureka.HTTPResponseStatusCode(err); present {
				if code == 404 {
					return nil
				}
			}
		}
		return err
	})
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
	cacheIns := *ins
	cacheIns.UniqueID = nil
	return dc.connectController.CacheRegisterInstance(ins.InstanceId, cacheIns, func() error {
		err := dc.eurekaClient().RegisterInstance(ins)
		if err == nil {
			if dc.connectController.GetEurekaHeartBeatInstance() {
				dc.heartbeatLock.Lock()
				if _, exists := dc.heartbeats[ins.InstanceId]; !exists {
					h := &heartbeat{
						stop:     make(chan struct{}),
						dc:       dc,
						instance: ins,
					}
					go h.run()
					dc.heartbeats[ins.InstanceId] = h
				}
				dc.heartbeatLock.Unlock()
			}
		}
		return err
	})
}

func (dc *EurekaDiscoveryClient) EnableNamespaces() bool {
	return false
}

// EnsureNamespaceExists ensures a namespace with name ns exists.
func (dc *EurekaDiscoveryClient) EnsureNamespaceExists(string) (bool, error) {
	return false, nil
}

// RegisteredNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func (dc *EurekaDiscoveryClient) RegisteredNamespace(string) string {
	return ""
}

func (dc *EurekaDiscoveryClient) MicroServiceProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.EurekaDiscoveryService
}

func (dc *EurekaDiscoveryClient) Close() {
	dc.heartbeatLock.Lock()
	defer dc.heartbeatLock.Unlock()
	for _, h := range dc.heartbeats {
		if h != nil {
			h.close()
		}
	}
	dc.heartbeats = make(map[string]*heartbeat)
}

func GetEurekaDiscoveryClient(connectController connector.ConnectController) (*EurekaDiscoveryClient, error) {
	eurekaDiscoveryClient := new(EurekaDiscoveryClient)
	eurekaDiscoveryClient.heartbeats = make(map[string]*heartbeat)
	eurekaDiscoveryClient.connectController = connectController
	level := env.GetString("LOG_LEVEL", "CRITICAL")
	if logLevel, err := logging.LogLevel(strings.ToUpper(level)); err == nil {
		logging.SetLevel(logLevel, "fargo")
	} else {
		logging.SetLevel(logging.CRITICAL, "fargo")
	}
	return eurekaDiscoveryClient, nil
}
