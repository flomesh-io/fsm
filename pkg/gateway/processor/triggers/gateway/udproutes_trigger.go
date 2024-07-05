package gateway

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// UDPRoutesTrigger is responsible for processing UDPRoute objects
type UDPRoutesTrigger struct{}

// Insert adds a UDPRoute to the processor and returns true if the route is effective
func (p *UDPRoutesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1alpha2.UDPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a UDPRoute from the processor and returns true if the route was found
func (p *UDPRoutesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1alpha2.UDPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}
