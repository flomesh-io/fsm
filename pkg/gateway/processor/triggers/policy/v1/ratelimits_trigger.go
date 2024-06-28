package v1

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// RateLimitPoliciesTrigger is responsible for processing RateLimitPolicy objects
type RateLimitPoliciesTrigger struct{}

// Insert adds a RateLimitPolicy to the cache and returns true if the target service is routable
func (p *RateLimitPoliciesTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsEffectiveTargetRef(policy, policy.Spec.TargetRef)
}

// Delete removes a RateLimitPolicy from the cache and returns true if the policy was found
func (p *RateLimitPoliciesTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsEffectiveTargetRef(policy, policy.Spec.TargetRef)
}
