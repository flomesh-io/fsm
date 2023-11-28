package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"
	eureka "github.com/hudl/fargo"
)

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

func (ac *AgentCheck) toConsul() *consul.AgentCheck {
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

func (aw *AgentWeights) toConsul() consul.AgentWeights {
	return consul.AgentWeights{
		Passing: aw.Passing,
		Warning: aw.Warning,
	}
}

func (aw *AgentWeights) fromConsul(w consul.AgentWeights) {
	aw.Passing = w.Passing
	aw.Warning = w.Warning
}

// AgentService represents a service known to the agent
type AgentService struct {
	ID         string
	Service    string
	InstanceId string
	Namespace  string
	Address    string
	Port       int
	Weights    AgentWeights
	Tags       []string
	Meta       map[string]interface{}
}

func (as *AgentService) toConsul() *consul.AgentService {
	agentService := new(consul.AgentService)
	agentService.ID = as.ID
	agentService.Service = as.Service
	agentService.Namespace = as.Namespace
	agentService.Address = as.Address
	agentService.Port = as.Port
	agentService.Weights = as.Weights.toConsul()
	if len(as.Tags) > 0 {
		agentService.Tags = append(agentService.Tags, as.Tags...)
	}
	if len(as.Meta) > 0 {
		agentService.Meta = make(map[string]string)
		for k, v := range as.Meta {
			agentService.Meta[k] = v.(string)
		}
	}
	return agentService
}

func (as *AgentService) fromConsul(agentService *consul.AgentService) {
	as.ID = agentService.ID
	as.Service = agentService.Service
	as.Namespace = agentService.Namespace
	as.Address = agentService.Address
	as.Port = agentService.Port
	as.Weights.fromConsul(agentService.Weights)
	if len(agentService.Tags) > 0 {
		as.Tags = append(as.Tags, agentService.Tags...)
	}
	if len(agentService.Meta) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range agentService.Meta {
			as.Meta[k] = v
		}
	}
}

func (as *AgentService) fromEureka(ins *eureka.Instance) {
	if ins == nil {
		return
	}
	as.ID = ins.Id()
	as.Service = ins.VipAddress
	as.InstanceId = ins.InstanceId
	as.Address = ins.IPAddr
	as.Port = ins.Port
	metadata := ins.Metadata.GetMap()
	if len(metadata) > 0 {
		as.Meta = make(map[string]interface{})
		for k, v := range metadata {
			as.Meta[k] = v
		}
	}
}

type CatalogDeregistration struct {
	Node      string
	ServiceID string
	Service   string
	Namespace string
}

func (cdr *CatalogDeregistration) toConsul() *consul.CatalogDeregistration {
	r := new(consul.CatalogDeregistration)
	r.Node = cdr.Node
	r.ServiceID = cdr.ServiceID
	r.Namespace = cdr.Namespace
	return r
}

func (cdr *CatalogDeregistration) toEureka() *eureka.Instance {
	r := new(eureka.Instance)
	r.InstanceId = cdr.ServiceID
	r.App = cdr.Service
	return r
}

type CatalogRegistration struct {
	Node           string
	Address        string
	NodeMeta       map[string]string
	Service        *AgentService
	Check          *AgentCheck
	SkipNodeUpdate bool
}

func (cr *CatalogRegistration) toConsul() *consul.CatalogRegistration {
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
		r.Service = cr.Service.toConsul()
	}
	if cr.Check != nil {
		r.Check = cr.Check.toConsul()
	}
	r.SkipNodeUpdate = cr.SkipNodeUpdate
	return r
}

func (cr *CatalogRegistration) toEureka() *eureka.Instance {
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
		r.HostName = cr.Service.Address
		r.IPAddr = cr.Service.Address
		r.App = cr.Service.Service
		r.VipAddress = strings.ToLower(cr.Service.Service)
		r.SecureVipAddress = strings.ToLower(cr.Service.Service)
		r.Port = cr.Service.Port
		r.Status = eureka.UP
		r.DataCenterInfo = eureka.DataCenterInfo{Name: eureka.MyOwn}

		r.HomePageUrl = fmt.Sprintf("http://%s:%d/", cr.Service.Address, cr.Service.Port)
		r.StatusPageUrl = fmt.Sprintf("http://%s:%d/actuator/info", cr.Service.Address, cr.Service.Port)
		r.HealthCheckUrl = fmt.Sprintf("http://%s:%d/actuator/health", cr.Service.Address, cr.Service.Port)
	}
	return r
}

type CatalogService struct {
	Node        string
	ServiceID   string
	ServiceName string
}

func (cs *CatalogService) fromConsul(svc *consul.CatalogService) {
	if svc == nil {
		return
	}
	cs.Node = svc.Node
	cs.ServiceID = svc.ServiceID
	cs.ServiceName = svc.ServiceName
}

func (cs *CatalogService) fromEureka(svc *eureka.Instance) {
	if svc == nil {
		return
	}
	cs.Node = svc.DataCenterInfo.Name
	cs.ServiceID = svc.Id()
	cs.ServiceName = svc.App
}

type CatalogNodeServiceList struct {
	Services []*AgentService
}

func (cnsl *CatalogNodeServiceList) fromConsul(svcList *consul.CatalogNodeServiceList) {
	if svcList == nil || len(svcList.Services) == 0 {
		return
	}
	for _, svc := range svcList.Services {
		agentService := new(AgentService)
		agentService.fromConsul(svc)
		cnsl.Services = append(cnsl.Services, agentService)
	}
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

func (o *QueryOptions) toConsul() *consul.QueryOptions {
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
	NodeServiceList(node string, q *QueryOptions) (*CatalogNodeServiceList, error)
	CatalogServices(q *QueryOptions) (map[string][]string, error)
	CatalogService(service, tag string, q *QueryOptions) ([]*CatalogService, error)
	HealthService(service, tag string, q *QueryOptions, passingOnly bool) ([]*AgentService, error)
	Register(reg *CatalogRegistration) error
	Deregister(dereg *CatalogDeregistration) error
	EnsureNamespaceExists(ns string, crossNSAClPolicy string) (bool, error)
	MicroServiceProvider() string
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

// CloudNamespace returns the cloud namespace that a service should be
// registered in based on the namespace options. It returns an
// empty string if namespaces aren't enabled.
func CloudNamespace(kubeNS string, enableCloudNamespaces bool, cloudDestNS string, enableMirroring bool, mirroringPrefix string) string {
	if !enableCloudNamespaces {
		return ""
	}

	// Mirroring takes precedence.
	if enableMirroring {
		return fmt.Sprintf("%s%s", mirroringPrefix, kubeNS)
	}

	return cloudDestNS
}
