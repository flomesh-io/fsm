package driver

import (
	"context"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/health"
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
