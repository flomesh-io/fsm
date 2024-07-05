package v2

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// BackendLBPoliciesTrigger is a processor for BackendLB objects
type BackendLBPoliciesTrigger struct{}

// Insert adds a BackendLB object to the processor and returns true if the processor is changed
func (p *BackendLBPoliciesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwv1alpha2.BackendLBPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableLocalTargetServices(policy, policy.Spec.TargetRefs)
}

// Delete removes a BackendLB object from the processor and returns true if the processor is changed
func (p *BackendLBPoliciesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwv1alpha2.BackendLBPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableLocalTargetServices(policy, policy.Spec.TargetRefs)
}
