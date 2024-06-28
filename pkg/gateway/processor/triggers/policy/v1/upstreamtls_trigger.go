package v1

import (
	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// UpstreamTLSPoliciesTrigger is responsible for processing TLSRoute objects
type UpstreamTLSPoliciesTrigger struct{}

// Insert adds a TLSRoute to the cache and returns true if the route is effective
func (p *UpstreamTLSPoliciesTrigger) Insert(obj interface{}, cache processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha1.UpstreamTLSPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a TLSRoute from the cache and returns true if the route was found
func (p *UpstreamTLSPoliciesTrigger) Delete(obj interface{}, cache processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha1.UpstreamTLSPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return cache.IsRoutableTargetService(policy, policy.Spec.TargetRef)
}
