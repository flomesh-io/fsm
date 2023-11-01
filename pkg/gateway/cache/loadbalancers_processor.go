package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// LoadBalancerPoliciesProcessor is responsible for processing LoadBalancerPolicy objects
type LoadBalancerPoliciesProcessor struct {
}

// Insert adds a LoadBalancerPolicy to the cache and returns true if the target service is routable
func (p *LoadBalancerPoliciesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.LoadBalancerPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.loadbalancers[utils.ObjectKey(policy)] = struct{}{}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a LoadBalancerPolicy from the cache and returns true if the policy was found
func (p *LoadBalancerPoliciesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.LoadBalancerPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(policy)
	_, found := cache.loadbalancers[key]
	delete(cache.loadbalancers, key)

	return found
}
