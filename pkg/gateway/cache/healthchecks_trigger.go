package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// HealthCheckPoliciesTrigger is responsible for processing HealthCheckPolicy objects
type HealthCheckPoliciesTrigger struct{}

// Insert adds a HealthCheckPolicy to the cache and returns true if the target service is routable
func (p *HealthCheckPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.HealthCheckPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//cache.healthchecks[utils.ObjectKey(policy)] = struct{}{}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a HealthCheckPolicy from the cache and returns true if the policy was found
func (p *HealthCheckPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.HealthCheckPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}
	//
	//cache.mutex.Lock()
	//defer cache.mutex.Unlock()
	//
	//key := utils.ObjectKey(policy)
	//_, found := cache.healthchecks[key]
	//delete(cache.healthchecks, key)
	//
	//return found

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}
