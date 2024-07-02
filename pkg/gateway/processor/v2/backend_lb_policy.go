package v2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	v2 "github.com/flomesh-io/fsm/pkg/gateway/fgw/v2"
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

func (p *BackendLBPolicyProcessor) Process(route client.Object, _ gwv1.ParentReference, backendRef gwv1.BackendObjectReference, svcPort *v2.ServicePortName) {
	targetRef := gwv1alpha2.LocalPolicyTargetReference{
		Group: ptr.Deref(backendRef.Group, corev1.GroupName),
		Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
		Name:  backendRef.Name,
	}

	policy, found := gwutils.FindBackendLBPolicy(p.generator.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	p2 := p.getOrCreateBackendLBPolicy(policy)
	if p2 == nil {
		return
	}

	p2.AddTargetRef(v2.NewBackendRef(svcPort.String()))
}

func (p *BackendLBPolicyProcessor) getOrCreateBackendLBPolicy(policy *gwv1alpha2.BackendLBPolicy) *v2.BackendLBPolicy {
	key := client.ObjectKeyFromObject(policy).String()

	p2, ok := p.generator.backendLBPolicies[key]
	if ok {
		return p2
	}

	p2 = &v2.BackendLBPolicy{}
	if err := gwutils.DeepCopy(p2, policy); err != nil {
		log.Error().Err(err).Msgf("Failed to copy BackendLBPolicy %s", key)
		return nil
	}

	p.generator.backendLBPolicies[key] = p2

	return p2
}
