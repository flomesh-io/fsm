package v2

import (
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// BackendTLSPoliciesTrigger is a processor for BackendTLS objects
type BackendTLSPoliciesTrigger struct{}

// Insert adds a BackendTLS object to the cache and returns true if the cache is changed
func (p *BackendTLSPoliciesTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	//cm, ok := obj.(*gwv1alpha3.BackendTLSPolicy)
	//if !ok {
	//    log.Error().Msgf("unexpected object type %T", obj)
	//    return false
	//}
	//
	//key := utils.ObjectKey(cm)
	//
	//return cache.IsBackendTLSReferred(key)

	return true
}

// Delete removes a BackendTLS object from the cache and returns true if the cache is changed
func (p *BackendTLSPoliciesTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	//cm, ok := obj.(*gwv1alpha3.BackendTLSPolicy)
	//if !ok {
	//    log.Error().Msgf("unexpected object type %T", obj)
	//    return false
	//}
	//
	//key := utils.ObjectKey(cm)
	//
	//return cache.IsBackendTLSReferred(key)

	return true
}
