package k8s

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/flomesh-io/fsm/pkg/zookeeper/urlenc"
)

type ServiceInstance struct {
	serviceName string
	instanceId  string

	Schema string `urlenc:"-"`
	Addr   string `urlenc:"-"`
	IP     string `urlenc:"-"`
	Port   int    `urlenc:"-"`
	Node   string `urlenc:"-"`

	Extends map[string]string `urlenc:"-"`

	FsmConnectorServicePort           string `urlenc:"fsm.connector.service.k8s.port,omitempty"`
	FsmConnectorServiceNamespace      string `urlenc:"fsm.connector.service.k8s.ns,omitempty"`
	FsmConnectorServiceClusterSet     string `urlenc:"fsm.connector.service.cluster.set,omitempty"`
	FsmConnectorServiceConnectorUid   string `urlenc:"fsm.connector.service.connector.uid,omitempty"`
	FsmConnectorServiceHTTPViaGateway string `urlenc:"fsm.connector.service.http.via.gateway"`
	FsmConnectorServiceViaGatewayMode string `urlenc:"fsm.connector.service.via.gateway.mode"`
}

func NewServiceInstance(serviceName, instanceId string) *ServiceInstance {
	return &ServiceInstance{
		serviceName: serviceName,
		instanceId:  instanceId,
	}
}

func (ins *ServiceInstance) ServiceName() string {
	return ins.serviceName
}

func (ins *ServiceInstance) ServiceSchema() string {
	return ins.Schema
}

func (ins *ServiceInstance) ServiceInterface() string {
	return ""
}

func (ins *ServiceInstance) ServiceMethods() []string {
	return []string{}
}

func (ins *ServiceInstance) InstanceId() string {
	return ins.instanceId
}

func (ins *ServiceInstance) Marshal() ([]byte, error) {
	if bytes, err := urlenc.Encode(ins); err != nil {
		return nil, err
	} else {
		if rawValues, err := url.ParseQuery(string(bytes)); err != nil {
			return nil, err
		} else {
			if len(ins.Extends) > 0 {
				if bytes, err = urlenc.Encode(ins.Extends); err != nil {
					return nil, err
				} else {
					if extValues, err := url.ParseQuery(string(bytes)); err == nil {
						for k, v := range extValues {
							rawValues[k] = v
						}
					} else {
						return nil, err
					}
				}
			}
			instanceUrl := url.URL{
				Scheme:   ins.Schema,
				Host:     ins.Addr,
				RawQuery: rawValues.Encode(),
			}
			if len(instanceUrl.Host) == 0 {
				instanceUrl.Host = fmt.Sprintf("%s:%d", ins.IP, ins.Port)
			}
			ins.instanceId = url.QueryEscape(instanceUrl.String())
			return []byte(ins.instanceId), nil
		}
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
		if extQuery, err := urlenc.Decode([]byte(instanceUrl.RawQuery), ins); err != nil {
			return err
		} else if len(extQuery) > 0 {
			ins.Extends = make(map[string]string)
			if _, err = urlenc.Decode(extQuery, &ins.Extends); err != nil {
				return err
			}
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
