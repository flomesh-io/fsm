package gateway

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// GRPCRoutesTrigger is responsible for processing GRPCRoute objects
type GRPCRoutesTrigger struct{}

// Insert adds a GRPCRoute to the cache and returns true if the route is effective
func (p *GRPCRoutesTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	route, ok := obj.(*gwv1.GRPCRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a GRPCRoute from the cache and returns true if the route was found
func (p *GRPCRoutesTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	route, ok := obj.(*gwv1.GRPCRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsEffectiveRoute(route.Spec.ParentRefs)
}
