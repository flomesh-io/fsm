package v2

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
	"github.com/flomesh-io/fsm/pkg/xnetwork"
)

const (
	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0
)

// NewXNetConfigServer creates a new xnetwork config Service server
func NewXNetConfigServer(ctx context.Context, cfg configurator.Configurator, xnetworkController xnetwork.Controller, kubecontroller k8s.Controller, msgBroker *messaging.Broker) *Server {
	server := Server{
		ctx:                ctx,
		cfg:                cfg,
		xnetworkController: xnetworkController,
		kubeController:     kubecontroller,
		msgBroker:          msgBroker,
		workQueues:         workerpool.NewWorkerPool(workerPoolSize),
	}

	return &server
}

func (s *Server) Start() error {
	s.ready = true
	return nil
}
