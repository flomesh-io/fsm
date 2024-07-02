// Package gateway implements the Kubernetes client for the resources in the gateway.networking.k8s.io API group
package gateway

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/logger"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// client is the type used to represent the Kubernetes client for the gateway.networking.k8s.io API group
type client struct {
	msgBroker *messaging.Broker
	cfg       configurator.Configurator
	processor processor.Processor
}

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client, and have our
	// own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
)

var (
	log = logger.New("controller-gatewayapi")
)
