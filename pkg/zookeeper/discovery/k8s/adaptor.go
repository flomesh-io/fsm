package k8s

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/flomesh-io/fsm/pkg/zookeeper/discovery"
)

type adaptor struct {
	discovery.NameAdaptor
}

func NewAdaptor() discovery.FuncOps {
	ops := adaptor{
		NameAdaptor: discovery.NameAdaptor{
			C2KNamesCache: expirable.NewLRU[string, string](1024*16, nil, time.Second*600),
			K2CNamesCache: expirable.NewLRU[string, string](1024*16, nil, time.Second*600),
		},
	}
	return &ops
}

func (op *adaptor) NewInstance(serviceName, instanceId string) discovery.ServiceInstance {
	return NewServiceInstance(serviceName, instanceId)
}

func (op *adaptor) PathForService(basePath, serviceName string) string {
	return path.Join(basePath, serviceName)
}

func (op *adaptor) PathForInstance(basePath, serviceName, instanceId string) string {
	return path.Join(basePath, serviceName, instanceId)
}

func (op *adaptor) ServiceInstanceId(basePath, instancePath string) (string, string, error) {
	instancePath = strings.TrimPrefix(instancePath, basePath)
	instancePath = strings.TrimPrefix(instancePath, string(os.PathSeparator))
	pathSlice := strings.Split(instancePath, string(os.PathSeparator))
	if len(pathSlice) < 2 {
		return "", "", fmt.Errorf("[ServiceDiscovery] path{%s} dont contain name and id", instancePath)
	}
	serviceName := pathSlice[0]
	instanceId := pathSlice[1]
	return serviceName, instanceId, nil
}
