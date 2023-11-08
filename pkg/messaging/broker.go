package messaging

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cskr/pubsub"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
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
)

// NewBroker returns a new message broker instance and starts the internal goroutine
// to process events added to the workqueue.
func NewBroker(stopCh <-chan struct{}) *Broker {
	b := &Broker{
		queue:               workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		proxyUpdatePubSub:   pubsub.New(10240),
		proxyUpdateCh:       make(chan proxyUpdateEvent),
		ingressUpdatePubSub: pubsub.New(10240),
		ingressUpdateCh:     make(chan ingressUpdateEvent),
		gatewayUpdatePubSub: pubsub.New(10240),
		gatewayUpdateCh:     make(chan gatewayUpdateEvent),
		kubeEventPubSub:     pubsub.New(10240),
		certPubSub:          pubsub.New(10240),
		mcsEventPubSub:      pubsub.New(10240),
		//mcsUpdateCh:         make(chan mcsUpdateEvent),
	}

	go b.runWorkqueueProcessor(stopCh)
	go b.runProxyUpdateDispatcher(stopCh)
	go b.runIngressUpdateDispatcher(stopCh)
	go b.runGatewayUpdateDispatcher(stopCh)
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
		//
		// K8s native resource events
		//
		// Endpoint event
		announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated,
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
		// Proxy events
		//
		announcements.ProxyUpdate:
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
		}

	case announcements.MeshConfigUpdated:
		return meshConfigUpdated(msg)

	case announcements.PodUpdated:
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
		prevSpec.Sidecar.SidecarTimeout != newSpec.Sidecar.SidecarTimeout ||
		prevSpec.Sidecar.LocalDNSProxy != newSpec.Sidecar.LocalDNSProxy ||
		prevSpec.Traffic.InboundExternalAuthorization.Enable != newSpec.Traffic.InboundExternalAuthorization.Enable ||
		// Only trigger an update on InboundExternalAuthorization field changes if the new spec has the 'Enable' flag set to true.
		(newSpec.Traffic.InboundExternalAuthorization.Enable && (prevSpec.Traffic.InboundExternalAuthorization != newSpec.Traffic.InboundExternalAuthorization)) ||
		prevSpec.FeatureFlags != newSpec.FeatureFlags ||
		!reflect.DeepEqual(prevSpec.PluginChains, newSpec.PluginChains) ||
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
	prevPod, okPrevCast := msg.OldObj.(*corev1.Pod)
	newPod, okNewCast := msg.NewObj.(*corev1.Pod)
	if !okPrevCast || !okNewCast {
		log.Error().Msgf("Expected *Pod type, got previous=%T, new=%T", okPrevCast, okNewCast)
		return nil
	}
	prevMetricAnnotation := prevPod.Annotations[constants.PrometheusScrapeAnnotation]
	newMetricAnnotation := newPod.Annotations[constants.PrometheusScrapeAnnotation]
	if prevMetricAnnotation != newMetricAnnotation {
		proxyUUID := newPod.Labels[constants.SidecarUniqueIDLabelName]
		return &proxyUpdateEvent{
			msg:   msg,
			topic: GetPubSubTopicForProxyUUID(proxyUUID),
		}
	} else if proxyUUID := newPod.Labels[constants.SidecarUniqueIDLabelName]; len(proxyUUID) > 0 {
		return &proxyUpdateEvent{
			msg:   msg,
			topic: announcements.ProxyUpdate.String(),
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
		//announcements.EndpointAdded, announcements.EndpointDeleted, announcements.EndpointUpdated,
		// EndpointSlices event
		announcements.EndpointSlicesAdded, announcements.EndpointSlicesDeleted, announcements.EndpointSlicesUpdated,
		// Service event
		announcements.ServiceAdded, announcements.ServiceDeleted, announcements.ServiceUpdated,
		// Secret event
		announcements.SecretAdded, announcements.SecretUpdated, announcements.SecretDeleted,

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
		// RateLimitPolicy event
		announcements.RateLimitPolicyAdded, announcements.RateLimitPolicyDeleted, announcements.RateLimitPolicyUpdated,
		// SessionStickyPolicy event
		announcements.SessionStickyPolicyAdded, announcements.SessionStickyPolicyDeleted, announcements.SessionStickyPolicyUpdated,
		// LoadBalancerPolicy event
		announcements.LoadBalancerPolicyAdded, announcements.LoadBalancerPolicyDeleted, announcements.LoadBalancerPolicyUpdated,
		// CircuitBreakingPolicy event
		announcements.CircuitBreakingPolicyAdded, announcements.CircuitBreakingPolicyDeleted, announcements.CircuitBreakingPolicyUpdated,
		// AccessControlPolicy event
		announcements.AccessControlPolicyAdded, announcements.AccessControlPolicyDeleted, announcements.AccessControlPolicyUpdated,
		// HealthCheckPolicy event
		announcements.HealthCheckPolicyAdded, announcements.HealthCheckPolicyDeleted, announcements.HealthCheckPolicyUpdated,
		// FaultInjectionPolicy event
		announcements.FaultInjectionPolicyAdded, announcements.FaultInjectionPolicyDeleted, announcements.FaultInjectionPolicyUpdated,
		// UpstreamTLSPolicy event
		announcements.UpstreamTLSPolicyAdded, announcements.UpstreamTLSPolicyDeleted, announcements.UpstreamTLSPolicyUpdated,
		// RetryPolicy event
		announcements.RetryPolicyAttachmentAdded, announcements.RetryPolicyAttachmentDeleted, announcements.RetryPolicyAttachmentUpdated,

		//
		// MultiCluster events
		//
		// ServiceImport event
		announcements.ServiceImportAdded, announcements.ServiceImportDeleted, announcements.ServiceImportUpdated:

		return &gatewayUpdateEvent{
			msg:   msg,
			topic: announcements.GatewayUpdate.String(),
		}
	case announcements.MeshConfigUpdated:
		return gatewayInterestedConfigChanged(msg)
	default:
		return nil
	}
}

func gatewayInterestedConfigChanged(msg events.PubSubMessage) *gatewayUpdateEvent {
	prevMeshConfig, okPrevCast := msg.OldObj.(*configv1alpha3.MeshConfig)
	newMeshConfig, okNewCast := msg.NewObj.(*configv1alpha3.MeshConfig)
	if !okPrevCast || !okNewCast {
		log.Error().Msgf("Expected MeshConfig type, got previous=%T, new=%T", okPrevCast, okNewCast)
		return nil
	}
	prevSpec := prevMeshConfig.Spec
	newSpec := newMeshConfig.Spec

	if prevSpec.GatewayAPI.FGWLogLevel != newSpec.GatewayAPI.FGWLogLevel ||
		prevSpec.FeatureFlags.EnableGatewayAgentService != newSpec.FeatureFlags.EnableGatewayAgentService ||
		prevSpec.GatewayAPI.StripAnyHostPort != newSpec.GatewayAPI.StripAnyHostPort ||
		prevSpec.GatewayAPI.SSLPassthroughUpstreamPort != newSpec.GatewayAPI.SSLPassthroughUpstreamPort ||
		prevSpec.GatewayAPI.HTTP1PerRequestLoadBalancing != newSpec.GatewayAPI.HTTP1PerRequestLoadBalancing ||
		prevSpec.GatewayAPI.HTTP2PerRequestLoadBalancing != newSpec.GatewayAPI.HTTP2PerRequestLoadBalancing {
		return &gatewayUpdateEvent{
			msg:   msg,
			topic: announcements.GatewayUpdate.String(),
		}
	}

	return nil
}

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

// GetPubSubTopicForProxyUUID returns the topic on which PubSubMessages specific to a proxy UUID are published
func GetPubSubTopicForProxyUUID(uuid string) string {
	return fmt.Sprintf("proxy:%s", uuid)
}
