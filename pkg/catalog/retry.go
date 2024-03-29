package catalog

import (
	"github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/service"
)

// GetRetryPolicy returns the RetryPolicySpec for the given downstream identity and upstream service
// TODO: Add support for wildcard destinations
func (mc *MeshCatalog) GetRetryPolicy(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) *v1alpha1.RetryPolicySpec {
	if !mc.configurator.GetFeatureFlags().EnableRetryPolicy {
		log.Trace().Msgf("Retry policy flag not enabled")
		return nil
	}
	src := downstreamIdentity.ToK8sServiceAccount()

	// List the retry policies for the source
	retryPolicies := mc.policyController.ListRetryPolicies(src)
	if retryPolicies == nil {
		log.Trace().Msgf("Did not find retry policy for downstream service %s", src)
		return nil
	}

	for _, retryCRD := range retryPolicies {
		for _, dest := range retryCRD.Spec.Destinations {
			if dest.Kind != "Service" {
				log.Error().Msgf("Retry policy destinations must be a service: %s is a %s", dest, dest.Kind)
				continue
			}
			destMeshSvc := service.MeshService{Name: dest.Name, Namespace: dest.Namespace}
			// we want all statefulset replicas to have the same retry policy regardless of how they're accessed
			// for the default use-case, this is equivalent to a name + namespace equality check
			if upstreamSvc.SiblingTo(destMeshSvc) {
				// Will return retry policy that applies to the specific upstream service
				return &retryCRD.Spec.RetryPolicy
			}
		}
	}

	log.Trace().Msgf("Could not find retry policy for source %s and destination %s", src, upstreamSvc)
	return nil
}
