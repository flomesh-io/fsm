package triggers

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// ConfigMapTrigger is a processor for ConfigMap objects
type ConfigMapTrigger struct{}

// Insert adds a ConfigMap object to the processor and returns true if the processor is changed
func (p *ConfigMapTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsConfigMapReferred(client.ObjectKeyFromObject(cm))
}

// Delete removes a ConfigMap object from the processor and returns true if the processor is changed
func (p *ConfigMapTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsConfigMapReferred(client.ObjectKeyFromObject(cm))
}
