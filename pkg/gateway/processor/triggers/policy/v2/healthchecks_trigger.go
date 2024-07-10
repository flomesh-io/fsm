package v2

import (
	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/gateway/processor"
)

// HealthCheckPoliciesTrigger is responsible for processing HealthCheckPolicy objects
type HealthCheckPoliciesTrigger struct{}

// Insert adds a HealthCheckPolicy to the processor and returns true if the target service is routable
func (p *HealthCheckPoliciesTrigger) Insert(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha2.HealthCheckPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableNamespacedTargetServices(policy, policy.Spec.TargetRefs)
}

// Delete removes a HealthCheckPolicy from the processor and returns true if the target service is routable
func (p *HealthCheckPoliciesTrigger) Delete(obj interface{}, processor processor.Processor) bool {
	policy, ok := obj.(*gwpav1alpha2.HealthCheckPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	return processor.IsRoutableNamespacedTargetServices(policy, policy.Spec.TargetRefs)
}
