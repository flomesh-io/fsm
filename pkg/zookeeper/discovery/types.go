package discovery

import (
	"sync"

	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/zookeeper"
)

var (
	log = logger.New("fsm-zookeeper-discovery")
)

type ServiceDiscovery struct {
	client   *zookeeper.Client
	mutex    *sync.Mutex
	basePath string
	category string
	services *sync.Map
	ops      FuncOps
}

type ServiceInstance interface {
	ServiceSchema() string
	ServiceName() string
	ServiceInterface() string
	ServiceMethods() []string
	InstanceId() string

	InstanceIP() string
	InstancePort() int

	Metadatas() map[string]string
	GetMetadata(key string) (string, bool)
	SetMetadata(key, value string) error

	Marshal() ([]byte, error)
	Unmarshal(string, []byte) error
}

// serviceEntry contain a service instance
type serviceEntry struct {
	sync.Mutex
	instance ServiceInstance
}

type FuncOps interface {
	NewInstance(serviceName, instanceId string) ServiceInstance
	PathForService(basePath, serviceName string) (servicePath string)
	PathForInstance(basePath, serviceName, instanceId string) (instancePath string)
	ServiceInstanceId(basePath, instancePath string) (serviceName, instanceId string, err error)
	KtoCName(serviceName string) string
	CToKName(serviceName string) string
}
