package cache

import (
	"github.com/flomesh-io/fsm/pkg/gateway/utils"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// TCPRoutesProcessor is responsible for processing TCPRoute objects
type TCPRoutesProcessor struct {
}

// Insert adds a TCPRoute to the cache and returns true if the route is effective
func (p *TCPRoutesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.tcproutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a TCPRoute from the cache and returns true if the route was found
func (p *TCPRoutesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(route)
	_, found := cache.tcproutes[key]
	delete(cache.tcproutes, key)

	return found
}
