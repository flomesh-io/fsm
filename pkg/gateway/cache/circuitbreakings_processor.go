package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// CircuitBreakingPoliciesProcessor is responsible for processing CircuitBreakingPolicy objects
type CircuitBreakingPoliciesProcessor struct {
}

// Insert adds a CircuitBreakingPolicy to the cache and returns true if the target service is routable
func (p *CircuitBreakingPoliciesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.CircuitBreakingPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.circuitbreakings[utils.ObjectKey(policy)] = struct{}{}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a CircuitBreakingPolicy from the cache and returns true if the policy was found
func (p *CircuitBreakingPoliciesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.CircuitBreakingPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(policy)
	_, found := cache.circuitbreakings[key]
	delete(cache.circuitbreakings, key)

	return found
}
