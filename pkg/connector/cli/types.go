package cli

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/connector"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	machineClientset "github.com/flomesh-io/fsm/pkg/gen/client/machine/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

// InformerKey stores the different Informers we keep for K8s resources
type InformerKey string

const (
	// ConsulConnectors lookup identifier
	ConsulConnectors InformerKey = "ConsulConnectors"
	// EurekaConnectors lookup identifier
	EurekaConnectors InformerKey = "EurekaConnectors"
	// NacosConnectors lookup identifier
	NacosConnectors InformerKey = "NacosConnectors"
	// MachineConnectors lookup identifier
	MachineConnectors InformerKey = "MachineConnectors"
	// GatewayConnectors lookup identifier
	GatewayConnectors InformerKey = "GatewayConnectors"
)

// client is the type used to represent the k8s client for the connector resources
type client struct {
	connectorProvider string
	connectorName     string
	connectorSpec     interface{}
	connectorUID      string
	connectorHash     uint64

	informers     *informers.InformerCollection
	msgBroker     *messaging.Broker
	msgWorkQueues *workerpool.WorkerPool
	// msgWorkerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	msgWorkerPoolSize int

	kubeConfig    *rest.Config
	kubeClient    kubernetes.Interface
	configClient  configClientset.Interface
	machineClient machineClientset.Interface
	//discClient    provider.ServiceDiscoveryClient

	lock        sync.Mutex
	context     context.Context
	cancelFuncs []context.CancelFunc
}

// connectorControllerJob is the job to generate pipy policy json
type connectorControllerJob struct {
	// Optional waiter
	done                chan struct{}
	connectorController connector.ConnectorController
}
