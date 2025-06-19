package gateway

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// HTTPRoutesTrigger is responsible for processing HTTPRoute objects
type HTTPRoutesTrigger struct{}

// Insert adds a HTTPRoute to the processor and returns true if the route is effective
func (p *HTTPRoutesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1.HTTPRoute)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a HTTPRoute from the processor and returns true if the route was found
func (p *HTTPRoutesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1.HTTPRoute)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}
