package provider

import (
	"fmt"
	"strings"
	"sync"

	consul "github.com/hashicorp/consul/api"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	consulServiceName = "consul"
)

type ConsulDiscoveryClient struct {
	connectController connector.ConnectController
	lock              sync.Mutex
	clientConfig      *consul.Config
	namingClient      *consul.Client
}

func (dc *ConsulDiscoveryClient) consulClient() *consul.Client {
	dc.lock.Lock()
	defer dc.lock.Unlock()

	if httpAddr := dc.connectController.GetHTTPAddr(); !strings.EqualFold(dc.clientConfig.Address, httpAddr) {
		dc.clientConfig.Address = httpAddr
		dc.namingClient = nil
	}

	username := dc.connectController.GetAuthConsulUsername()
	password := dc.connectController.GetAuthConsulPassword()
	httpAuthUsername := ""
	httpAuthPassword := ""
	if dc.clientConfig != nil && dc.clientConfig.HttpAuth != nil {
		httpAuthUsername = dc.clientConfig.HttpAuth.Username
		httpAuthPassword = dc.clientConfig.HttpAuth.Password
	}
	if !strings.EqualFold(username, httpAuthUsername) || !strings.EqualFold(password, httpAuthPassword) {
		dc.namingClient = nil
		if len(username) > 0 || len(password) > 0 {
			if dc.clientConfig.HttpAuth == nil {
				dc.clientConfig.HttpAuth = new(consul.HttpBasicAuth)
			}
			dc.clientConfig.HttpAuth.Username = username
			dc.clientConfig.HttpAuth.Password = password
		} else {
			dc.clientConfig.HttpAuth = nil
		}
	}

	if dc.namingClient == nil {
		dc.namingClient, _ = consul.NewClient(dc.clientConfig)
	}

	dc.connectController.WaitLimiter()
	return dc.namingClient
}

func (dc *ConsulDiscoveryClient) IsInternalServices() bool {
	return dc.connectController.AsInternalServices()
}

func (dc *ConsulDiscoveryClient) CatalogServices(q *connector.QueryOptions) ([]connector.MicroService, error) {
	opts := q.ToConsul()
	filters := []string{fmt.Sprintf("Service.Meta.%s != `%s`",
		connector.ClusterSetKey,
		dc.connectController.GetClusterSet())}
	if filterMetadatas := dc.connectController.GetFilterMetadatas(); len(filterMetadatas) > 0 {
		for _, meta := range filterMetadatas {
			filters = append(filters, fmt.Sprintf("Service.Meta.%s == `%s`", meta.Key, meta.Value))
		}
	}
	servicesMap, meta, err := dc.consulClient().Catalog().Services(opts)
	if err != nil {
		return nil, err
	}
	q.WaitIndex = meta.LastIndex

	var catalogServices []connector.MicroService
	if len(servicesMap) > 0 {
		for svc := range servicesMap {
			if strings.EqualFold(svc, consulServiceName) {
				continue
			}
			catalogServices = append(catalogServices, connector.MicroService{Service: svc})
		}
	}
	return catalogServices, nil
}

func (dc *ConsulDiscoveryClient) CatalogInstances(service string, q *connector.QueryOptions) ([]*connector.AgentService, error) {
	opts := q.ToConsul()
	filters := []string{fmt.Sprintf("Service.Meta.%s != `%s`",
		connector.ClusterSetKey,
		dc.connectController.GetClusterSet())}
	if filterMetadatas := dc.connectController.GetFilterMetadatas(); len(filterMetadatas) > 0 {
		for _, meta := range filterMetadatas {
			filters = append(filters, fmt.Sprintf("Service.Meta.%s == `%s`", meta.Key, meta.Value))
		}
	}
	opts.Filter = strings.Join(filters, " and ")
	services, _, err := dc.consulClient().Health().Service(service, dc.connectController.GetFilterTag(), false, opts)
	if err != nil {
		return nil, err
	}

	agentServices := make([]*connector.AgentService, 0)
	for _, svc := range services {
		if !dc.connectController.GetPassingOnly() {
			agentService := new(connector.AgentService)
			agentService.FromConsul(svc.Service)
			agentService.ClusterId = dc.connectController.GetClusterId()
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
			agentService := new(connector.AgentService)
			agentService.FromConsul(svc.Service)
			agentService.ClusterId = dc.connectController.GetClusterId()
			agentServices = append(agentServices, agentService)
		}

		checkService := new(connector.AgentService)
		checkService.FromConsul(svc.Service)
		checkService.ClusterId = dc.connectController.GetClusterId()
		checkService.Service = fmt.Sprintf("%s-check", svc.Service.Service)
		checkService.HealthCheck = true
		checkService.Tags = nil
		checkService.Meta = nil
		agentServices = append(agentServices, checkService)
	}
	return agentServices, nil
}

func (dc *ConsulDiscoveryClient) RegisteredServices(q *connector.QueryOptions) ([]connector.MicroService, error) {
	var registeredServices []connector.MicroService
	var opts = q.ToConsul()
	opts.Filter = fmt.Sprintf("ServiceMeta.%s == `%s`",
		connector.ConnectUIDKey,
		dc.connectController.GetConnectorUID())
	servicesMap, meta, err := dc.consulClient().Catalog().Services(opts)
	if err == nil {
		q.WaitIndex = meta.LastIndex
		if len(servicesMap) > 0 {
			for svc := range servicesMap {
				if strings.EqualFold(svc, consulServiceName) {
					continue
				}
				registeredServices = append(registeredServices, connector.MicroService{Service: svc})
			}
		}
	}
	return registeredServices, err
}

// RegisteredInstances is used to query catalog entries for a given service
func (dc *ConsulDiscoveryClient) RegisteredInstances(service string, q *connector.QueryOptions) ([]*connector.CatalogService, error) {
	opts := q.ToConsul()
	opts.Filter = fmt.Sprintf("ServiceMeta.%s == `%s`",
		connector.ConnectUIDKey,
		dc.connectController.GetConnectorUID())
	services, _, err := dc.consulClient().Catalog().Service(service, "", opts)
	if err != nil {
		return nil, err
	}

	catalogServices := make([]*connector.CatalogService, 0)
	for _, svc := range services {
		catalogService := new(connector.CatalogService)
		catalogService.FromConsul(svc)
		catalogServices = append(catalogServices, catalogService)
	}
	return catalogServices, nil
}

func (dc *ConsulDiscoveryClient) Deregister(dereg *connector.CatalogDeregistration) error {
	ins := dereg.ToConsul()
	return dc.connectController.CacheDeregisterInstance(ins.ServiceID, func() error {
		_, err := dc.consulClient().Catalog().Deregister(ins, nil)
		return err
	})
}

func (dc *ConsulDiscoveryClient) Register(reg *connector.CatalogRegistration) error {
	reg.Node = dc.connectController.GetConsulNodeName()
	if len(reg.Node) == 0 {
		reg.Node = dc.connectController.GetClusterSet()
	}
	reg.Address = "127.0.0.1"
	ins := reg.ToConsul()

	ins.Service.Tags = append(ins.Service.Tags, dc.GetClusterTag())
	ins.Service.Tags = append(ins.Service.Tags, dc.GetConnectorUidTag())

	appendTagSet := dc.connectController.GetAppendTagSet().ToSlice()
	if len(appendTagSet) > 0 {
		for _, tag := range appendTagSet {
			ins.Service.Tags = append(ins.Service.Tags, tag.(string))
		}
	}
	appendMetadataSet := dc.connectController.GetAppendMetadataSet().ToSlice()
	if len(appendMetadataSet) > 0 {
		rMetadata := ins.Service.Meta
		for _, item := range appendMetadataSet {
			metadata := item.(ctv1.Metadata)
			rMetadata[metadata.Key] = metadata.Value
		}
	}
	ins.Checks = consul.HealthChecks{
		&consul.HealthCheck{
			Node:        ins.Node,
			CheckID:     fmt.Sprintf("service:%s", ins.Service.ID),
			Name:        fmt.Sprintf("%s-liveness", ins.Service.Service),
			Status:      connector.HealthPassing,
			Notes:       fmt.Sprintf("%s is alive and well.", ins.Service.Service),
			ServiceID:   ins.Service.ID,
			ServiceName: ins.Service.Service,
		},
	}

	return dc.connectController.CacheRegisterInstance(ins.Service.ID, ins, func() error {
		_, err := dc.consulClient().Catalog().Register(ins, nil)
		return err
	})
}

const (
	WildcardNamespace = "*"
	DefaultNamespace  = "default"
)

func (dc *ConsulDiscoveryClient) EnableNamespaces() bool {
	return dc.connectController.GetConsulEnableNamespaces()
}

// EnsureNamespaceExists ensures a namespace with name ns exists. If it doesn't,
// it will create it and set crossNSACLPolicy as a policy default.
// Boolean return value indicates if the namespace was created by this call.
func (dc *ConsulDiscoveryClient) EnsureNamespaceExists(ns string) (bool, error) {
	if ns == WildcardNamespace || ns == DefaultNamespace {
		return false, nil
	}
	// Check if the Consul namespace exists.
	namespaceInfo, _, err := dc.consulClient().Namespaces().Read(ns, nil)
	if err != nil {
		return false, err
	}
	if namespaceInfo != nil {
		return false, nil
	}

	// If not, create it.
	var aclConfig consul.NamespaceACLConfig
	if len(dc.connectController.GetConsulCrossNamespaceACLPolicy()) > 0 {
		// Create the ACLs config for the cross-Consul-namespace
		// default policy that needs to be attached
		aclConfig = consul.NamespaceACLConfig{
			PolicyDefaults: []consul.ACLLink{
				{Name: dc.connectController.GetConsulCrossNamespaceACLPolicy()},
			},
		}
	}

	consulNamespace := consul.Namespace{
		Name:        ns,
		Description: "Auto-generated by flomesh",
		ACLs:        &aclConfig,
		Meta:        map[string]string{"external-source": "kubernetes(flomesh)"},
	}

	_, _, err = dc.consulClient().Namespaces().Create(&consulNamespace, nil)
	return true, err
}

// RegisteredNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func (dc *ConsulDiscoveryClient) RegisteredNamespace(kubeNS string) string {
	if !dc.connectController.GetConsulEnableNamespaces() {
		return ""
	}

	// Mirroring takes precedence.
	if dc.connectController.GetConsulEnableK8SNSMirroring() {
		return fmt.Sprintf("%s%s", dc.connectController.GetConsulK8SNSMirroringPrefix(), kubeNS)
	}

	return dc.connectController.GetConsulDestinationNamespace()
}

func (dc *ConsulDiscoveryClient) MicroServiceProvider() ctv1.DiscoveryServiceProvider {
	return ctv1.ConsulDiscoveryService
}

func (dc *ConsulDiscoveryClient) GetClusterTag() string {
	return fmt.Sprintf("flomesh_cluster_id=%s", dc.connectController.GetClusterSet())
}

func (dc *ConsulDiscoveryClient) GetConnectorUidTag() string {
	return fmt.Sprintf("flomesh_connector_uid=%s", dc.connectController.GetConnectorUID())
}

func GetConsulDiscoveryClient(connectController connector.ConnectController) (*ConsulDiscoveryClient, error) {
	consulDiscoveryClient := new(ConsulDiscoveryClient)
	consulDiscoveryClient.connectController = connectController
	consulDiscoveryClient.clientConfig = consul.DefaultConfig()

	connector.ClusterSetKey = "fsm_connector_service_cluster_set"
	connector.ConnectUIDKey = "fsm_connector_service_connector_uid"
	connector.CloudK8SNS = "fsm_connector_service_k8s_ns"
	connector.CloudK8SRefKind = "fsm_connector_service_k8s_ref_kind"
	connector.CloudK8SRefValue = "fsm_connector_service_k8s_ref_name"
	connector.CloudK8SNodeName = "fsm_connector_service_k8s_node_name"
	connector.CloudK8SPort = "fsm_connector_service_k8s_port"
	connector.CloudK8SVia = "fsm_connector_service_via_gateway"

	return consulDiscoveryClient, nil
}
