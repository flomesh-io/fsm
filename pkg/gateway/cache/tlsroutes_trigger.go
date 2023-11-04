package cache

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// TLSRoutesTrigger is responsible for processing TLSRoute objects
type TLSRoutesTrigger struct {
}

// Insert adds a TLSRoute to the cache and returns true if the route is effective
func (p *TLSRoutesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TLSRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.tlsroutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a TLSRoute from the cache and returns true if the route was found
func (p *TLSRoutesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TLSRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(route)
	_, found := cache.tlsroutes[key]
	delete(cache.tlsroutes, key)

	return found
}
