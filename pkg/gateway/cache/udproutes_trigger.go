package cache

import (
	"sync"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// UDPRoutesTrigger is responsible for processing UDPRoute objects
type UDPRoutesTrigger struct {
	mu sync.Mutex
}

// Insert adds a UDPRoute to the cache and returns true if the route is effective
func (p *UDPRoutesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.UDPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cache.udproutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a UDPRoute from the cache and returns true if the route was found
func (p *UDPRoutesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.UDPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(route)
	_, found := cache.udproutes[key]
	delete(cache.udproutes, key)

	return found
}
