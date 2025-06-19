package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// ConcurrencyLimitTrigger is a processor for ConcurrencyLimit objects
type ConcurrencyLimitTrigger struct{}

// Insert adds a ConcurrencyLimit object to the processor and returns true if the processor is changed
func (p *ConcurrencyLimitTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.ConcurrencyLimit)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}

// Delete removes a ConcurrencyLimit object from the processor and returns true if the processor is changed
func (p *ConcurrencyLimitTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.ConcurrencyLimit)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}
