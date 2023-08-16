// Package repo contains the repository for the ingress controller
package repo

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/cache"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

const (
	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0
)

// Server is the ingress controller server
type Server struct {
	//fsmNamespace    string
	cfg configurator.Configurator
	//certManager     *certificate.Manager
	//ready           bool
	workQueues *workerpool.WorkerPool
	msgBroker  *messaging.Broker
	cache      *cache.Cache
	//retryProxiesJob func()
}

// NewServer creates a new ingress controller server
func NewServer(cfg configurator.Configurator, msgBroker *messaging.Broker, cache *cache.Cache) *Server {
	return &Server{
		//fsmNamespace: fsmNamespace,
		cfg: cfg,
		//certManager: certManager,
		workQueues: workerpool.NewWorkerPool(workerPoolSize),
		msgBroker:  msgBroker,
		cache:      cache,
	}
}
