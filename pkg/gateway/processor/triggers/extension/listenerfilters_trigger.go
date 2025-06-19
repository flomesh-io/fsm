package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// ListenerFilterTrigger is a processor for ListenerFilter objects
type ListenerFilterTrigger struct{}

// Insert adds a ListenerFilter object to the processor and returns true if the processor is changed
func (p *ListenerFilterTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	filter, ok := obj.(*extv1alpha1.ListenerFilter)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsListenerFilterReferred(client.ObjectKeyFromObject(filter))
}

// Delete removes a ListenerFilter object from the processor and returns true if the processor is changed
func (p *ListenerFilterTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	filter, ok := obj.(*extv1alpha1.ListenerFilter)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return processor.IsListenerFilterReferred(client.ObjectKeyFromObject(filter))
}
