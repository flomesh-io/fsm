package policy

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
type GetAttachedPoliciesFunc func(svc client.Object) ([]client.Object, *metav1.Condition)

// FindConflictFunc is a function for finding conflicted policy for a service port
type FindConflictFunc func(policy client.Object, allPolicies []client.Object, port int32) *types.NamespacedName

// Process processes the service level policy status
func (p *ServicePolicyStatusProcessor) Process(ctx context.Context, update *PolicyUpdate) metav1.Condition {
	policy := update.GetResource()
	targetRef := update.GetTargetRef()

	_, ok := p.getServiceGroupKindObjectMapping()[string(targetRef.Group)]
	if !ok {
		return update.AddCondition(invalidCondition(fmt.Sprintf("Invalid target reference group %q, only %q is/are supported", targetRef.Group, strings.Join(p.supportedGroups(), ","))))
	}

	svc := p.getServiceObjectByGroupKind(targetRef.Group, targetRef.Kind)
	if svc == nil {
		return update.AddCondition(invalidCondition(fmt.Sprintf("Invalid target reference kind %q, only %q are supported", targetRef.Kind, strings.Join(p.supportedKinds(), ","))))
	}

	key := types.NamespacedName{
		Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()),
		Name:      string(targetRef.Name),
	}

	if err := p.Get(ctx, key, svc); err != nil {
		if errors.IsNotFound(err) {
			return update.AddCondition(notFoundCondition(fmt.Sprintf("Invalid target reference, cannot find target %s %q", targetRef.Kind, key.String())))
		} else {
			return update.AddCondition(invalidCondition(fmt.Sprintf("Failed to get target %s %q: %s", targetRef.Kind, key, err)))
		}
	}

	policies, condition := p.getSortedAttachedPolices(svc)
	if condition != nil {
		return *condition
	}

	switch svc := svc.(type) {
	case *corev1.Service:
		if conflict := p.getConflictedPolicyByService(policy, policies, svc); conflict != nil {
			return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
		}
	case *mcsv1alpha1.ServiceImport:
		if conflict := p.getConflictedPolicyByServiceImport(policy, policies, svc); conflict != nil {
			return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
		}
	}

	return update.AddCondition(acceptedCondition())
}

func (p *ServicePolicyStatusProcessor) getSortedAttachedPolices(svc client.Object) ([]client.Object, *metav1.Condition) {
	policies, condition := p.GetAttachedPolicies(svc)
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
