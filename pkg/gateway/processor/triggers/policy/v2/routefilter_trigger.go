package v2

import (
	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// RouteRuleFilterPoliciesTrigger is a processor for RouteFilter objects
type RouteRuleFilterPoliciesTrigger struct{}

// Insert adds a RouteFilter object to the processor and returns true if the processor is changed
func (p *RouteRuleFilterPoliciesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha2.RouteRuleFilterPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsValidLocalTargetRoutes(policy, policy.Spec.TargetRefs)
}

// Delete removes a RouteFilter object from the processor and returns true if the processor is changed
func (p *RouteRuleFilterPoliciesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha2.RouteRuleFilterPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsValidLocalTargetRoutes(policy, policy.Spec.TargetRefs)
}
