package triggers

import (
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// EndpointSlicesTrigger is responsible for processing EndpointSlices
type EndpointSlicesTrigger struct{}

// Insert adds the EndpointSlice object to the processor and returns true if the processor was modified
func (p *EndpointSlicesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	eps, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	if len(eps.Labels) == 0 {
		return false
	}

	svcName := eps.Labels[discoveryv1.LabelServiceName]
	if len(svcName) == 0 {
		return false
	}

	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: svcName}

	return processor.IsRoutableService(svcKey)
}

// Delete removes the EndpointSlice object from the processor and returns true if the processor was modified
func (p *EndpointSlicesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	eps, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	owner := metav1.GetControllerOf(eps)
	if owner == nil {
		return false
	}

	svcKey := client.ObjectKey{Namespace: eps.Namespace, Name: owner.Name}

	return processor.IsRoutableService(svcKey)
}
