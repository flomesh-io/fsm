package v2

import (
	"context"
	"time"

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
	nodeName, cniBridge4, cniBridge6 string) *Server {
	server := Server{
		nodeName:           nodeName,
		ctx:                ctx,
		cfg:                cfg,
		xnetworkController: xnetworkController,
		kubeClient:         KubeClient,
		kubeController:     kubecontroller,
		msgBroker:          msgBroker,
		workQueues:         workerpool.NewWorkerPool(workerPoolSize),
		cniBridge4:         cniBridge4,
		cniBridge6:         cniBridge6,
		xnatCache:          make(map[string]*XNat),
	}

	return &server
}

func (s *Server) Start() error {
	for {
		if err := s.loadNatEntries(); err != nil {
			log.Warn().Msg(err.Error())
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	s.ready = true
	return nil
}
