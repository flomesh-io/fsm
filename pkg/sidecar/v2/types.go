package v2

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
	"github.com/flomesh-io/fsm/pkg/xnetwork"
)

const (
	aclId   = uint16('c'<<8 | 'l')
	aclFlag = uint8('a')

	bridgeDev = `cni0`
)

var (
	log = logger.New("fsm-xnetwork-config")
)

type Server struct {
	ctx                context.Context
	cfg                configurator.Configurator
	xnetworkController xnetwork.Controller
	kubeController     k8s.Controller
	msgBroker          *messaging.Broker
	workQueues         *workerpool.WorkerPool
	ready              bool
}
