package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// CircuitBreakerTrigger is a processor for CircuitBreaker objects
type CircuitBreakerTrigger struct{}

// Insert adds a CircuitBreaker object to the processor and returns true if the processor is changed
func (p *CircuitBreakerTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.CircuitBreaker)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}

// Delete removes a CircuitBreaker object from the processor and returns true if the processor is changed
func (p *CircuitBreakerTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.CircuitBreaker)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}
