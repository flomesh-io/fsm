package cache

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// TCPRoutesTrigger is responsible for processing TCPRoute objects
type TCPRoutesTrigger struct{}

// Insert adds a TCPRoute to the cache and returns true if the route is effective
func (p *TCPRoutesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a TCPRoute from the cache and returns true if the route was found
func (p *TCPRoutesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}
