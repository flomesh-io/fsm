package v2

import (
	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// RetryPoliciesTrigger is responsible for processing TLSRoute objects
type RetryPoliciesTrigger struct{}

// Insert adds a RetryPolicy to the cache and returns true if the target service is routable
func (p *RetryPoliciesTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha2.RetryPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsRoutableTargetServices(policy, policy.Spec.TargetRefs)
}

// Delete removes a RetryPolicy from the cache and returns true if the policy was found
func (p *RetryPoliciesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha2.RetryPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableTargetServices(policy, policy.Spec.TargetRefs)
}
