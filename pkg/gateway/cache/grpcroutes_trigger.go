package cache

import gwv1 "sigs.k8s.io/gateway-api/apis/v1"

// GRPCRoutesTrigger is responsible for processing GRPCRoute objects
type GRPCRoutesTrigger struct{}

// Insert adds a GRPCRoute to the cache and returns true if the route is effective
func (p *GRPCRoutesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1.GRPCRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//cache.grpcroutes[utils.ObjectKey(route)] = struct{}{}

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a GRPCRoute from the cache and returns true if the route was found
func (p *GRPCRoutesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	route, ok := obj.(*gwv1.GRPCRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}
	//
	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//key := utils.ObjectKey(route)
	//_, found := cache.grpcroutes[key]
	//delete(cache.grpcroutes, key)
	//
	//return found

	return cache.isEffectiveRoute(route.Spec.ParentRefs)
}
