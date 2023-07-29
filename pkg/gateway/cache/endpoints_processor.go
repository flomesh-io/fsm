package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	corev1 "k8s.io/api/core/v1"
)

// EndpointsProcessor is responsible for processing Endpoints objects
type EndpointsProcessor struct {
}

// Insert adds the Endpoints object to the cache and returns true if the cache was modified
func (p *EndpointsProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)
	cache.endpoints[key] = struct{}{}

	return cache.isRoutableService(key)
}

// Delete removes the Endpoints object from the cache and returns true if the cache was modified
func (p *EndpointsProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(ep)
	_, found := cache.endpoints[key]
	delete(cache.endpoints, key)

	return found
}
