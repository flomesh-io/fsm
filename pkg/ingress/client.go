package ingress

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// Initialize initializes the client and starts the ingress gateway certificate manager routine
func Initialize(kubeClient kubernetes.Interface, kubeController k8s.Controller, stop chan struct{},
	cfg configurator.Configurator, certProvider *certificate.Manager, msgBroker *messaging.Broker) error {
	c := &client{
		kubeClient:     kubeClient,
		kubeController: kubeController,
		cfg:            cfg,
		certProvider:   certProvider,
		msgBroker:      msgBroker,
	}

	if err := c.provisionIngressGatewayCert(stop); err != nil {
		return fmt.Errorf("Error provisioning ingress gateway certificate: %w", err)
	}

	return nil
}
