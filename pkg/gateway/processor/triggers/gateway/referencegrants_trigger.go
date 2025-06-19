package gateway

import (
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// ReferenceGrantTrigger is responsible for processing ReferenceGrant objects
type ReferenceGrantTrigger struct{}

// Insert adds a ReferenceGrant to the processor and returns true if the ReferenceGrant is effective
func (p *ReferenceGrantTrigger) Insert(obj interface{}, _ processor.Processor) bool {
	_, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return true
}

// Delete removes a ReferenceGrant from the processor and returns true if the ReferenceGrant was found
func (p *ReferenceGrantTrigger) Delete(obj interface{}, _ processor.Processor) bool {
	_, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("[GW] unexpected object type %T", obj)
		return false
	}

	return true
}
