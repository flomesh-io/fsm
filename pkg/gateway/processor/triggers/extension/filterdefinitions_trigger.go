package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// FilterDefinitionTrigger is a processor for FilterDefinition objects
type FilterDefinitionTrigger struct{}

// Insert adds a FilterDefinition object to the processor and returns true if the processor is changed
func (p *FilterDefinitionTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	definition, ok := obj.(*extv1alpha1.FilterDefinition)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterDefinitionReferred(client.ObjectKeyFromObject(definition))
}

// Delete removes a FilterDefinition object from the processor and returns true if the processor is changed
func (p *FilterDefinitionTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	definition, ok := obj.(*extv1alpha1.FilterDefinition)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterDefinitionReferred(client.ObjectKeyFromObject(definition))
}
