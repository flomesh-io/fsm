package gateway

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// TLSRoutesTrigger is responsible for processing TLSRoute objects
type TLSRoutesTrigger struct{}

// Insert adds a TLSRoute to the processor and returns true if the route is effective
func (p *TLSRoutesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1alpha2.TLSRoute)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}

// Delete removes a TLSRoute from the processor and returns true if the route was found
func (p *TLSRoutesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	route, ok := obj.(*gwv1alpha2.TLSRoute)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsEffectiveRoute(route.Spec.ParentRefs)
}
