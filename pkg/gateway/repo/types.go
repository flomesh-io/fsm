package repo

import (
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

type Server struct {
	fsmNamespace    string
	cfg             configurator.Configurator
	certManager     *certificate.Manager
	ready           bool
	workQueues      *workerpool.WorkerPool
	kubeController  k8s.Controller
	msgBroker       *messaging.Broker
	repoClient      *client.PipyRepoClient
	retryProxiesJob func()
}
