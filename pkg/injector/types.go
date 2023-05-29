// Package injector implements FSM's automatic sidecar injection facility. The sidecar injector's mutating webhook
// admission controller intercepts pod creation requests to mutate the pod spec to inject the sidecar proxy.
package injector

import (
	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
)

const (
	// SidecarBootstrapConfigVolume defines sidecar bootstrap config volume.
	SidecarBootstrapConfigVolume = "sidecar-bootstrap-config-volume"
)

var log = logger.New("sidecar-injector")

// mutatingWebhook is the type used to represent the webhook for sidecar injection
type mutatingWebhook struct {
	kubeClient             kubernetes.Interface
	certManager            *certificate.Manager
	kubeController         k8s.Controller
	fsmNamespace           string
	meshName               string
	configurator           configurator.Configurator
	fsmContainerPullPolicy corev1.PullPolicy

	nonInjectNamespaces mapset.Set
}

// Config is the type used to represent the config options for the sidecar injection
type Config struct {
	// ListenPort defines the port on which the sidecar injector listens
	ListenPort int
}
