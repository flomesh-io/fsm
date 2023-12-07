package cache

import (
	"sync"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ServicesTrigger is responsible for processing Service objects
type ServicesTrigger struct {
	mu sync.Mutex
}

// Insert adds the Service object to the cache and returns true if the cache was modified
func (p *ServicesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(svc)
	cache.services[key] = struct{}{}

	return cache.isRoutableService(key)
}

// Delete removes the Service object from the cache and returns true if the cache was modified
func (p *ServicesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(svc)
	_, found := cache.services[key]
	delete(cache.services, key)

	return found
}
