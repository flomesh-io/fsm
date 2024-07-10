package gateway

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// TCPRoutesTrigger is responsible for processing TCPRoute objects
type TCPRoutesTrigger struct{}

// Insert adds a TCPRoute to the processor and returns true if the route is effective
func (p *TCPRoutesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a TCPRoute from the processor and returns true if the route was found
func (p *TCPRoutesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1alpha2.TCPRoute)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}
