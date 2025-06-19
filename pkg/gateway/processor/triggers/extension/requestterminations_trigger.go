package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// RequestTerminationTrigger is a processor for RequestTermination objects
type RequestTerminationTrigger struct{}

// Insert adds a RequestTermination object to the processor and returns true if the processor is changed
func (p *RequestTerminationTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.RequestTermination)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}

// Delete removes a RequestTermination object from the processor and returns true if the processor is changed
func (p *RequestTerminationTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.RequestTermination)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}
