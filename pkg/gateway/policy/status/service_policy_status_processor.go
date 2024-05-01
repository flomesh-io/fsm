package status

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServicePolicyStatusProcessor is an interface for processing service level policy status
type ServicePolicyStatusProcessor struct {
	client.Client
	Informer               *informers.InformerCollection
	GetAttachedPolicies    GetAttachedPoliciesFunc
	FindConflict           FindConflictFunc
	GroupKindObjectMapping map[string]map[string]client.Object
}

// GetAttachedPoliciesFunc is a function for getting attached policies for a service
type GetAttachedPoliciesFunc func(policy client.Object, svc client.Object) ([]client.Object, *metav1.Condition)

// FindConflictFunc is a function for finding conflicted policy for a service port
type FindConflictFunc func(policy client.Object, allPolicies []client.Object, port int32) *types.NamespacedName

// Process processes the service level policy status
func (p *ServicePolicyStatusProcessor) Process(ctx context.Context, policy client.Object, targetRef gwv1alpha2.PolicyTargetReference) metav1.Condition {
	_, ok := p.getServiceGroupKindObjectMapping()[string(targetRef.Group)]
	if !ok {
		return InvalidCondition(policy, fmt.Sprintf("Invalid target reference group %q, only %q is/are supported", targetRef.Group, strings.Join(p.supportedGroups(), ",")))
	}

	svc := p.getServiceObjectByGroupKind(targetRef.Group, targetRef.Kind)
	if svc == nil {
		return InvalidCondition(policy, fmt.Sprintf("Invalid target reference kind %q, only %q are supported", targetRef.Kind, strings.Join(p.supportedKinds(), ",")))
	}

	referenceGrants := p.Informer.GetGatewayResourcesFromCache(informers.ReferenceGrantResourceType, false)
	if !gwutils.HasAccessToTargetRef(policy, targetRef, referenceGrants) {
		return NoAccessCondition(policy, fmt.Sprintf("Cross namespace reference to target %s/%s/%s is not allowed", targetRef.Kind, ns(targetRef.Namespace), targetRef.Name))
	}

	key := types.NamespacedName{
		Namespace: gwutils.Namespace(targetRef.Namespace, policy.GetNamespace()),
		Name:      string(targetRef.Name),
	}

	if err := p.Get(ctx, key, svc); err != nil {
		if errors.IsNotFound(err) {
			return NotFoundCondition(policy, fmt.Sprintf("Invalid target reference, cannot find target %s %q", targetRef.Kind, key.String()))
		} else {
			return InvalidCondition(policy, fmt.Sprintf("Failed to get target %s %q: %s", targetRef.Kind, key, err))
		}
	}

	policies, condition := p.getSortedAttachedPolices(policy, svc)
	if condition != nil {
		return *condition
	}

	switch svc := svc.(type) {
	case *corev1.Service:
		if conflict := p.getConflictedPolicyByService(policy, policies, svc); conflict != nil {
			return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
		}
	case *mcsv1alpha1.ServiceImport:
		if conflict := p.getConflictedPolicyByServiceImport(policy, policies, svc); conflict != nil {
			return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
		}
	}

	return AcceptedCondition(policy)
}

func (p *ServicePolicyStatusProcessor) getSortedAttachedPolices(policy client.Object, svc client.Object) ([]client.Object, *metav1.Condition) {
	policies, condition := p.GetAttachedPolicies(policy, svc)
	if condition != nil {
		return nil, condition
	}

	sort.Slice(policies, func(i, j int) bool {
		if policies[i].GetCreationTimestamp().Time.Equal(policies[j].GetCreationTimestamp().Time) {
			return client.ObjectKeyFromObject(policies[i]).String() < client.ObjectKeyFromObject(policies[j]).String()
		}

		return policies[i].GetCreationTimestamp().Time.Before(policies[j].GetCreationTimestamp().Time)
	})

	return policies, nil
}

func (p *ServicePolicyStatusProcessor) getConflictedPolicyByService(policy client.Object, allPolicies []client.Object, svc *corev1.Service) *types.NamespacedName {
	for _, port := range svc.Spec.Ports {
		if conflict := p.FindConflict(policy, allPolicies, port.Port); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (p *ServicePolicyStatusProcessor) getConflictedPolicyByServiceImport(policy client.Object, allPolicies []client.Object, svcimp *mcsv1alpha1.ServiceImport) *types.NamespacedName {
	for _, port := range svcimp.Spec.Ports {
		if conflict := p.FindConflict(policy, allPolicies, port.Port); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (p *ServicePolicyStatusProcessor) getServiceObjectByGroupKind(group gwv1.Group, kind gwv1.Kind) client.Object {
	mapping := p.getServiceGroupKindObjectMapping()

	g, found := mapping[string(group)]
	if !found {
		return nil
	}

	obj, found := g[string(kind)]
	if !found {
		return nil
	}

	return obj
}

func (p *ServicePolicyStatusProcessor) getServiceGroupKindObjectMapping() map[string]map[string]client.Object {
	if len(p.GroupKindObjectMapping) > 0 {
		return p.GroupKindObjectMapping
	}

	return defaultServiceObjectMapping
}

func (p *ServicePolicyStatusProcessor) supportedGroups() []string {
	groups := make([]string, 0)
	for g := range p.getServiceGroupKindObjectMapping() {
		if g == constants.KubernetesCoreGroup {
			groups = append(groups, "Kubernetes core")
			continue
		}

		groups = append(groups, g)
	}

	return groups
}

func (p *ServicePolicyStatusProcessor) supportedKinds() []string {
	kinds := make([]string, 0)
	for _, g := range p.getServiceGroupKindObjectMapping() {
		for k := range g {
			kinds = append(kinds, k)
		}
	}

	return kinds
}
