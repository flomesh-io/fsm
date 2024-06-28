// Package repo contains the repository for the gateway
package repo

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

const (
	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0
)

// Server is the gateway server
type Server struct {
	//fsmNamespace    string
	cfg configurator.Configurator
	//certManager     *certificate.Manager
	//ready      bool
	workQueues *workerpool.WorkerPool
	//kubeController  k8s.Controller
	msgBroker *messaging.Broker
	cache     processor.Processor
	//retryProxiesJob func()
}

// NewServer creates a new gateway server
func NewServer(cfg configurator.Configurator, msgBroker *messaging.Broker, cache processor.Processor) *Server {
	return &Server{
		//fsmNamespace: fsmNamespace,
		cfg: cfg,
		//certManager: certManager,
		workQueues: workerpool.NewWorkerPool(workerPoolSize),
		msgBroker:  msgBroker,
		cache:      cache,
	}
}
