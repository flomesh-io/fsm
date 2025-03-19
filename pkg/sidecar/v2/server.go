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
	kubeController k8s.Controller,
	msgBroker *messaging.Broker,
	nodeName, cniBridge4, cniBridge6 string) *Server {
	server := &Server{
		nodeName:           nodeName,
		ctx:                ctx,
		cfg:                cfg,
		xnetworkController: xnetworkController,
		kubeClient:         KubeClient,
		kubeController:     kubeController,
		msgBroker:          msgBroker,
		workQueues:         workerpool.NewWorkerPool(workerPoolSize),
		cniBridge4:         cniBridge4,
		cniBridge6:         cniBridge6,
		xnatCache:          make(map[string]*XNat),
	}
	kubeController.AddObserveFilter(server.xNetDNSProxyUpstreamsObserveFilter)
	return server
}

func (s *Server) Start() error {
	retries := 0
	for {
		retries++
		if retries > 12 {
			log.Fatal().Msg(`timeout waiting for xnet to be ready`)
		}
		if err := s.loadNatEntries(); err != nil {
			if retries > 8 {
				log.Error().Err(err).Msg(`waiting for xnet to be ready ...`)
			} else if retries > 4 {
				log.Warn().Err(err).Msg(`waiting for xnet to be ready ...`)
			} else {
				log.Debug().Err(err).Msg(`waiting for xnet to be ready ...`)
			}
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	s.ready = true
	return nil
}
