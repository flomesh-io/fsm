package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// RateLimitPoliciesProcessor is responsible for processing TLSRoute objects
type RateLimitPoliciesProcessor struct {
}

// Insert adds a TLSRoute to the cache and returns true if the route is effective
func (p *RateLimitPoliciesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.ratelimits[utils.ObjectKey(policy)] = struct{}{}

	return cache.isEffectiveRateLimitPolicy(policy.Spec.TargetRef)
}

// Delete removes a TLSRoute from the cache and returns true if the route was found
func (p *RateLimitPoliciesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.RateLimitPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(policy)
	_, found := cache.ratelimits[key]
	delete(cache.ratelimits, key)

	return found
}
