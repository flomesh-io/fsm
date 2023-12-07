package cache

import (
	"sync"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// EndpointsTrigger is responsible for processing Endpoints objects
type EndpointsTrigger struct {
	mu sync.Mutex
}

// Insert adds the Endpoints object to the cache and returns true if the cache was modified
func (p *EndpointsTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(ep)
	cache.endpoints[key] = struct{}{}

	return cache.isRoutableService(key)
}

// Delete removes the Endpoints object from the cache and returns true if the cache was modified
func (p *EndpointsTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(ep)
	_, found := cache.endpoints[key]
	delete(cache.endpoints, key)

	return found
}
