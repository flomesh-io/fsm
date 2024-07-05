package triggers

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// EndpointsTrigger is responsible for processing Endpoints objects
type EndpointsTrigger struct{}

// Insert adds the Endpoints object to the processor and returns true if the processor was modified
func (p *EndpointsTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := client.ObjectKeyFromObject(ep)

	if processor.UseEndpointSlices() {
		return processor.IsHeadlessService(key) && processor.IsRoutableService(key)
	} else {
		return processor.IsRoutableService(key)
	}
}

// Delete removes the Endpoints object from the processor and returns true if the processor was modified
func (p *EndpointsTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := client.ObjectKeyFromObject(ep)

	if processor.UseEndpointSlices() {
		return processor.IsHeadlessService(key) && processor.IsRoutableService(key)
	} else {
		return processor.IsRoutableService(key)
	}
}
