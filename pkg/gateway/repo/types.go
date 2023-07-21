package repo

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/gateway/cache"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

const (
	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0
)

type Server struct {
	//fsmNamespace    string
	cfg configurator.Configurator
	//certManager     *certificate.Manager
	ready      bool
	workQueues *workerpool.WorkerPool
	//kubeController  k8s.Controller
	msgBroker       *messaging.Broker
	cache           cache.Cache
	retryProxiesJob func()
}

func NewServer(cfg configurator.Configurator, msgBroker *messaging.Broker, cache cache.Cache) *Server {
	return &Server{
		//fsmNamespace: fsmNamespace,
		cfg: cfg,
		//certManager: certManager,
		workQueues: workerpool.NewWorkerPool(workerPoolSize),
		msgBroker:  msgBroker,
		cache:      cache,
	}
}
