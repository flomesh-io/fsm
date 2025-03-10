package cli

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/connector"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"
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
	// ZookeeperConnectors lookup identifier
	ZookeeperConnectors InformerKey = "ZookeeperConnectors"
	// MachineConnectors lookup identifier
	MachineConnectors InformerKey = "MachineConnectors"
	// GatewayConnectors lookup identifier
	GatewayConnectors InformerKey = "GatewayConnectors"
	// GatewayHTTPRoutes lookup identifier
	GatewayHTTPRoutes InformerKey = "GatewayHTTPRoutes"
	// GatewayGRPCRoutes lookup identifier
	GatewayGRPCRoutes InformerKey = "GatewayGRPCRoutes"
	// GatewayTCPRoutes lookup identifier
	GatewayTCPRoutes InformerKey = "GatewayTCPRoutes"
)

// client is the type used to represent the k8s client for the connector resources
type client struct {
	connectorProvider  string
	connectorNamespace string
	connectorName      string
	connectorSpec      interface{}
	connectorUID       string
	connectorHash      uint64
	clusterSet         string

	informers     *informers.InformerCollection
	msgBroker     *messaging.Broker
	msgWorkQueues *workerpool.WorkerPool
	// msgWorkerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	msgWorkerPoolSize int

	kubeConfig   *rest.Config
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface

	connectorClient connectorClientset.Interface
	machineClient   machineClientset.Interface
	gatewayClient   gwapi.Interface

	discClient connector.ServiceDiscoveryClient

	lock        sync.Mutex
	limiter     *rate.Limiter
	context     context.Context
	cancelFuncs []context.CancelFunc

	c2kContext *connector.C2KContext
	k2cContext *connector.K2CContext
	k2gContext *connector.K2GContext

	serviceInstanceIDFunc connector.ServiceInstanceIDFunc

	cache
}
