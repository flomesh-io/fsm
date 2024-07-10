package v2

import (
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// BackendTLSPoliciesTrigger is a processor for BackendTLS objects
type BackendTLSPoliciesTrigger struct{}

// Insert adds a BackendTLS object to the processor and returns true if the processor is changed
func (p *BackendTLSPoliciesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwv1alpha3.BackendTLSPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	targetRefs := make([]gwv1alpha2.LocalPolicyTargetReference, 0)
	for _, ref := range policy.Spec.TargetRefs {
		targetRefs = append(targetRefs, ref.LocalPolicyTargetReference)
	}

	return processor.IsRoutableLocalTargetServices(policy, targetRefs)
}

// Delete removes a BackendTLS object from the processor and returns true if the processor is changed
func (p *BackendTLSPoliciesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwv1alpha3.BackendTLSPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	targetRefs := make([]gwv1alpha2.LocalPolicyTargetReference, 0)
	for _, ref := range policy.Spec.TargetRefs {
		targetRefs = append(targetRefs, ref.LocalPolicyTargetReference)
	}

	return processor.IsRoutableLocalTargetServices(policy, targetRefs)
}
