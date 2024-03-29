// Package ingress implements functionality to monitor and retrieve Kubernetes Ingress resources.
package ingress

import (
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

var (
	log = logger.New("ingress")
)

// client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type client struct {
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
	cfg            configurator.Configurator
	certProvider   *certificate.Manager
	msgBroker      *messaging.Broker
}
