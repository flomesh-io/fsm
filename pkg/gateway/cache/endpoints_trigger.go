package cache

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// EndpointsTrigger is responsible for processing Endpoints objects
type EndpointsTrigger struct{}

// Insert adds the Endpoints object to the cache and returns true if the cache was modified
func (p *EndpointsTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)

	if cache.useEndpointSlices {
		return cache.isHeadlessServiceWithoutSelector(key) && cache.isRoutableService(key)
	} else {
		return cache.isRoutableService(key)
	}
}

// Delete removes the Endpoints object from the cache and returns true if the cache was modified
func (p *EndpointsTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)

	if cache.useEndpointSlices {
		return cache.isHeadlessServiceWithoutSelector(key) && cache.isRoutableService(key)
	} else {
		return cache.isRoutableService(key)
	}
}
