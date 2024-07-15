package v2

import (
	"fmt"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// ---

type BackendTLSPolicyProcessor struct {
	generator *ConfigGenerator
}

func NewBackendTLSPolicyProcessor(c *ConfigGenerator) BackendPolicyProcessor {
	return &BackendTLSPolicyProcessor{
		generator: c,
	}
}

func (p *BackendTLSPolicyProcessor) Process(route client.Object, routeParentRef gwv1.ParentReference, routeRule any, backendRef gwv1.BackendObjectReference, svcPort *fgwv2.ServicePortName) {
	targetRef := gwv1alpha2.LocalPolicyTargetReferenceWithSectionName{
		LocalPolicyTargetReference: gwv1alpha2.LocalPolicyTargetReference{
			Group: ptr.Deref(backendRef.Group, corev1.GroupName),
			Kind:  ptr.Deref(backendRef.Kind, constants.KubernetesServiceKind),
			Name:  backendRef.Name,
		},
		SectionName: ptr.To(gwv1alpha2.SectionName(svcPort.SectionName)),
	}

	policy, found := gwutils.FindBackendTLSPolicy(p.generator.client, targetRef, route.GetNamespace())
	if !found {
		return
	}

	if !gwutils.IsPolicyAcceptedForAncestor(routeParentRef, policy.Status.Ancestors) {
		return
	}

	p2 := p.getOrCreateBackendTLSPolicy(policy)
	if p2 == nil {
		return
	}

	p2.AddTargetRef(fgwv2.NewBackendRef(svcPort.String()))
	p.processCACertificates(policy, p2)
}

func (p *BackendTLSPolicyProcessor) getOrCreateBackendTLSPolicy(policy *gwv1alpha3.BackendTLSPolicy) *fgwv2.BackendTLSPolicy {
	key := client.ObjectKeyFromObject(policy).String()

	p2, ok := p.generator.backendTLSPolicies[key]
	if ok {
		return p2
	}

	p2 = &fgwv2.BackendTLSPolicy{}
	if err := gwutils.DeepCopy(p2, policy); err != nil {
		log.Error().Err(err).Msgf("Failed to copy BackendTLSPolicy %s", key)
		return nil
	}

	p2.Spec.Validation.CACertificates = make([]map[string]string, 0)
	p.generator.backendTLSPolicies[key] = p2

	return p2
}

func (p *BackendTLSPolicyProcessor) processCACertificates(policy *gwv1alpha3.BackendTLSPolicy, p2 *fgwv2.BackendTLSPolicy) {
	for index, ref := range policy.Spec.Validation.CACertificateRefs {
		caName := fmt.Sprintf("bktls-%s-%s-%d.crt", policy.Namespace, policy.Name, index)
		if _, ok := p.generator.secretFiles[caName]; ok {
			continue
		}

		ref := gwv1.ObjectReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Namespace: ptr.To(gwv1.Namespace(policy.Namespace)),
			Name:      ref.Name,
		}

		ca := gwutils.ObjectRefToCACertificate(p.generator.client, policy, ref, nil)
		if len(ca) == 0 {
			continue
		}

		p2.Spec.Validation.CACertificates = append(p2.Spec.Validation.CACertificates, map[string]string{
			corev1.ServiceAccountRootCAKey: caName,
		})

		p.generator.secretFiles[caName] = string(ca)
	}
}
