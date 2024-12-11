package nebula

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery/nebula/urlenc"
)

const (
	PickFirstLoadBalance        = "pick_first"
	RoundRobinLoadBalance       = "round_robin"
	WeightRoundRobinLoadBalance = "weight_round_robin"
	ConsistentHashLoadBalance   = "consistent_hash"
)

type ServiceInstance struct {
	serviceName string
	instanceId  string

	Schema string `urlenc:"-"`
	Addr   string `urlenc:"-"`
	IP     string `urlenc:"-"`
	Port   int    `urlenc:"-"`
	Node   string `urlenc:"-"`

	Interface string `urlenc:"interface"`
	Methods   string `urlenc:"methods"`

	Application string `urlenc:"application"`
	Project     string `urlenc:"project"`
	Owner       string `urlenc:"owner"`
	Ops         string `urlenc:"ops,omitempty"`
	Category    string `urlenc:"category"`
	Timestamp   uint64 `urlenc:"timestamp"`
	GRPC        string `urlenc:"grpc"`
	PID         uint32 `urlenc:"pid"`
	Group       bool   `urlenc:"group,omitempty"`
	Weight      uint32 `urlenc:"weight"`
	Deprecated  bool   `urlenc:"deprecated"`
	Master      bool   `urlenc:"master"`

	DefaultAsync       bool   `urlenc:"default.async"`
	DefaultCluster     string `urlenc:"default.cluster"`
	DefaultConnections uint32 `urlenc:"default.connections"`
	DefaultLoadBalance string `urlenc:"default.loadbalance"`
	DefaultRequests    uint32 `urlenc:"default.requests"`
	DefaultReties      uint32 `urlenc:"default.reties"`
	DefaultTimeout     uint32 `urlenc:"default.timeout"`

	ServiceType     string `urlenc:"service.type"`
	RealIP          string `urlenc:"real.ip"`
	RealPort        uint16 `urlenc:"real.port"`
	AccessProtected bool   `urlenc:"access.protected"`

	Accesslog bool   `urlenc:"accesslog"`
	Anyhost   bool   `urlenc:"anyhost"`
	Dynamic   bool   `urlenc:"dynamic"`
	Token     bool   `urlenc:"token"`
	Side      string `urlenc:"side"`
	Version   string `urlenc:"version"`

	FsmConnectorServiceClusterSet     string `urlenc:"fsm.connector.service.cluster.set,omitempty"`
	FsmConnectorServiceConnectorUid   string `urlenc:"fsm.connector.service.connector.uid,omitempty"`
	FsmConnectorServiceGRPCViaGateway string `urlenc:"fsm.connector.service.grpc.via.gateway"`
	FsmConnectorServiceViaGatewayMode string `urlenc:"fsm.connector.service.via.gateway.mode"`
}

func NewServiceInstance(serviceName, instanceId string) *ServiceInstance {
	return &ServiceInstance{
		serviceName:        serviceName,
		instanceId:         instanceId,
		AccessProtected:    false,
		DefaultConnections: 20,
		DefaultRequests:    2000,
		DefaultLoadBalance: PickFirstLoadBalance,
		Weight:             100,
		Deprecated:         false,
		Master:             true,
	}
}

func (ins *ServiceInstance) ServiceName() string {
	return ins.serviceName
}

func (ins *ServiceInstance) ServiceSchema() string {
	return ins.Schema
}

func (ins *ServiceInstance) ServiceInterface() string {
	return ins.Interface
}

func (ins *ServiceInstance) ServiceMethods() []string {
	var methods []string
	if len(ins.Methods) > 0 {
		segs := strings.Split(ins.Methods, `,`)
		if len(segs) > 0 {
			for _, method := range segs {
				if len(method) > 0 {
					methods = append(methods, method)
				}
			}
		}
	}
	return methods
}

func (ins *ServiceInstance) InstanceId() string {
	return ins.instanceId
}

func (ins *ServiceInstance) Marshal() ([]byte, error) {
	if bytes, err := urlenc.Encode(ins); err != nil {
		return nil, err
	} else {
		instanceUrl := url.URL{
			Scheme:   ins.Schema,
			Host:     ins.Addr,
			RawQuery: string(bytes),
		}
		if len(instanceUrl.Host) == 0 {
			instanceUrl.Host = fmt.Sprintf("%s:%d", ins.IP, ins.Port)
		}
		ins.instanceId = url.QueryEscape(instanceUrl.String())
		return []byte(ins.instanceId), nil
	}
}

func (ins *ServiceInstance) Unmarshal(instanceId string, data []byte) error {
	var err error
	var instancePath string
	var instanceUrl *url.URL

	if len(instanceId) > 0 {
		if instancePath, err = url.QueryUnescape(instanceId); err != nil {
			return err
		}
	} else {
		if instancePath, err = url.QueryUnescape(ins.instanceId); err != nil {
			return err
		}
	}

	if instanceUrl, err = url.Parse(instancePath); err != nil {
		return err
	}
	if len(instanceUrl.RawQuery) > 0 {
		if err = urlenc.Decode([]byte(instanceUrl.RawQuery), ins); err != nil {
			fmt.Println(err.Error())
			return err
		}
	}
	ins.Schema = instanceUrl.Scheme
	ins.Addr = instanceUrl.Host
	ins.IP = instanceUrl.Hostname()
	ins.Port, _ = strconv.Atoi(instanceUrl.Port())
	ins.Node = string(data)
	return nil
}

func (ins *ServiceInstance) InstanceIP() string {
	return ins.IP
}

func (ins *ServiceInstance) InstancePort() int {
	return ins.Port
}

func (ins *ServiceInstance) GetMetadata(key string) (string, bool) {
	if field, exists := mapFields[key]; exists {
		return field.getter(ins), true
	}
	return "", false
}

func (ins *ServiceInstance) SetMetadata(key, value string) error {
	if field, exists := mapFields[key]; exists {
		return field.setter(ins, value)
	}
	return nil
}

func (ins *ServiceInstance) Metadatas() map[string]string {
	metadata := make(map[string]string)
	for key, field := range mapFields {
		metadata[key] = field.getter(ins)
	}
	return metadata
}
