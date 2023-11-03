package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// FaultInjectionPoliciesProcessor is responsible for processing FaultInjectionPolicy objects
type FaultInjectionPoliciesProcessor struct {
}

// Insert adds a FaultInjectionPolicy to the cache and returns true if the target service is routable
func (p *FaultInjectionPoliciesProcessor) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.FaultInjectionPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.faultinjections[utils.ObjectKey(policy)] = struct{}{}

	return cache.isEffectiveTargetRef(policy.Spec.TargetRef)
}

// Delete removes a FaultInjectionPolicy from the cache and returns true if the policy was found
func (p *FaultInjectionPoliciesProcessor) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.FaultInjectionPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(policy)
	_, found := cache.faultinjections[key]
	delete(cache.faultinjections, key)

	return found
}
