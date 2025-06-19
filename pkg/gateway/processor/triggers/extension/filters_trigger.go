package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// FilterTrigger is a processor for Filter objects
type FilterTrigger struct{}

// Insert adds a Filter object to the processor and returns true if the processor is changed
func (p *FilterTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	filter, ok := obj.(*extv1alpha1.Filter)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterReferred(client.ObjectKeyFromObject(filter))
}

// Delete removes a Filter object from the processor and returns true if the processor is changed
func (p *FilterTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	filter, ok := obj.(*extv1alpha1.Filter)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterReferred(client.ObjectKeyFromObject(filter))
}
