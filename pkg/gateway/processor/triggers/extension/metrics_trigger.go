package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// MetricsTrigger is a processor for Metrics objects
type MetricsTrigger struct{}

// Insert adds a Metrics object to the processor and returns true if the processor is changed
func (p *MetricsTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.Metrics)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}

// Delete removes a Metrics object from the processor and returns true if the processor is changed
func (p *MetricsTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	config, ok := obj.(*extv1alpha1.Metrics)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsFilterConfigReferred(config.Kind, client.ObjectKeyFromObject(config))
}
