// Package gateway implements the Kubernetes client for the resources in the gateway.networking.k8s.io API group
package gateway

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	gwcache "github.com/flomesh-io/fsm/pkg/gateway/cache"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"

	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// client is the type used to represent the Kubernetes client for the gateway.networking.k8s.io API group
type client struct {
	informers  *informers.InformerCollection
	kubeClient kubernetes.Interface
	msgBroker  *messaging.Broker
	cfg        configurator.Configurator
	cache      gwcache.Cache
}

// Controller is the interface for the functionality provided by the resources part of the gateway.networking.k8s.io API group
type Controller interface {
	cache.ResourceEventHandler

	// Start runs the backend broadcast listener
	Start() error
}

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client, and have our
	// own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
)
