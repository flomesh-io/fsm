package cache

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// GatewayTLSPoliciesTrigger is responsible for processing GatewayTLSPolicy objects
type GatewayTLSPoliciesTrigger struct{}

// Insert adds a GatewayTLSPolicy to the cache and returns true if the target service is routable
func (p *GatewayTLSPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.GatewayTLSPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	cache.gatewaytls[utils.ObjectKey(policy)] = struct{}{}

	return cache.isEffectiveTargetRef(policy.Spec.TargetRef)
}

// Delete removes a GatewayTLSPolicy from the cache and returns true if the policy was found
func (p *GatewayTLSPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.GatewayTLSPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	key := utils.ObjectKey(policy)
	_, found := cache.gatewaytls[key]
	delete(cache.gatewaytls, key)

	return found
}
