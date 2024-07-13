package v2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	"github.com/flomesh-io/fsm/pkg/gateway/status/policies"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ---

type RetryPolicyProcessor struct {
	generator *ConfigGenerator
}

func NewRetryPolicyProcessor(c *ConfigGenerator) BackendPolicyProcessor {
	return &RetryPolicyProcessor{
		generator: c,
	}
}

func (p *RetryPolicyProcessor) Process(route client.Object, routeParentRef gwv1.ParentReference, routeRule any, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName) {
	targetRef := gwv1alpha2.NamespacedPolicyTargetReference{
		Group:     ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:      ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Namespace: backendRef.Namespace,
		Name:      backendRef.Name,
	}

	policy, port, found := gwutils.FindRetryPolicy(p.generator.client, targetRef, route.GetNamespace(), svcPort)
	if !found {
		return
	}

	if !gwutils.IsPolicyAcceptedForAncestor(routeParentRef, policy.Status.Ancestors) {
		return
	}

	psh := policies.NewPolicyStatusHolderWithNamespacedPolicyTargetReference(
		policy,
		&policy.ObjectMeta,
		&policy.TypeMeta,
		policy.Spec.TargetRefs,
		gwutils.ToSlicePtr(policy.Status.Ancestors),
	)

	ash := psh.StatusUpdateFor(routeParentRef)

	if !gwutils.HasAccessToBackendTargetRef(p.generator.client, policy, targetRef, ash) {
		return
	}

	p2 := p.getOrCreateRetryPolicy(policy)
	if p2 == nil {
		return
	}

	port2 := &gwpav1alpha2.PortRetry{}
	if err := gwutils.DeepCopy(port2, port); err != nil {
		log.Error().Err(err).Msgf("Failed to copy PortRetry: %s", err)
		return
	}

	p2.AddPort(*port2)
	p2.AddTargetRef(fgwv2.NewBackendRef(svcPort.String()))
}

func (p *RetryPolicyProcessor) getOrCreateRetryPolicy(policy *gwpav1alpha2.RetryPolicy) *fgwv2.RetryPolicy {
	key := client.ObjectKeyFromObject(policy).String()

	p2, ok := p.generator.retryPolicies[key]
	if ok {
		return p2
	}

	p2 = &fgwv2.RetryPolicy{}
	if err := gwutils.DeepCopy(p2, policy); err != nil {
		log.Error().Err(err).Msgf("Failed to copy RetryPolicy %s", key)
		return nil
	}

	p.generator.retryPolicies[key] = p2

	return p2
}
