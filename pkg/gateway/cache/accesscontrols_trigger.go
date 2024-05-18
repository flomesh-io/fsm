package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

// AccessControlPoliciesTrigger is responsible for processing AccessControlPolicy objects
type AccessControlPoliciesTrigger struct{}

// Insert adds a AccessControlPolicy to the cache and returns true if the target service is routable
func (p *AccessControlPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.AccessControlPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isEffectiveTargetRef(policy, policy.Spec.TargetRef)
}

// Delete removes a AccessControlPolicy from the cache and returns true if the policy was found
func (p *AccessControlPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.AccessControlPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.isEffectiveTargetRef(policy, policy.Spec.TargetRef)
}
