package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// LoadBalancerPoliciesTrigger is responsible for processing LoadBalancerPolicy objects
type LoadBalancerPoliciesTrigger struct{}

// Insert adds a LoadBalancerPolicy to the cache and returns true if the target service is routable
func (p *LoadBalancerPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.LoadBalancerPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//cache.loadbalancers[utils.ObjectKey(policy)] = struct{}{}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a LoadBalancerPolicy from the cache and returns true if the policy was found
func (p *LoadBalancerPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.LoadBalancerPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}
	//
	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//key := utils.ObjectKey(policy)
	//_, found := cache.loadbalancers[key]
	//delete(cache.loadbalancers, key)
	//
	//return found

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}
