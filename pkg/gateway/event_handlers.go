package gateway

import (
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetEventHandlerFuncs()
type observeFilter func(obj interface{}) bool

// GetEventHandlerFuncs returns the ResourceEventHandlerFuncs object used to receive events when a k8s
// object is added/updated/deleted.
func GetEventHandlerFuncs(shouldObserveUpsert, shouldObserveDelete observeFilter, eventTypes k8s.EventTypes, msgBroker *messaging.Broker) cache.ResourceEventHandlerFuncs {
	if shouldObserveUpsert == nil {
		shouldObserveUpsert = func(obj interface{}) bool { return true }
	}

	if shouldObserveDelete == nil {
		shouldObserveDelete = func(obj interface{}) bool { return true }
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserveUpsert(obj) {
				return
			}
			logResourceEvent(log, eventTypes.Add, obj)
			//ns := getNamespace(obj)
			//metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Add.String(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   eventTypes.Add,
				NewObj: obj,
				OldObj: nil,
			})
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserveUpsert(newObj) {
				return
			}
			logResourceEvent(log, eventTypes.Update, newObj)
			//ns := getNamespace(newObj)
			//metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Update.String(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   eventTypes.Update,
				NewObj: newObj,
				OldObj: oldObj,
			})
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserveDelete(obj) {
				return
			}
			logResourceEvent(log, eventTypes.Delete, obj)
			//ns := getNamespace(obj)
			//metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Delete.String(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   eventTypes.Delete,
				NewObj: nil,
				OldObj: obj,
			})
		},
	}
}

//func getNamespace(obj interface{}) string {
//	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
//}

func logResourceEvent(parent zerolog.Logger, event announcements.Kind, obj interface{}) {
	log := parent.With().Str("event", event.String()).Logger()
	o, err := meta.Accessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("error parsing object, ignoring")
		return
	}
	name := o.GetName()
	if o.GetNamespace() != "" {
		name = o.GetNamespace() + "/" + name
	}
	log.Debug().Str("resource_name", name).Msg("received kubernetes resource event")
}
