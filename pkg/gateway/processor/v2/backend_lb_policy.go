package v2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ---

type BackendLBPolicyProcessor struct {
	generator *ConfigGenerator
}

func NewBackendLBPolicyProcessor(c *ConfigGenerator) BackendPolicyProcessor {
	return &BackendLBPolicyProcessor{
		generator: c,
	}
}

func (p *BackendLBPolicyProcessor) Process(route client.Object, routeParentRef gwv1.ParentReference, routeRule any, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName) {
	targetRef := gwv1alpha2.LocalPolicyTargetReference{
		Group: ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Name:  backendRef.Name,
	}

	policy, found := gwutils.FindBackendLBPolicy(p.generator.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	if !gwutils.IsPolicyAcceptedForAncestor(routeParentRef, policy.Status.Ancestors) {
		return
	}

	p2 := p.getOrCreateBackendLBPolicy(policy, route, routeRule)
	if p2 == nil {
		return
	}

	p2.AddTargetRef(fgwv2.NewBackendRef(svcPort.String()))
}

func (p *BackendLBPolicyProcessor) getOrCreateBackendLBPolicy(policy *gwpav1alpha2.BackendLBPolicy, route client.Object, routeRule any) *fgwv2.BackendLBPolicy {
	key := client.ObjectKeyFromObject(policy).String()

	p2, ok := p.generator.backendLBPolicies[key]
	if ok {
		return p2
	}

	p2 = &fgwv2.BackendLBPolicy{}
	if err := gwutils.DeepCopy(p2, policy); err != nil {
		log.Error().Err(err).Msgf("[GW] Failed to copy BackendLBPolicy %s", key)
		return nil
	}

	// Any configuration that is specified at Route Rule level MUST override configuration
	// that is attached at the backend level because route rule have a more global view and
	// responsibility for the overall traffic routing.
	// https://gateway-api.sigs.k8s.io/geps/gep-1619/#route-rule-api
	switch route.(type) {
	case *gwv1.HTTPRoute:
		rule, ok := routeRule.(*gwv1.HTTPRouteRule)
		if !ok {
			log.Error().Msgf("[GW] Unexpected route rule type %T", routeRule)
			return nil
		}

		if rule.SessionPersistence != nil {
			p2.Spec.SessionPersistence = nil
		}
	case *gwv1.GRPCRoute:
		rule, ok := routeRule.(*gwv1.GRPCRouteRule)
		if !ok {
			log.Error().Msgf("[GW] Unexpected route rule type %T", routeRule)
			return nil
		}

		if rule.SessionPersistence != nil {
			p2.Spec.SessionPersistence = nil
		}
	}

	if p2.Spec.SessionPersistence == nil && p2.Spec.Algorithm == nil {
		return nil
	}

	p.generator.backendLBPolicies[key] = p2

	return p2
}
