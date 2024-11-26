package context

import (
	"context"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"

	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/repo"

	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// ControllerCtxKey the pointer is the key that a ControllerContext returns itself for.
var ControllerCtxKey int

// ControllerContext carries the arguments for invoking ControllerDriver.Start
type ControllerContext struct {
	context.Context

	ProxyServerPort  uint32
	ProxyServiceCert *certificate.Certificate
	FsmNamespace     string
	KubeConfig       *rest.Config
	Configurator     configurator.Configurator
	MeshCatalog      catalog.MeshCataloger
	CertManager      *certificate.Manager
	MsgBroker        *messaging.Broker
	CancelFunc       func()
	Stop             chan struct{}

	// Merge with FSM ControllerContext to simplify the code
	client.Client
	Manager             manager.Manager
	Scheme              *runtime.Scheme
	KubeClient          kubernetes.Interface
	RepoClient          *repo.PipyRepoClient
	InformerCollection  *fsminformers.InformerCollection
	GatewayEventHandler gwtypes.Controller
	StatusUpdater       status.Updater
	MeshName            string
	TrustDomain         string
	FSMVersion          string
}
