package triggers

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// EndpointsTrigger is responsible for processing Endpoints objects
type EndpointsTrigger struct{}

// Insert adds the Endpoints object to the cache and returns true if the cache was modified
func (p *EndpointsTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)

	if cache.UseEndpointSlices() {
		return cache.IsHeadlessService(key) && cache.IsRoutableService(key)
	} else {
		return cache.IsRoutableService(key)
	}
}

// Delete removes the Endpoints object from the cache and returns true if the cache was modified
func (p *EndpointsTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)

	if cache.UseEndpointSlices() {
		return cache.IsHeadlessService(key) && cache.IsRoutableService(key)
	} else {
		return cache.IsRoutableService(key)
	}
}
