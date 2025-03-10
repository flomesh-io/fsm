package connector

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"
	eureka "github.com/hudl/fargo"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	nacos "github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	machinev1alpha1 "github.com/flomesh-io/fsm/pkg/apis/machine/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery"
)

const (
	NACOS_DEFAULT_CLUSTER = "DEFAULT"
)

type ServiceAddress struct {
	HostName string
	Port     int32
}

// AgentCheck represents a check known to the agent
type AgentCheck struct {
	CheckID   string
	ServiceID string
	Name      string
	Namespace string
	Type      string
	Status    string
	Output    string
}

func (ac *AgentCheck) ToConsul() *consul.AgentCheck {
	check := new(consul.AgentCheck)
	check.CheckID = ac.CheckID
	check.ServiceID = ac.ServiceID
	check.Name = ac.Name
	check.Namespace = ac.Namespace
	check.Type = ac.Type
	check.Status = ac.Status
	check.Output = ac.Output
	return check
}

type AgentWeights struct {
	Passing int
	Warning int
}

func (aw *AgentWeights) ToConsul() consul.AgentWeights {
	return consul.AgentWeights{
		Passing: aw.Passing,
		Warning: aw.Warning,
	}
}

func (aw *AgentWeights) FromConsul(w consul.AgentWeights) {
	aw.Passing = w.Passing
	aw.Warning = w.Warning
}

// AgentService represents a service known to the agent
type AgentService struct {
	ID           string
	InstanceId   string
	ClusterId    string
	MicroService MicroService
	Weights      AgentWeights
	Ports        map[MicroServicePort]MicroServicePort
	Tags         []string
	Meta         map[string]interface{}

	GRPCInterface    string
	GRPCMethods      []string
	GRPCInstanceMeta map[string]interface{}

	HealthCheck bool
}

func (as *AgentService) ToConsul() *consul.AgentService {
	agentService := new(consul.AgentService)
	agentService.ID = as.ID
	agentService.Service = as.MicroService.Service
	agentService.Namespace = as.MicroService.Namespace
	agentService.Address = as.MicroService.EndpointAddress().Get()
	agentService.Port = int(as.MicroService.EndpointPort().Get())
	agentService.Weights = as.Weights.ToConsul()
	if len(as.Tags) > 0 {
		agentService.Tags = append(agentService.Tags, as.Tags...)
	}
	agentService.Tags = append(agentService.Tags, "secure=false")
	if len(as.Meta) > 0 {
		agentService.Meta = make(map[string]string)
		for k, v := range as.Meta {
			agentService.Meta[k] = v.(string)
		}
	}
	return agentService
}

func (as *AgentService) FromConsul(ins *consul.AgentService) {
	as.ID = ins.ID
	as.MicroService.Service = ins.Service
	as.MicroService.Namespace = ins.Namespace
	as.MicroService.Protocol().SetVar(ProtocolHTTP)
	as.MicroService.Endpoint().Set(MicroServiceAddress(ins.Address), MicroServicePort(ins.Port))
	as.Weights.FromConsul(ins.Weights)
	if len(ins.Tags) > 0 {
		as.Tags = append(as.Tags, ins.Tags...)
	}
	if len(ins.Meta) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range ins.Meta {
			as.Meta[k] = v
		}
	}
}

func (as *AgentService) FromEureka(ins *eureka.Instance) {
	if ins == nil {
		return
	}
	as.ID = ins.Id()
	as.MicroService.Service = strings.ToLower(ins.VipAddress)
	as.InstanceId = ins.InstanceId
	as.MicroService.Protocol().SetVar(ProtocolHTTP)
	as.MicroService.Endpoint().Set(MicroServiceAddress(ins.IPAddr), MicroServicePort(ins.Port))
	metadata := ins.Metadata.GetMap()
	if len(metadata) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range metadata {
			as.Meta[k] = v
		}
	}
}

func (as *AgentService) FromNacos(ins *nacos.Instance) {
	if ins == nil {
		return
	}
	as.ID = ins.InstanceId
	as.MicroService.Service = strings.ToLower(strings.Split(ins.ServiceName, constant.SERVICE_INFO_SPLITER)[1])
	as.InstanceId = ins.InstanceId
	as.MicroService.Protocol().SetVar(ProtocolHTTP)
	as.MicroService.Endpoint().Set(MicroServiceAddress(ins.Ip), MicroServicePort(ins.Port))
	if len(ins.Metadata) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range ins.Metadata {
			as.Meta[k] = v
		}
	}
}

func (as *AgentService) FromZookeeper(ins discovery.ServiceInstance) {
	if ins == nil {
		return
	}
	as.ID = ins.InstanceId()
	as.MicroService.Service = ins.ServiceName()
	as.InstanceId = ins.InstanceId()
	as.MicroService.Endpoint().Set(MicroServiceAddress(ins.InstanceIP()), MicroServicePort(ins.InstancePort()))
	if MicroServiceProtocol(ins.ServiceSchema()) == ProtocolGRPC {
		as.GRPCInterface = ins.ServiceInterface()
		as.GRPCMethods = append(as.GRPCMethods, ins.ServiceMethods()...)
		as.MicroService.Protocol().SetVar(ProtocolGRPC)
	} else {
		as.MicroService.Protocol().SetVar(ProtocolHTTP)
	}
	if metadata := ins.Metadatas(); len(metadata) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range metadata {
			as.Meta[k] = v
			if strings.EqualFold(CloudK8SPort, k) && len(v) > 0 {
				as.Ports = make(map[MicroServicePort]MicroServicePort)
				portPairs := strings.Split(v, ",")
				for _, portPair := range portPairs {
					ports := strings.Split(portPair, ":")
					if len(ports) == 2 {
						if nv, err := strconv.ParseInt(ports[0], 10, 32); err == nil && nv > 0 && nv < math.MaxInt32 {
							targetPort := int32(nv)
							if nv, err := strconv.ParseInt(ports[1], 10, 32); err == nil && nv > 0 && nv < math.MaxInt32 {
								svcPort := int32(nv)
								as.Ports[MicroServicePort(targetPort)] = MicroServicePort(svcPort)
							}
						}
					}
				}
			}
		}
	}
}

func (as *AgentService) FromVM(vm machinev1alpha1.VirtualMachine, svc machinev1alpha1.ServiceSpec) {
	as.ID = fmt.Sprintf("%s-%s", svc.ServiceName, vm.UID)
	as.MicroService.Service = svc.ServiceName
	as.InstanceId = fmt.Sprintf("%s-%s-%s", vm.Name, svc.ServiceName, vm.UID)
	as.MicroService.Protocol().SetVar(ProtocolHTTP)
	as.MicroService.Endpoint().Set(MicroServiceAddress(vm.Spec.MachineIP), MicroServicePort(svc.Port))
	metadata := vm.Labels
	if len(metadata) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range metadata {
			as.Meta[k] = v
		}
	}
}

type CatalogDeregistration struct {
	ctv1.NamespacedService

	Node       string
	ServiceID  string
	ServiceRef string
}

func (cdr *CatalogDeregistration) ToConsul() *consul.CatalogDeregistration {
	r := new(consul.CatalogDeregistration)
	r.Node = cdr.Node
	r.ServiceID = cdr.ServiceID
	r.Namespace = cdr.Namespace
	return r
}

func (cdr *CatalogDeregistration) ToEureka() *eureka.Instance {
	r := new(eureka.Instance)
	r.InstanceId = cdr.ServiceID
	r.App = strings.ToUpper(cdr.Service)
	return r
}

func (cdr *CatalogDeregistration) ToNacos() *vo.DeregisterInstanceParam {
	svcInfoSegs := strings.Split(cdr.ServiceID, constant.SERVICE_INFO_SPLITER)
	if len(svcInfoSegs) < 2 {
		return nil
	}
	r := new(vo.DeregisterInstanceParam)
	r.ServiceName = svcInfoSegs[1]
	insInfoSegs := strings.Split(svcInfoSegs[0], constant.NAMING_INSTANCE_ID_SPLITTER)
	r.Ip = insInfoSegs[0]
	r.Port, _ = strconv.ParseUint(insInfoSegs[1], 10, 64)
	r.Cluster = insInfoSegs[2]
	r.GroupName = insInfoSegs[3]
	r.Ephemeral = true
	return r
}

func (cdr *CatalogDeregistration) ToZookeeper(ops discovery.FuncOps) discovery.ServiceInstance {
	instance := ops.NewInstance(cdr.Service, cdr.ServiceRef)
	if err := instance.Unmarshal("", []byte(cdr.Node)); err == nil {
		return instance
	} else {
		return nil
	}
}

type CatalogRegistration struct {
	Node           string
	Address        string
	NodeMeta       map[string]string
	Service        *AgentService
	Check          *AgentCheck
	SkipNodeUpdate bool
}

func (cr *CatalogRegistration) ToConsul() *consul.CatalogRegistration {
	r := new(consul.CatalogRegistration)
	r.Node = cr.Node
	r.Address = cr.Address
	if len(cr.NodeMeta) > 0 {
		r.NodeMeta = make(map[string]string)
		for k, v := range cr.NodeMeta {
			r.NodeMeta[k] = v
		}
	}
	if cr.Service != nil {
		r.Service = cr.Service.ToConsul()
	}
	if cr.Check != nil {
		r.Check = cr.Check.ToConsul()
	}
	r.SkipNodeUpdate = cr.SkipNodeUpdate
	return r
}

func (cr *CatalogRegistration) ToEureka() *eureka.Instance {
	r := new(eureka.Instance)
	if len(cr.NodeMeta) > 0 {
		for k, v := range cr.NodeMeta {
			r.SetMetadataString(k, v)
		}
	}
	if cr.Service != nil {
		r.UniqueID = func(i eureka.Instance) string {
			return cr.Service.ID
		}
		r.InstanceId = cr.Service.ID
		r.HostName = cr.Service.MicroService.EndpointAddress().Get()
		r.IPAddr = cr.Service.MicroService.EndpointAddress().Get()
		r.App = strings.ToUpper(cr.Service.MicroService.Service)
		r.VipAddress = strings.ToUpper(cr.Service.MicroService.Service)
		r.SecureVipAddress = strings.ToUpper(cr.Service.MicroService.Service)
		r.Port = int(cr.Service.MicroService.EndpointPort().Get())
		r.PortEnabled = true
		r.Status = eureka.UP
		r.DataCenterInfo = eureka.DataCenterInfo{Name: eureka.MyOwn}
		rMetadata := r.Metadata.GetMap()
		if len(cr.Service.Meta) > 0 {
			for k, v := range cr.Service.Meta {
				rMetadata[k] = v
			}
		}
	}
	return r
}

func (cr *CatalogRegistration) ToNacos(cluster, group string, weight float64) *vo.RegisterInstanceParam {
	r := new(vo.RegisterInstanceParam)
	r.Metadata = make(map[string]string)
	if len(cr.NodeMeta) > 0 {
		for k, v := range cr.NodeMeta {
			r.Metadata[k] = v
		}
	}
	if cr.Service != nil {
		r.ClusterName = cluster
		r.GroupName = group
		r.ServiceName = strings.ToLower(cr.Service.MicroService.Service)
		r.Ip = cr.Service.MicroService.EndpointAddress().Get()
		r.Port = uint64(cr.Service.MicroService.EndpointPort().Get())
		r.Weight = weight
		r.Enable = true
		r.Healthy = true
		r.Ephemeral = true
		if len(cr.Service.Meta) > 0 {
			for k, v := range cr.Service.Meta {
				r.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return r
}

func (cr *CatalogRegistration) ToZookeeper(adaptor discovery.FuncOps) (discovery.ServiceInstance, error) {
	var r discovery.ServiceInstance
	if cr.Service.MicroService.protocol == ProtocolGRPC {
		r = adaptor.NewInstance(cr.Service.GRPCInterface, "")
	} else {
		r = adaptor.NewInstance(cr.Service.MicroService.Service, "")
	}

	if err := r.Unmarshal(
		fmt.Sprintf("%s://%s:%d",
			*cr.Service.MicroService.Protocol(),
			*cr.Service.MicroService.EndpointAddress(),
			*cr.Service.MicroService.EndpointPort()),
		[]byte(cr.Service.MicroService.EndpointAddress().Get())); err != nil {
		return nil, err
	}
	if cr.Service.MicroService.protocol == ProtocolGRPC {
		if cr.Service.GRPCInstanceMeta != nil {
			for k, v := range cr.Service.GRPCInstanceMeta {
				r.SetMetadata(k, fmt.Sprintf("%v", v))
			}
		}
		if cr.Service.Meta != nil {
			if clusterSetKey, exists := cr.Service.Meta[ClusterSetKey]; exists {
				r.SetMetadata(ClusterSetKey, fmt.Sprintf("%v", clusterSetKey))
			}
			if connectUIDKey, exists := cr.Service.Meta[ConnectUIDKey]; exists {
				r.SetMetadata(ConnectUIDKey, fmt.Sprintf("%v", connectUIDKey))
			}
			if grpcViaGateway, exists := cr.Service.Meta[CloudGRPCViaGateway]; exists {
				r.SetMetadata(CloudGRPCViaGateway, fmt.Sprintf("%v", grpcViaGateway))
			}
			if viaGatewayMode, exists := cr.Service.Meta[CloudViaGatewayMode]; exists {
				r.SetMetadata(CloudViaGatewayMode, fmt.Sprintf("%v", viaGatewayMode))
			}
		}
	} else {
		if cr.Service.Meta != nil {
			for k, v := range cr.Service.Meta {
				r.SetMetadata(k, fmt.Sprintf("%v", v))
			}
		}
	}
	_, _ = r.Marshal()
	return r, nil
}

type CatalogService struct {
	Node        string
	ServiceID   string
	ServiceName string
	ServiceRef  string
}

func (cs *CatalogService) FromConsul(svc *consul.CatalogService) {
	if svc == nil {
		return
	}
	cs.Node = svc.Node
	cs.ServiceID = svc.ServiceID
	cs.ServiceName = svc.ServiceName
}

func (cs *CatalogService) FromEureka(svc *eureka.Instance) {
	if svc == nil {
		return
	}
	cs.Node = svc.DataCenterInfo.Name
	cs.ServiceID = svc.Id()
	cs.ServiceName = strings.ToLower(svc.App)
}

func (cs *CatalogService) FromNacos(svc *nacos.Instance) {
	if svc == nil {
		return
	}
	cs.Node = svc.ClusterName
	cs.ServiceID = svc.InstanceId
	cs.ServiceName = strings.ToLower(strings.Split(svc.ServiceName, constant.SERVICE_INFO_SPLITER)[1])
}

func (cs *CatalogService) FromZookeeper(svc discovery.ServiceInstance) {
	if svc == nil {
		return
	}
	cs.Node = svc.InstanceIP()
	cs.ServiceID = svc.InstanceId()
	cs.ServiceName = svc.ServiceName()
	cs.ServiceRef = svc.InstanceId()
}

// QueryOptions are used to parameterize a query
type QueryOptions struct {
	// AllowStale allows any Consul server (non-leader) to service
	// a read. This allows for lower latency and higher throughput
	AllowStale bool

	// Namespace overrides the `default` namespace
	// Note: Namespaces are available only in Consul Enterprise
	Namespace string

	// WaitIndex is used to enable a blocking query. Waits
	// until the timeout or the next index is reached
	WaitIndex uint64

	// WaitTime is used to bound the duration of a wait.
	// Defaults to that of the Config, but can be overridden.
	WaitTime time.Duration

	// Providing a peer name in the query option
	Peer string

	// Filter requests filtering data prior to it being returned. The string
	// is a go-bexpr compatible expression.
	Filter string

	// ctx is an optional context pass through to the underlying HTTP
	// request layer. Use Context() and WithContext() to manage this.
	ctx context.Context
}

func (o *QueryOptions) Context() context.Context {
	if o != nil && o.ctx != nil {
		return o.ctx
	}
	return context.Background()
}

func (o *QueryOptions) WithContext(ctx context.Context) *QueryOptions {
	o2 := new(QueryOptions)
	if o != nil {
		*o2 = *o
	}
	o2.ctx = ctx
	return o2
}

func (o *QueryOptions) ToConsul() *consul.QueryOptions {
	opts := new(consul.QueryOptions)
	opts.AllowStale = o.AllowStale
	opts.Namespace = o.Namespace
	opts.WaitIndex = o.WaitIndex
	opts.WaitTime = o.WaitTime
	opts.Peer = o.Peer
	opts.Filter = o.Filter
	opts.WithContext(o.Context())
	return opts
}

type ServiceDiscoveryClient interface {
	CatalogServices(q *QueryOptions) ([]ctv1.NamespacedService, error)
	CatalogInstances(service string, q *QueryOptions) ([]*AgentService, error)
	RegisteredInstances(service string, q *QueryOptions) ([]*CatalogService, error)
	RegisteredServices(q *QueryOptions) ([]ctv1.NamespacedService, error)
	Register(reg *CatalogRegistration) error
	Deregister(dereg *CatalogDeregistration) error
	EnableNamespaces() bool
	EnsureNamespaceExists(ns string) (bool, error)
	RegisteredNamespace(kubeNS string) string
	MicroServiceProvider() ctv1.DiscoveryServiceProvider
	IsInternalServices() bool
	Close()
}

const (
	// HealthAny is special, and is used as a wild card,
	// not as a specific state.
	HealthAny      = "any"
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
	HealthMaint    = "maintenance"
)
