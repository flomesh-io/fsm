package cache

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// UDPRoutesTrigger is responsible for processing UDPRoute objects
type UDPRoutesTrigger struct{}

// Insert adds a UDPRoute to the cache and returns true if the route is effective
func (p *UDPRoutesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.UDPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a UDPRoute from the cache and returns true if the route was found
func (p *UDPRoutesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1alpha2.UDPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}
