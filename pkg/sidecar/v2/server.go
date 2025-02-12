package v2

import (
	"context"

	"k8s.io/client-go/kubernetes"

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
func NewXNetConfigServer(ctx context.Context,
	cfg configurator.Configurator,
	xnetworkController xnetwork.Controller,
	KubeClient kubernetes.Interface,
	kubecontroller k8s.Controller,
	msgBroker *messaging.Broker,
	nodeName string) *Server {
	server := Server{
		nodeName:           nodeName,
		ctx:                ctx,
		cfg:                cfg,
		xnetworkController: xnetworkController,
		kubeClient:         KubeClient,
		kubeController:     kubecontroller,
		msgBroker:          msgBroker,
		workQueues:         workerpool.NewWorkerPool(workerPoolSize),
		e4lbNatCache:       make(map[string]*E4LBNat),
	}

	return &server
}

func (s *Server) Start() error {
	for {
		if err := s.loadNatEntries(); err != nil {
			log.Warn().Msg(err.Error())
		} else {
			break
		}
	}

	s.ready = true
	return nil
}
