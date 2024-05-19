package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// CircuitBreakingPoliciesTrigger is responsible for processing CircuitBreakingPolicy objects
type CircuitBreakingPoliciesTrigger struct{}

// Insert adds a CircuitBreakingPolicy to the cache and returns true if the target service is routable
func (p *CircuitBreakingPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.CircuitBreakingPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a CircuitBreakingPolicy from the cache and returns true if the policy was found
func (p *CircuitBreakingPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.CircuitBreakingPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}
