package messaging

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cskr/pubsub"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	"github.com/flomesh-io/fsm/pkg/announcements"
	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/lru"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
)

const (
	// proxyUpdateSlidingWindow is the sliding window duration used to batch proxy update events
	proxyUpdateSlidingWindow = 2 * time.Second

	// proxyUpdateMaxWindow is the max window duration used to batch proxy update events, and is
	// the max amount of time a proxy update event can be held for batching before being dispatched.
	proxyUpdateMaxWindow = 10 * time.Second

	// ingressUpdateSlidingWindow is the sliding window duration used to batch ingress update events
	ingressUpdateSlidingWindow = 2 * time.Second

	// ingressUpdateMaxWindow is the max window duration used to batch ingress update events, and is
	// the max amount of time an ingress update event can be held for batching before being dispatched.
	ingressUpdateMaxWindow = 10 * time.Second

	// gatewayUpdateSlidingWindow is the sliding window duration used to batch gateway update events
	gatewayUpdateSlidingWindow = 2 * time.Second

	// gatewayUpdateMaxWindow is the max window duration used to batch gateway update events, and is
	// the max amount of time a gateway update event can be held for batching before being dispatched.
	gatewayUpdateMaxWindow = 10 * time.Second

	// serviceUpdateSlidingWindow is the sliding window duration used to batch service update events
	serviceUpdateSlidingWindow = 2 * time.Second

	// serviceUpdateMaxWindow is the max window duration used to batch service update events, and is
	// the max amount of time a service update event can be held for batching before being dispatched.
	serviceUpdateMaxWindow = 5 * time.Second

	// ConnectorUpdateSlidingWindow is the sliding window duration used to batch connector update events
	ConnectorUpdateSlidingWindow = 2 * time.Second

	// ConnectorUpdateMaxWindow is the max window duration used to batch connector update events, and is
	// the max amount of time a connector update event can be held for batching before being dispatched.
	ConnectorUpdateMaxWindow = 5 * time.Second

	// XNetworkUpdateSlidingWindow is the sliding window duration used to batch xnetwork update events
	XNetworkUpdateSlidingWindow = 2 * time.Second

	// XNetworkUpdateMaxWindow is the max window duration used to batch xnetwork update events, and is
	// the max amount of time a xnetwork update event can be held for batching before being dispatched.
	XNetworkUpdateMaxWindow = 5 * time.Second
)

// NewBroker returns a new message broker instance and starts the internal goroutine
// to process events added to the workqueue.
func NewBroker(stopCh <-chan struct{}) *Broker {
	rateLimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		// 1024*5 qps, 1024*8 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(1024*8), 1024*9)},
	)
	b := &Broker{
		queue:                 workqueue.NewRateLimitingQueue(rateLimiter),
		proxyUpdatePubSub:     pubsub.New(1024 * 10),
		proxyUpdateCh:         make(chan proxyUpdateEvent),
		ingressUpdatePubSub:   pubsub.New(1024 * 10),
		ingressUpdateCh:       make(chan ingressUpdateEvent),
		gatewayUpdatePubSub:   pubsub.New(1024 * 10),
		gatewayUpdateCh:       make(chan gatewayUpdateEvent),
		serviceUpdatePubSub:   pubsub.New(1024 * 10),
		serviceUpdateCh:       make(chan serviceUpdateEvent),
		connectorUpdatePubSub: pubsub.New(1024 * 10),
		connectorUpdateCh:     make(chan connectorUpdateEvent),
		xnetworkUpdatePubSub:  pubsub.New(1024 * 10),
		xnetworkUpdateCh:      make(chan xnetworkUpdateEvent),
		kubeEventPubSub:       pubsub.New(1024 * 10),
		certPubSub:            pubsub.New(1024 * 10),
		mcsEventPubSub:        pubsub.New(1024 * 10),
		//mcsUpdateCh:         make(chan mcsUpdateEvent),
	}

	go b.runWorkqueueProcessor(stopCh)
	go b.runProxyUpdateDispatcher(stopCh)
	go b.runIngressUpdateDispatcher(stopCh)
	go b.runGatewayUpdateDispatcher(stopCh)
	go b.runServiceUpdateDispatcher(stopCh)
	go b.runConnectorUpdateDispatcher(stopCh)
	go b.runXNetworkUpdateDispatcher(stopCh)
	go b.queueLenMetric(stopCh, 5*time.Second)

	return b
}

func (b *Broker) queueLenMetric(stop <-chan struct{}, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			metricsstore.DefaultMetricsStore.EventsQueued.Set(float64(b.queue.Len()))
		}
	}
}

// GetProxyUpdatePubSub returns the PubSub instance corresponding to proxy update events
func (b *Broker) GetProxyUpdatePubSub() *pubsub.PubSub {
	return b.proxyUpdatePubSub
}

// GetIngressUpdatePubSub returns the PubSub instance corresponding to ingress update events
func (b *Broker) GetIngressUpdatePubSub() *pubsub.PubSub {
	return b.ingressUpdatePubSub
}

// GetGatewayUpdatePubSub returns the PubSub instance corresponding to gateway update events
func (b *Broker) GetGatewayUpdatePubSub() *pubsub.PubSub {
	return b.gatewayUpdatePubSub
}

// GetServiceUpdatePubSub returns the PubSub instance corresponding to service update events
func (b *Broker) GetServiceUpdatePubSub() *pubsub.PubSub {
	return b.serviceUpdatePubSub
}

// GetConnectorUpdatePubSub returns the PubSub instance corresponding to connector update events
func (b *Broker) GetConnectorUpdatePubSub() *pubsub.PubSub {
	return b.connectorUpdatePubSub
}

// GetXNetworkUpdatePubSub returns the PubSub instance corresponding to xnetwork update events
func (b *Broker) GetXNetworkUpdatePubSub() *pubsub.PubSub {
	return b.xnetworkUpdatePubSub
}

// GetMCSEventPubSub returns the PubSub instance corresponding to MCS update events
func (b *Broker) GetMCSEventPubSub() *pubsub.PubSub {
	return b.mcsEventPubSub
}

// GetKubeEventPubSub returns the PubSub instance corresponding to k8s events
func (b *Broker) GetKubeEventPubSub() *pubsub.PubSub {
	return b.kubeEventPubSub
}

// GetCertPubSub returns the PubSub instance corresponding to certificate events
func (b *Broker) GetCertPubSub() *pubsub.PubSub {
	return b.certPubSub
}

// GetTotalQProxyEventCount returns the total number of events read from the workqueue
// pertaining to proxy updates
func (b *Broker) GetTotalQProxyEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQProxyEventCount)
}

// GetTotalDispatchedProxyEventCount returns the total number of events dispatched
// to subscribed proxies
func (b *Broker) GetTotalDispatchedProxyEventCount() uint64 {
	return atomic.LoadUint64(&b.totalDispatchedProxyEventCount)
}

// GetTotalQIngressEventCount returns the total number of events read from the workqueue
// pertaining to ingress updates
func (b *Broker) GetTotalQIngressEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQIngressEventCount)
}

// GetTotalDispatchedIngressEventCount returns the total number of events dispatched
// to subscribed ingresses
func (b *Broker) GetTotalDispatchedIngressEventCount() uint64 {
	return atomic.LoadUint64(&b.totalDispatchedIngressEventCount)
}

// GetTotalQGatewayEventCount returns the total number of events read from the workqueue
// pertaining to gateway updates
func (b *Broker) GetTotalQGatewayEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQGatewayEventCount)
}

// GetTotalDispatchedGatewayEventCount returns the total number of events dispatched
// to subscribed gateways
func (b *Broker) GetTotalDispatchedGatewayEventCount() uint64 {
	return atomic.LoadUint64(&b.totalDispatchedGatewayEventCount)
}

// GetTotalQServiceEventCount returns the total number of events read from the workqueue
// pertaining to service updates
func (b *Broker) GetTotalQServiceEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQServiceEventCount)
}

// GetTotalDispatchedServiceEventCount returns the total number of events dispatched
// to subscribed services
func (b *Broker) GetTotalDispatchedServiceEventCount() uint64 {
	return atomic.LoadUint64(&b.totalDispatchedServiceEventCount)
}

// GetTotalQConnectorEventCount returns the total number of events read from the workqueue
// pertaining to connector updates
func (b *Broker) GetTotalQConnectorEventCount() uint64 {
	return atomic.LoadUint64(&b.totalQConnectorEventCount)
}

// GetTotalDispatchedConnectorEventCount returns the total number of events dispatched
// to subscribed connectors
func (b *Broker) GetTotalDispatchedConnectorEventCount() uint64 {
	return atomic.LoadUint64(&b.totalDispatchedConnectorEventCount)
}

// runWorkqueueProcessor starts a goroutine to process events from the workqueue until
// signalled to stop on the given channel.
func (b *Broker) runWorkqueueProcessor(stopCh <-chan struct{}) {
	// Start the goroutine workqueue to process kubernetes events
	// The continuous processing of items in the workqueue will run
	// until signalled to stop.
	// The 'wait.Until' helper is used here to ensure the processing
	// of items in the workqueue continues until signalled to stop, even
	// if 'processNextItems()' returns false.
	go wait.Until(
		func() {
			for b.processNextItem() {
			}
		},
		time.Second,
		stopCh,
	)
}

// runProxyUpdateDispatcher runs the dispatcher responsible for batching
// proxy update events received in close proximity.
// It batches proxy update events with the use of 2 timers:
// 1. Sliding window timer that resets when a proxy update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the proxy update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runProxyUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether a proxy update event is pending
	// from being published on the pub-sub. A proxy update event will
	// be held for 'proxyUpdateSlidingWindow' duration to be able to
	// coalesce multiple proxy update events within that duration, before
	// it is dispatched on the pub-sub. The 'proxyUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'proxyUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering proxy update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many proxy update events within the 'proxyUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of proxy update events batched per dispatch

	var event proxyUpdateEvent
	for {
		select {
		case e, ok := <-b.proxyUpdateCh:
			if !ok {
				log.Warn().Msgf("Proxy update event chan closed, exiting dispatcher")
				return
			}
			event = e

			if !dispatchPending {
				// No proxy update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(proxyUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(proxyUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", event.msg.Kind)
			} else {
				// A proxy update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(proxyUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", event.msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.proxyUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
			metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount.Inc()
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.proxyUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
			metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount.Inc()
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-stopCh:
			log.Info().Msg("Proxy update dispatcher received stop signal, exiting")
			return
		}
	}
}

// runIngressUpdateDispatcher runs the dispatcher responsible for batching
// ingress update events received in close proximity.
// It batches ingress update events with the use of 2 timers:
// 1. Sliding window timer that resets when an ingress update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the ingress update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runIngressUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether an ingress update event is pending
	// from being published on the pub-sub. An ingress update event will
	// be held for 'ingressUpdateSlidingWindow' duration to be able to
	// coalesce multiple ingress update events within that duration, before
	// it is dispatched on the pub-sub. The 'ingressUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'ingressUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering ingress update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many ingresses update events within the 'ingressUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of ingress update events batched per dispatch

	var event ingressUpdateEvent
	for {
		select {
		case e, ok := <-b.ingressUpdateCh:
			if !ok {
				log.Warn().Msgf("Ingress update event chan closed, exiting dispatcher")
				return
			}
			event = e

			if !dispatchPending {
				// No ingress update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(ingressUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(ingressUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", event.msg.Kind)
			} else {
				// An ingress update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(ingressUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", event.msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.ingressUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedIngressEventCount, 1)
			metricsstore.DefaultMetricsStore.IngressBroadcastEventCount.Inc()
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.ingressUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedIngressEventCount, 1)
			metricsstore.DefaultMetricsStore.IngressBroadcastEventCount.Inc()
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-stopCh:
			log.Info().Msg("Ingress update dispatcher received stop signal, exiting")
			return
		}
	}
}

// runGatewayUpdateDispatcher runs the dispatcher responsible for batching
// gateway update events received in close proximity.
// It batches gateway update events with the use of 2 timers:
// 1. Sliding window timer that resets when a gateway update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the gateway update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runGatewayUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether a gateway update event is pending
	// from being published on the pub-sub. A gateway update event will
	// be held for 'gatewayUpdateSlidingWindow' duration to be able to
	// coalesce multiple gateway update events within that duration, before
	// it is dispatched on the pub-sub. The 'gatewayUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'gatewayUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering gateway update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many gateway update events within the 'gatewayUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of gateway update events batched per dispatch

	var event gatewayUpdateEvent
	for {
		select {
		case e, ok := <-b.gatewayUpdateCh:
			if !ok {
				log.Warn().Msgf("Gateway update event chan closed, exiting dispatcher")
				return
			}
			event = e

			if !dispatchPending {
				// No gateway update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(gatewayUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(gatewayUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", event.msg.Kind)
			} else {
				// A gateway update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(gatewayUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", event.msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.gatewayUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedGatewayEventCount, 1)
			metricsstore.DefaultMetricsStore.GatewayBroadcastEventCounter.Inc()
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.gatewayUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedGatewayEventCount, 1)
			metricsstore.DefaultMetricsStore.GatewayBroadcastEventCounter.Inc()
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-stopCh:
			log.Info().Msg("Proxy update dispatcher received stop signal, exiting")
			return
		}
	}
}

// runServiceUpdateDispatcher runs the dispatcher responsible for batching
// service update events received in close proximity.
// It batches service update events with the use of 2 timers:
// 1. Sliding window timer that resets when a service update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the service update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runServiceUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether a service update event is pending
	// from being published on the pub-sub. A service update event will
	// be held for 'serviceUpdateSlidingWindow' duration to be able to
	// coalesce multiple service update events within that duration, before
	// it is dispatched on the pub-sub. The 'serviceUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'serviceUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering service update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many service update events within the 'serviceUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of service update events batched per dispatch

	var event serviceUpdateEvent
	for {
		select {
		case e, ok := <-b.serviceUpdateCh:
			if !ok {
				log.Warn().Msgf("Service update event chan closed, exiting dispatcher")
				return
			}
			event = e

			if !dispatchPending {
				// No service update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(serviceUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(serviceUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", event.msg.Kind)
			} else {
				// A service update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(serviceUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", event.msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.serviceUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedServiceEventCount, 1)
			metricsstore.DefaultMetricsStore.ServiceBroadcastEventCounter.Inc()
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.serviceUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedServiceEventCount, 1)
			metricsstore.DefaultMetricsStore.ServiceBroadcastEventCounter.Inc()
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-stopCh:
			log.Info().Msg("Service update dispatcher received stop signal, exiting")
			return
		}
	}
}

// runConnectorUpdateDispatcher runs the dispatcher responsible for batching
// service update events received in close proximity.
// It batches connector update events with the use of 2 timers:
// 1. Sliding window timer that resets when a connector update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the connector update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runConnectorUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether a connector update event is pending
	// from being published on the pub-sub. A connector update event will
	// be held for 'ConnectorUpdateSlidingWindow' duration to be able to
	// coalesce multiple connector update events within that duration, before
	// it is dispatched on the pub-sub. The 'ConnectorUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'ConnectorUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering connector update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many connector update events within the 'ConnectorUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of connector update events batched per dispatch

	var event connectorUpdateEvent
	for {
		select {
		case e, ok := <-b.connectorUpdateCh:
			if !ok {
				log.Warn().Msgf("Connector update event chan closed, exiting dispatcher")
				return
			}
			event = e

			if !dispatchPending {
				// No connector update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(ConnectorUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(ConnectorUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", event.msg.Kind)
			} else {
				// A connector update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(ConnectorUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", event.msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.connectorUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedConnectorEventCount, 1)
			metricsstore.DefaultMetricsStore.ConnectorBroadcastEventCounter.Inc()
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.connectorUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedConnectorEventCount, 1)
			metricsstore.DefaultMetricsStore.ConnectorBroadcastEventCounter.Inc()
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-stopCh:
			log.Info().Msg("Connector update dispatcher received stop signal, exiting")
			return
		}
	}
}

// runXNetworkUpdateDispatcher runs the dispatcher responsible for batching
// xnetwork update events received in close proximity.
// It batches xnetwork update events with the use of 2 timers:
// 1. Sliding window timer that resets when a xnetwork update event is received
// 2. Max window timer that caps the max duration a sliding window can be reset to
// When either of the above timers expire, the xnetwork update event is published
// on the dedicated pub-sub instance.
func (b *Broker) runXNetworkUpdateDispatcher(stopCh <-chan struct{}) {
	// batchTimer and maxTimer are updated by the dispatcher routine
	// when events are processed and timeouts expire. They are initialized
	// with a large timeout (a decade) so they don't time out till an event
	// is received.
	noTimeout := 87600 * time.Hour // A decade
	slidingTimer := time.NewTimer(noTimeout)
	maxTimer := time.NewTimer(noTimeout)

	// dispatchPending indicates whether a xnetwork update event is pending
	// from being published on the pub-sub. A xnetwork update event will
	// be held for 'XNetworkUpdateSlidingWindow' duration to be able to
	// coalesce multiple xnetwork update events within that duration, before
	// it is dispatched on the pub-sub. The 'XNetworkUpdateSlidingWindow' duration
	// is a sliding window, which means each event received within a window
	// slides the window further ahead in time, up to a max of 'XNetworkUpdateMaxWindow'.
	//
	// This mechanism is necessary to avoid triggering xnetwork update pub-sub events in
	// a hot loop, which would otherwise result in CPU spikes on the controller.
	// We want to coalesce as many connector update events within the 'XNetworkUpdateMaxWindow'
	// duration.
	dispatchPending := false
	batchCount := 0 // number of xnetwork update events batched per dispatch

	var event xnetworkUpdateEvent
	for {
		select {
		case e, ok := <-b.xnetworkUpdateCh:
			if !ok {
				log.Warn().Msgf("XNetwork update event chan closed, exiting dispatcher")
				return
			}
			event = e

			if !dispatchPending {
				// No xnetwork update events are pending send on the pub-sub.
				// Reset the dispatch timers. The events will be dispatched
				// when either of the timers expire.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(XNetworkUpdateSlidingWindow)
				if !maxTimer.Stop() {
					<-maxTimer.C
				}
				maxTimer.Reset(XNetworkUpdateMaxWindow)
				dispatchPending = true
				batchCount++
				log.Trace().Msgf("Pending dispatch of msg kind %s", event.msg.Kind)
			} else {
				// A xnetwork update event is pending dispatch. Update the sliding window.
				if !slidingTimer.Stop() {
					<-slidingTimer.C
				}
				slidingTimer.Reset(XNetworkUpdateSlidingWindow)
				batchCount++
				log.Trace().Msgf("Reset sliding window for msg kind %s", event.msg.Kind)
			}

		case <-slidingTimer.C:
			slidingTimer.Reset(noTimeout) // 'slidingTimer' drained in this case statement
			// Stop and drain 'maxTimer' before Reset()
			if !maxTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-maxTimer.C
			}
			maxTimer.Reset(noTimeout)
			b.xnetworkUpdatePubSub.Pub(event.msg, event.topic)
			log.Trace().Msgf("Sliding window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-maxTimer.C:
			maxTimer.Reset(noTimeout) // 'maxTimer' drained in this case statement
			// Stop and drain 'slidingTimer' before Reset()
			if !slidingTimer.Stop() {
				// Drain channel. Refer to Reset() doc for more info.
				<-slidingTimer.C
			}
			slidingTimer.Reset(noTimeout)
			b.xnetworkUpdatePubSub.Pub(event.msg, event.topic)
			log.Trace().Msgf("Max window expired, msg kind %s, batch size %d", event.msg.Kind, batchCount)
			dispatchPending = false
			batchCount = 0

		case <-stopCh:
			log.Info().Msg("XNetwork update dispatcher received stop signal, exiting")
			return
		}
	}
}

// processEvent processes an event dispatched from the workqueue.
// It does the following:
// 1. If the event must update a proxy/ingress/gateway, it publishes a proxy/ingress/gateway update message
// 2. Processes other internal control plane events
// 3. Updates metrics associated with the event
func (b *Broker) processEvent(msg events.PubSubMessage) {
	log.Trace().Msgf("Processing msg kind: %s", msg.Kind)
	// Update proxies if applicable
	if event := getProxyUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update proxies", msg.Kind)
		atomic.AddUint64(&b.totalQProxyEventCount, 1)
		if event.topic != announcements.ProxyUpdate.String() {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more proxies.
			b.proxyUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedProxyEventCount, 1)
		} else {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.proxyUpdateCh <- *event
		}
	}

	// Update ingress if applicable
	if event := getIngressUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update ingress", msg.Kind)
		atomic.AddUint64(&b.totalQIngressEventCount, 1)
		if event.topic != announcements.IngressUpdate.String() {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more proxies.
			b.ingressUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedIngressEventCount, 1)
		} else {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.ingressUpdateCh <- *event
		}
	}

	// Update gateways if applicable
	if event := getGatewayUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update gateways", msg.Kind)
		atomic.AddUint64(&b.totalQGatewayEventCount, 1)
		if event.topic != announcements.GatewayUpdate.String() {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more proxies.
			b.gatewayUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedGatewayEventCount, 1)
		} else {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.gatewayUpdateCh <- *event
		}
	}

	// Update services if applicable
	if event := getServiceUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update services", msg.Kind)
		atomic.AddUint64(&b.totalQServiceEventCount, 1)
		if event.topic != announcements.ServiceUpdate.String() {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more services.
			b.serviceUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedServiceEventCount, 1)
		} else {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.serviceUpdateCh <- *event
		}
	}

	// Update connectors if applicable
	if event := getConnectorUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update connectors", msg.Kind)
		atomic.AddUint64(&b.totalQConnectorEventCount, 1)
		if event.topic != announcements.ConnectorUpdate.String() {
			// This is not a broadcast event, so it cannot be coalesced with
			// other events as the event is specific to one or more connectors.
			b.connectorUpdatePubSub.Pub(event.msg, event.topic)
			atomic.AddUint64(&b.totalDispatchedConnectorEventCount, 1)
		} else {
			// Pass the broadcast event to the dispatcher routine, that coalesces
			// multiple broadcasts received in close proximity.
			b.connectorUpdateCh <- *event
		}
	}

	// Update xnetworks if applicable
	if event := getXNetworkUpdateEvent(msg); event != nil {
		log.Trace().Msgf("Msg kind %s will update xnetworks", msg.Kind)
		if event.topic != announcements.XNetworkUpdate.String() {
			b.xnetworkUpdatePubSub.Pub(event.msg, event.topic)
		} else {
			b.xnetworkUpdateCh <- *event
		}
	}

	// Publish MCS event to other interested clients
	if event := getMCSUpdateEvent(msg); event != nil {
		log.Debug().Msgf("[MCS] Publishing event type: %s", msg.Kind)
		b.mcsEventPubSub.Pub(event.msg, event.topic)
	}

	// Publish event to other interested clients, e.g. log level changes, debug server on/off etc.
	b.kubeEventPubSub.Pub(msg, msg.Kind.String())

	// Update event metric
	updateMetric(msg)
}

// updateMetric updates metrics related to the event
func updateMetric(msg events.PubSubMessage) {
	switch msg.Kind {
	case announcements.NamespaceAdded:
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.Inc()
	case announcements.NamespaceDeleted:
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter.Dec()
	}
}

// Unsub unsubscribes the given channel from the PubSub instance
func (b *Broker) Unsub(pubSub *pubsub.PubSub, ch chan interface{}) {
	// Unsubscription should be performed from a different goroutine and
	// existing messages on the subscribed channel must be drained as noted
	// in https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L95.
	go pubSub.Unsub(ch)
	for range ch {
		// Drain channel until 'Unsub' results in a close on the subscribed channel
	}
}

// getProxyUpdateEvent returns a proxyUpdateEvent type indicating whether the given PubSubMessage should
// result in a Proxy configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a proxy update event.
func getProxyUpdateEvent(msg events.PubSubMessage) *proxyUpdateEvent {
	switch msg.Kind {
	case
		// Namepace event
		announcements.NamespaceUpdated:
		return namespaceUpdated(msg)
	case
		// Endpoint event
		announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated:
		if msg.NewObj != nil {
			if endpoints, ok := msg.NewObj.(*corev1.Endpoints); ok {
				if len(endpoints.Labels) > 0 {
					if _, exists := endpoints.Labels[constants.CloudSourcedServiceLabel]; exists {
						return nil
					}
				}
			}
		}
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
		}
	case
		// Service event
		announcements.ServiceAdded, announcements.ServiceDeleted, announcements.ServiceUpdated:
		if msg.NewObj != nil {
			if service, ok := msg.NewObj.(*corev1.Service); ok {
				if len(service.Labels) > 0 {
					if _, exists := service.Labels[constants.CloudSourcedServiceLabel]; exists {
						if !lru.MicroSvcMetaExists(service) {
							return &proxyUpdateEvent{
								msg:   msg,
								topic: announcements.ProxyUpdate.String(),
							}
						}
					}
				}
			}
		}
		if msg.OldObj != nil {
			if service, ok := msg.OldObj.(*corev1.Service); ok {
				if len(service.Labels) > 0 {
					if _, exists := service.Labels[constants.CloudSourcedServiceLabel]; exists {
						return &proxyUpdateEvent{
							msg:   msg,
							topic: announcements.ProxyUpdate.String(),
						}
					}
				}
			}
		}
		return nil
	case
		//
		// K8s native resource events
		//
		// k8s Ingress event
		announcements.IngressAdded, announcements.IngressDeleted, announcements.IngressUpdated,
		// k8s IngressClass event
		announcements.IngressClassAdded, announcements.IngressClassDeleted, announcements.IngressClassUpdated,
		//
		// FSM resource events
		//
		// Egress event
		announcements.EgressAdded, announcements.EgressDeleted, announcements.EgressUpdated,
		// EgressGateway event
		announcements.EgressGatewayAdded, announcements.EgressGatewayDeleted, announcements.EgressGatewayUpdated,
		// IngressBackend event
		announcements.IngressBackendAdded, announcements.IngressBackendDeleted, announcements.IngressBackendUpdated,
		// AccessControl event
		announcements.AccessControlAdded, announcements.AccessControlDeleted, announcements.AccessControlUpdated,
		// Isolation event
		announcements.IsolationPolicyAdded, announcements.IsolationPolicyDeleted, announcements.IsolationPolicyUpdated,
		// Retry event
		announcements.RetryPolicyAdded, announcements.RetryPolicyDeleted, announcements.RetryPolicyUpdated,
		// UpstreamTrafficSetting event
		announcements.UpstreamTrafficSettingAdded, announcements.UpstreamTrafficSettingDeleted, announcements.UpstreamTrafficSettingUpdated,
		//
		// SMI resource events
		//
		// SMI HTTPRouteGroup event
		announcements.RouteGroupAdded, announcements.RouteGroupDeleted, announcements.RouteGroupUpdated,
		// SMI TCPRoute event
		announcements.TCPRouteAdded, announcements.TCPRouteDeleted, announcements.TCPRouteUpdated,
		// SMI TrafficSplit event
		announcements.TrafficSplitAdded, announcements.TrafficSplitDeleted, announcements.TrafficSplitUpdated,
		// SMI TrafficTarget event
		announcements.TrafficTargetAdded, announcements.TrafficTargetDeleted, announcements.TrafficTargetUpdated,
		//
		// MultiCluster events
		//
		// ServiceImport event
		announcements.ServiceImportAdded, announcements.ServiceImportDeleted, announcements.ServiceImportUpdated,
		// ServiceExport event
		announcements.ServiceExportAdded, announcements.ServiceExportDeleted, announcements.ServiceExportUpdated,
		// GlobalTrafficPolicy event
		announcements.GlobalTrafficPolicyAdded, announcements.GlobalTrafficPolicyDeleted, announcements.GlobalTrafficPolicyUpdated,
		//
		// Plugin events
		//
		// Plugin event
		announcements.PluginAdded, announcements.PluginDeleted, announcements.PluginUpdated,
		// PluginChain event
		announcements.PluginChainAdded, announcements.PluginChainDeleted, announcements.PluginChainUpdated,
		// PluginService event
		announcements.PluginConfigAdded, announcements.PluginConfigDeleted, announcements.PluginConfigUpdated,
		//
		// Machine events
		//
		// VM event
		announcements.VirtualMachineAdded, announcements.VirtualMachineDeleted, announcements.VirtualMachineUpdated,
		//
		// Proxy events
		//
		announcements.ProxyUpdate:
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
		}

	case announcements.MeshConfigUpdated:
		return meshConfigUpdated(msg)

	case announcements.PodAdded, announcements.PodDeleted, announcements.PodUpdated:
		return podUpdated(msg)

	default:
		return nil
	}
}

func namespaceUpdated(msg events.PubSubMessage) *proxyUpdateEvent {
	prevExclusionList := ``
	newExclusionList := ``
	if ns, okPrevCast := msg.OldObj.(*corev1.Namespace); okPrevCast {
		if len(ns.Annotations) > 0 {
			prevExclusionList = ns.Annotations[constants.ServiceExclusionListAnnotation]
		}
	}
	if ns, okNewCast := msg.NewObj.(*corev1.Namespace); okNewCast {
		if len(ns.Annotations) > 0 {
			newExclusionList = ns.Annotations[constants.ServiceExclusionListAnnotation]
		}
	}
	if !strings.EqualFold(prevExclusionList, newExclusionList) {
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
		}
	}
	return nil
}

func meshConfigUpdated(msg events.PubSubMessage) *proxyUpdateEvent {
	prevMeshConfig, okPrevCast := msg.OldObj.(*configv1alpha3.MeshConfig)
	newMeshConfig, okNewCast := msg.NewObj.(*configv1alpha3.MeshConfig)
	if !okPrevCast || !okNewCast {
		log.Error().Msgf("Expected MeshConfig type, got previous=%T, new=%T", okPrevCast, okNewCast)
		return nil
	}
	prevSpec := prevMeshConfig.Spec
	newSpec := newMeshConfig.Spec
	// A proxy config update must only be triggered when a MeshConfig field that maps to a proxy config
	// changes.
	if prevSpec.Traffic.EnableEgress != newSpec.Traffic.EnableEgress ||
		prevSpec.Traffic.EnablePermissiveTrafficPolicyMode != newSpec.Traffic.EnablePermissiveTrafficPolicyMode ||
		prevSpec.Traffic.HTTP1PerRequestLoadBalancing != newSpec.Traffic.HTTP1PerRequestLoadBalancing ||
		prevSpec.Traffic.HTTP2PerRequestLoadBalancing != newSpec.Traffic.HTTP2PerRequestLoadBalancing ||
		prevSpec.Traffic.ServiceAccessMode != newSpec.Traffic.ServiceAccessMode ||
		prevSpec.Observability.Tracing != newSpec.Observability.Tracing ||
		prevSpec.Observability.RemoteLogging != newSpec.Observability.RemoteLogging ||
		prevSpec.Sidecar.LogLevel != newSpec.Sidecar.LogLevel ||
		prevSpec.Sidecar.CompressConfig != newSpec.Sidecar.CompressConfig ||
		prevSpec.Sidecar.SidecarTimeout != newSpec.Sidecar.SidecarTimeout ||
		!reflect.DeepEqual(prevSpec.Sidecar.LocalDNSProxy, newSpec.Sidecar.LocalDNSProxy) ||
		prevSpec.Traffic.InboundExternalAuthorization.Enable != newSpec.Traffic.InboundExternalAuthorization.Enable ||
		// Only trigger an update on InboundExternalAuthorization field changes if the new spec has the 'Enable' flag set to true.
		(newSpec.Traffic.InboundExternalAuthorization.Enable && (prevSpec.Traffic.InboundExternalAuthorization != newSpec.Traffic.InboundExternalAuthorization)) ||
		prevSpec.FeatureFlags != newSpec.FeatureFlags ||
		!reflect.DeepEqual(prevSpec.PluginChains, newSpec.PluginChains) ||
		!reflect.DeepEqual(prevSpec.Connector, newSpec.Connector) ||
		!reflect.DeepEqual(prevSpec.ClusterSet, newSpec.ClusterSet) {
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
		}
	}
	return nil
}

func podUpdated(msg events.PubSubMessage) *proxyUpdateEvent {
	// Only trigger a proxy update for proxies associated with this pod based on the proxy UUID
	prePod, okPreCast := msg.OldObj.(*corev1.Pod)
	newPod, okNewCast := msg.NewObj.(*corev1.Pod)

	if !okPreCast && !okNewCast {
		log.Error().Msgf("Expected *Pod type, got previous=%T, new=%T", okPreCast, okNewCast)
		return nil
	}

	if okPreCast && okNewCast {
		prevMetricAnnotation := prePod.Annotations[constants.PrometheusScrapeAnnotation]
		newMetricAnnotation := newPod.Annotations[constants.PrometheusScrapeAnnotation]
		if prevMetricAnnotation != newMetricAnnotation {
			proxyUUID := newPod.Labels[constants.SidecarUniqueIDLabelName]
			return &proxyUpdateEvent{
				msg:   msg,
				topic: GetPubSubTopicForProxyUUID(proxyUUID),
			}
		}
	}
	if okNewCast {
		if proxyUUID := newPod.Labels[constants.SidecarUniqueIDLabelName]; len(proxyUUID) > 0 {
			return &proxyUpdateEvent{
				msg:   msg,
				topic: announcements.ProxyUpdate.String(),
			}
		}
	}
	if okPreCast {
		if proxyUUID := prePod.Labels[constants.SidecarUniqueIDLabelName]; len(proxyUUID) > 0 {
			return &proxyUpdateEvent{
				msg:   msg,
				topic: announcements.ProxyUpdate.String(),
			}
		}
	}
	return nil
}

// getIngressUpdateEvent returns a ingressUpdateEvent type indicating whether the given PubSubMessage should
// result in an ingress configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in an ingress update event.
func getIngressUpdateEvent(msg events.PubSubMessage) *ingressUpdateEvent {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		//
		// Endpoint event
		announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated,
		// Service event
		announcements.ServiceAdded, announcements.ServiceDeleted, announcements.ServiceUpdated,
		// k8s Ingress event
		announcements.IngressAdded, announcements.IngressDeleted, announcements.IngressUpdated,
		// k8s IngressClass event
		announcements.IngressClassAdded, announcements.IngressClassDeleted, announcements.IngressClassUpdated,
		//
		// MultiCluster events
		//
		// ServiceImport event
		announcements.ServiceImportAdded, announcements.ServiceImportDeleted, announcements.ServiceImportUpdated:

		return &ingressUpdateEvent{
			msg:   msg,
			topic: announcements.IngressUpdate.String(),
		}
	default:
		return nil
	}
}

// getGatewayUpdateEvent returns a gatewayUpdateEvent type indicating whether the given PubSubMessage should
// result in a gateway configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a gateway update event.
func getGatewayUpdateEvent(msg events.PubSubMessage) *gatewayUpdateEvent {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		//
		// Endpoint event
		announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated,
		// EndpointSlices event
		announcements.EndpointSlicesAdded, announcements.EndpointSlicesDeleted, announcements.EndpointSlicesUpdated,
		// Service event
		announcements.ServiceAdded, announcements.ServiceDeleted, announcements.ServiceUpdated,
		// Secret event
		announcements.SecretAdded, announcements.SecretUpdated, announcements.SecretDeleted,
		// ConfigMap event
		announcements.ConfigMapAdded, announcements.ConfigMapUpdated, announcements.ConfigMapDeleted,
		// Isolation event
		announcements.IsolationPolicyAdded, announcements.IsolationPolicyDeleted, announcements.IsolationPolicyUpdated,

		//
		// GatewayAPI events
		//
		// Gateway event
		announcements.GatewayAPIGatewayAdded, announcements.GatewayAPIGatewayDeleted, announcements.GatewayAPIGatewayUpdated,
		// GatewayClass event
		announcements.GatewayAPIGatewayClassAdded, announcements.GatewayAPIGatewayClassDeleted, announcements.GatewayAPIGatewayClassUpdated,
		// HTTPRoute event
		announcements.GatewayAPIHTTPRouteAdded, announcements.GatewayAPIHTTPRouteDeleted, announcements.GatewayAPIHTTPRouteUpdated,
		// GRPCRoute event
		announcements.GatewayAPIGRPCRouteAdded, announcements.GatewayAPIGRPCRouteDeleted, announcements.GatewayAPIGRPCRouteUpdated,
		// TLSCRoute event
		announcements.GatewayAPITLSRouteAdded, announcements.GatewayAPITLSRouteDeleted, announcements.GatewayAPITLSRouteUpdated,
		// TCPRoute event
		announcements.GatewayAPITCPRouteAdded, announcements.GatewayAPITCPRouteDeleted, announcements.GatewayAPITCPRouteUpdated,
		// UDPRoute event
		announcements.GatewayAPIUDPRouteAdded, announcements.GatewayAPIUDPRouteDeleted, announcements.GatewayAPIUDPRouteUpdated,
		// ReferenceGrant event
		announcements.GatewayAPIReferenceGrantAdded, announcements.GatewayAPIReferenceGrantDeleted, announcements.GatewayAPIReferenceGrantUpdated,
		// RateLimit event
		announcements.RateLimitAdded, announcements.RateLimitDeleted, announcements.RateLimitUpdated,
		// CircuitBreaker event
		announcements.CircuitBreakerAdded, announcements.CircuitBreakerDeleted, announcements.CircuitBreakerUpdated,
		// HealthCheckPolicy event
		announcements.HealthCheckPolicyAdded, announcements.HealthCheckPolicyDeleted, announcements.HealthCheckPolicyUpdated,
		// FaultInjection event
		announcements.FaultInjectionAdded, announcements.FaultInjectionDeleted, announcements.FaultInjectionUpdated,
		// BackendTLSPolicy event
		announcements.BackendTLSPolicyAdded, announcements.BackendTLSPolicyDeleted, announcements.BackendTLSPolicyUpdated,
		// BackendLBPolicy event
		announcements.BackendLBPolicyAdded, announcements.BackendLBPolicyDeleted, announcements.BackendLBPolicyUpdated,
		// Filter event
		announcements.FilterAdded, announcements.FilterDeleted, announcements.FilterUpdated,
		// ListenerFilter event
		announcements.ListenerFilterAdded, announcements.ListenerFilterDeleted, announcements.ListenerFilterUpdated,
		// FilterDefinition event
		announcements.FilterDefinitionAdded, announcements.FilterDefinitionDeleted, announcements.FilterDefinitionUpdated,
		// FilterConfig event
		announcements.FilterConfigAdded, announcements.FilterConfigDeleted, announcements.FilterConfigUpdated,
		// HTTPLog event
		announcements.GatewayHTTPLogAdded, announcements.GatewayHTTPLogDeleted, announcements.GatewayHTTPLogUpdated,
		// Metrics event
		announcements.GatewayMetricsAdded, announcements.GatewayMetricsDeleted, announcements.GatewayMetricsUpdated,
		// Zipkin event
		announcements.GatewayZipkinAdded, announcements.GatewayZipkinDeleted, announcements.GatewayZipkinUpdated,
		// ProxyTag event
		announcements.GatewayProxyTagAdded, announcements.GatewayProxyTagDeleted, announcements.GatewayProxyTagUpdated,

		//
		// MultiCluster events
		//
		// ServiceImport event
		announcements.ServiceImportAdded, announcements.ServiceImportDeleted, announcements.ServiceImportUpdated:

		return &gatewayUpdateEvent{
			msg:   msg,
			topic: announcements.GatewayUpdate.String(),
		}
	//case announcements.MeshConfigUpdated:
	//	return gatewayInterestedConfigChanged(msg)
	default:
		return nil
	}
}

//func gatewayInterestedConfigChanged(msg events.PubSubMessage) *gatewayUpdateEvent {
//	prevMeshConfig, okPrevCast := msg.OldObj.(*configv1alpha3.MeshConfig)
//	newMeshConfig, okNewCast := msg.NewObj.(*configv1alpha3.MeshConfig)
//	if !okPrevCast || !okNewCast {
//		log.Error().Msgf("Expected MeshConfig type, got previous=%T, new=%T", okPrevCast, okNewCast)
//		return nil
//	}
//	prevSpec := prevMeshConfig.Spec
//	newSpec := newMeshConfig.Spec
//
//	if prevSpec.GatewayAPI.FGWLogLevel != newSpec.GatewayAPI.FGWLogLevel ||
//		prevSpec.FeatureFlags.EnableGatewayAgentService != newSpec.FeatureFlags.EnableGatewayAgentService ||
//		prevSpec.GatewayAPI.StripAnyHostPort != newSpec.GatewayAPI.StripAnyHostPort ||
//		prevSpec.GatewayAPI.ProxyPreserveHost != newSpec.GatewayAPI.ProxyPreserveHost ||
//		prevSpec.GatewayAPI.SSLPassthroughUpstreamPort != newSpec.GatewayAPI.SSLPassthroughUpstreamPort ||
//		prevSpec.GatewayAPI.ProxyTag.DstHostHeader != newSpec.GatewayAPI.ProxyTag.DstHostHeader ||
//		prevSpec.GatewayAPI.ProxyTag.SrcHostHeader != newSpec.GatewayAPI.ProxyTag.SrcHostHeader ||
//		prevSpec.GatewayAPI.HTTP1PerRequestLoadBalancing != newSpec.GatewayAPI.HTTP1PerRequestLoadBalancing ||
//		prevSpec.GatewayAPI.HTTP2PerRequestLoadBalancing != newSpec.GatewayAPI.HTTP2PerRequestLoadBalancing {
//		return &gatewayUpdateEvent{
//			msg:   msg,
//			topic: announcements.GatewayUpdate.String(),
//		}
//	}
//
//	return nil
//}

// getMCSUpdateEvent returns a mcsUpdateEvent type indicating whether the given PubSubMessage should
// result in a gateway configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a gateway update event.
func getMCSUpdateEvent(msg events.PubSubMessage) *mcsUpdateEvent {
	switch msg.Kind {
	case
		//
		// MultiCluster events
		//
		// ServiceImport event
		//announcements.ServiceImportAdded, announcements.ServiceImportDeleted, announcements.ServiceImportUpdated,
		// ServiceExport event
		//announcements.ServiceExportAdded, announcements.ServiceExportDeleted, announcements.ServiceExportUpdated,
		// GlobalTrafficPolicy event
		//announcements.GlobalTrafficPolicyAdded, announcements.GlobalTrafficPolicyUpdated, announcements.GlobalTrafficPolicyDeleted,
		// MultiCluster ServiceExport event
		announcements.MultiClusterServiceExportCreated, announcements.MultiClusterServiceExportDeleted,
		announcements.MultiClusterServiceExportAccepted, announcements.MultiClusterServiceExportRejected:

		return &mcsUpdateEvent{
			msg:   msg,
			topic: msg.Kind.String(),
		}
	default:
		return nil
	}
}

// getServiceUpdateEvent returns a serviceUpdateEvent type indicating whether the given PubSubMessage should
// result in a service configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a service update event.
func getServiceUpdateEvent(msg events.PubSubMessage) *serviceUpdateEvent {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		//
		// Service event
		announcements.ServiceUpdate:

		return &serviceUpdateEvent{
			msg:   msg,
			topic: announcements.ServiceUpdate.String(),
		}
	default:
		return nil
	}
}

// getConnectorUpdateEvent returns a connectorUpdateEvent type indicating whether the given PubSubMessage should
// result in a connector configuration update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a connector update event.
func getConnectorUpdateEvent(msg events.PubSubMessage) *connectorUpdateEvent {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		//
		// Connector event
		announcements.ConsulConnectorAdded, announcements.ConsulConnectorUpdated, announcements.ConsulConnectorDeleted,
		announcements.EurekaConnectorAdded, announcements.EurekaConnectorUpdated, announcements.EurekaConnectorDeleted,
		announcements.NacosConnectorAdded, announcements.NacosConnectorUpdated, announcements.NacosConnectorDeleted,
		announcements.MachineConnectorAdded, announcements.MachineConnectorUpdated, announcements.MachineConnectorDeleted,
		announcements.GatewayConnectorAdded, announcements.GatewayConnectorUpdated, announcements.GatewayConnectorDeleted,
		announcements.ConnectorUpdate:

		return &connectorUpdateEvent{
			msg:   msg,
			topic: announcements.ConnectorUpdate.String(),
		}
	default:
		return nil
	}
}

// getXNetworkUpdateEvent returns a xnetworkUpdateEvent type indicating whether the given PubSubMessage should
// result in a xnetwork policy update on an appropriate topic. Nil is returned if the PubSubMessage
// does not result in a xnetwork policy update event.
func getXNetworkUpdateEvent(msg events.PubSubMessage) *xnetworkUpdateEvent {
	switch msg.Kind {
	case
		//
		// K8s native resource events
		announcements.ServiceAdded, announcements.ServiceUpdated, announcements.ServiceDeleted,
		announcements.EndpointAdded, announcements.EndpointUpdated, announcements.EndpointDeleted,

		//
		// XNetwork event
		announcements.XAccessControlAdded, announcements.XAccessControlUpdated, announcements.XAccessControlDeleted,
		announcements.XNetworkUpdate:

		return &xnetworkUpdateEvent{
			msg:   msg,
			topic: announcements.XNetworkUpdate.String(),
		}
	default:
		return nil
	}
}

// GetPubSubTopicForProxyUUID returns the topic on which PubSubMessages specific to a proxy UUID are published
func GetPubSubTopicForProxyUUID(uuid string) string {
	return fmt.Sprintf("proxy:%s", uuid)
}
