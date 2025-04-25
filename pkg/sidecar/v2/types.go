package v2

import (
	"context"
	"net"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/configurator"
	xnetworkClientset "github.com/flomesh-io/fsm/pkg/gen/client/xnetwork/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/utils/chm"
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
	xnetworkClient     xnetworkClientset.Interface
	msgBroker          *messaging.Broker
	workQueues         *workerpool.WorkerPool
	ready              bool

	enableE4lb bool
	enableMesh bool

	cniBridge4 string
	cniBridge6 string

	xnatCache map[string]*XNat
	eipCache  chm.ConcurrentMap[string, *e4lbNeigh]

	Leading bool
}

type e4lbTopo struct {
	existsE4lbNodes   bool
	nodeCache         map[string]bool
	nodeEipLayout     map[string]map[string]uint8
	eipNodeLayout     map[string]string
	eipSvcCache       map[string]uint8
	advertisementHash map[types.UID]uint64
}

type e4lbNeigh struct {
	eip     net.IP
	ifName  string
	ifIndex int
	macAddr net.HardwareAddr
	adv     bool
}
