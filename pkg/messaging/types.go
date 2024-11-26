// Package messaging implements the messaging infrastructure between different
// components within the control plane.
package messaging

import (
	"github.com/cskr/pubsub"
	"k8s.io/client-go/util/workqueue"

	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("message-broker")
)

// Broker implements the message broker functionality
type Broker struct {
	queue                              workqueue.RateLimitingInterface
	proxyUpdatePubSub                  *pubsub.PubSub
	proxyUpdateCh                      chan proxyUpdateEvent
	ingressUpdatePubSub                *pubsub.PubSub
	ingressUpdateCh                    chan ingressUpdateEvent
	gatewayUpdatePubSub                *pubsub.PubSub
	gatewayUpdateCh                    chan gatewayUpdateEvent
	serviceUpdatePubSub                *pubsub.PubSub
	serviceUpdateCh                    chan serviceUpdateEvent
	connectorUpdatePubSub              *pubsub.PubSub
	connectorUpdateCh                  chan connectorUpdateEvent
	xnetworkUpdatePubSub               *pubsub.PubSub
	xnetworkUpdateCh                   chan xnetworkUpdateEvent
	mcsEventPubSub                     *pubsub.PubSub
	kubeEventPubSub                    *pubsub.PubSub
	certPubSub                         *pubsub.PubSub
	totalQEventCount                   uint64
	totalQProxyEventCount              uint64
	totalQIngressEventCount            uint64
	totalQGatewayEventCount            uint64
	totalQServiceEventCount            uint64
	totalQConnectorEventCount          uint64
	totalDispatchedProxyEventCount     uint64
	totalDispatchedIngressEventCount   uint64
	totalDispatchedGatewayEventCount   uint64
	totalDispatchedServiceEventCount   uint64
	totalDispatchedConnectorEventCount uint64
}

// proxyUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a proxy config update
type proxyUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}

// ingressUpdateEvent specifies the PubSubMessage and topic for an event that
// results in an ingress config update
type ingressUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}

// gatewayUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a gateway config update
type gatewayUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}

// mcsUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a mcs config update
type mcsUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}

// serviceUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a service config update
type serviceUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}

// connectorUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a connector config update
type connectorUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}

// xnetworkUpdateEvent specifies the PubSubMessage and topic for an event that
// results in a xnet policy update
type xnetworkUpdateEvent struct {
	msg   events.PubSubMessage
	topic string
}
