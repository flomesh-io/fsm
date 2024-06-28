package v2

import (
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// BackendLBPoliciesTrigger is a processor for BackendLB objects
type BackendLBPoliciesTrigger struct{}

// Insert adds a BackendLB object to the cache and returns true if the cache is changed
func (p *BackendLBPoliciesTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	//cm, ok := obj.(*corev1.BackendLB)
	//if !ok {
	//	log.Error().Msgf("unexpected object type %T", obj)
	//	return false
	//}
	//
	//key := utils.ObjectKey(cm)
	//
	//return cache.IsBackendLBReferred(key)

	return true
}

// Delete removes a BackendLB object from the cache and returns true if the cache is changed
func (p *BackendLBPoliciesTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	//cm, ok := obj.(*corev1.BackendLB)
	//if !ok {
	//	log.Error().Msgf("unexpected object type %T", obj)
	//	return false
	//}
	//
	//key := utils.ObjectKey(cm)
	//
	//return cache.IsBackendLBReferred(key)

	return true
}
