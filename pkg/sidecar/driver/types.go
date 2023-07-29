package driver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/health"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// Driver is an interface that must be implemented by a sidecar driver.
// Patch method is invoked by fsm-injector and Start method is invoked by fsm-controller
type Driver interface {
	Patch(ctx context.Context) error
	Start(ctx context.Context) (health.Probes, error)
}

// InjectorCtxKey the pointer is the key that a InjectorContext returns itself for.
var InjectorCtxKey int

// InjectorContext carries the arguments for invoking InjectorDriver.Patch
type InjectorContext struct {
	context.Context

	MeshName                     string
	KubeClient                   kubernetes.Interface
	FsmNamespace                 string
	FsmContainerPullPolicy       corev1.PullPolicy
	Configurator                 configurator.Configurator
	CertManager                  *certificate.Manager
	Pod                          *corev1.Pod
	PodOS                        string
	PodNamespace                 string
	ProxyUUID                    uuid.UUID
	BootstrapCertificateCNPrefix string
	BootstrapCertificate         *certificate.Certificate
	DryRun                       bool
}

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
	DebugHandlers    map[string]http.Handler
	CancelFunc       func()
	Stop             chan struct {
	}
}
