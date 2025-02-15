package v2

import (
	"context"

	"k8s.io/client-go/kubernetes"

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
)

var (
	log = logger.New("fsm-xnetwork-config")
)

type Server struct {
	ctx                context.Context
	cfg                configurator.Configurator
	nodeName           string
	xnetworkController xnetwork.Controller
	kubeClient         kubernetes.Interface
	kubeController     k8s.Controller
	msgBroker          *messaging.Broker
	workQueues         *workerpool.WorkerPool
	ready              bool

	cniBridge4 string
	cniBridge6 string

	xnatCache map[string]*XNat
}
