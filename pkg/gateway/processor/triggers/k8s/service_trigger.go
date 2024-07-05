package triggers

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// ServicesTrigger is responsible for processing Service objects
type ServicesTrigger struct{}

// Insert adds the Service object to the processor and returns true if the processor was modified
func (p *ServicesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableService(client.ObjectKeyFromObject(svc))
}

// Delete removes the Service object from the processor and returns true if the processor was modified
func (p *ServicesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableService(client.ObjectKeyFromObject(svc))
}
