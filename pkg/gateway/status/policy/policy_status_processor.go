package policy

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
	"github.com/flomesh-io/fsm/pkg/gateway/status/route"

	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PolicyStatusProcessor is a processor for processing port level policy status
type PolicyStatusProcessor struct {
	client.Client
	Informer                           *informers.InformerCollection
	GetPolicies                        GetPoliciesFunc
	FindConflictPort                   FindConflictPortFunc
	FindConflictedHostnamesBasedPolicy FindConflictedHostnamesBasedPolicyFunc
	FindConflictedHTTPRouteBasedPolicy FindConflictedHTTPRouteBasedPolicyFunc
	FindConflictedGRPCRouteBasedPolicy FindConflictedGRPCRouteBasedPolicyFunc
	GroupKindObjectMapping             map[string]map[string]client.Object
}

// GetObjectByGroupKindFunc returns the object by group and kind
type GetObjectByGroupKindFunc func(group gwv1.Group, kind gwv1.Kind) client.Object

// GetPoliciesFunc returns the policies, and returns condition if there is any error
type GetPoliciesFunc func(target client.Object) map[gwpkg.PolicyMatchType][]client.Object

// FindConflictPortFunc finds the conflicted port level policy
type FindConflictPortFunc func(gateway *gwv1.Gateway, policy client.Object, allPortLevelPolicies []client.Object) *types.NamespacedName

// FindConflictedHostnamesBasedPolicyFunc finds the conflicted hostnames based policy
type FindConflictedHostnamesBasedPolicyFunc func(route status.RouteStatusObject, parentRefs []gwv1.ParentReference, policy client.Object, allHostnamesBasedPolicies []client.Object) *types.NamespacedName

// FindConflictedHTTPRouteBasedPolicyFunc finds the conflicted HTTPRoute based policy
type FindConflictedHTTPRouteBasedPolicyFunc func(route *gwv1.HTTPRoute, policy client.Object, allRouteBasedPolicies []client.Object) *types.NamespacedName

// FindConflictedGRPCRouteBasedPolicyFunc finds the conflicted GRPCRoute based policy
type FindConflictedGRPCRouteBasedPolicyFunc func(route *gwv1.GRPCRoute, policy client.Object, allRouteBasedPolicies []client.Object) *types.NamespacedName

// Process processes the policy status of port, hostnames and route level
func (p *PolicyStatusProcessor) Process(ctx context.Context, updater status.Updater, u *PolicyUpdate) {
	p.internalProcess(ctx, u)

	updater.Send(status.Update{
		Resource:       u.GetResource(),
		NamespacedName: u.GetFullName(),
		Mutator:        u,
	})
}

func (p *PolicyStatusProcessor) internalProcess(ctx context.Context, update *PolicyUpdate) metav1.Condition {
	policy := update.GetResource()
	targetRef := update.GetTargetRef()

	_, ok := p.getGatewayAPIGroupKindObjectMapping()[string(targetRef.Group)]
	if !ok {
		return update.AddCondition(invalidCondition(fmt.Sprintf("Invalid target reference group, only %q is/are supported", strings.Join(p.supportedGroups(), ","))))
	}

	target := p.getGatewayAPIObjectByGroupKind(targetRef.Group, targetRef.Kind)
	if target == nil {
		return update.AddCondition(invalidCondition(fmt.Sprintf("Invalid target reference kind, only %q are supported", strings.Join(p.supportedKinds(), ","))))
	}

	key := types.NamespacedName{
		Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()),
		Name:      string(targetRef.Name),
	}

	if err := p.Get(ctx, key, target); err != nil {
		if errors.IsNotFound(err) {
			return update.AddCondition(notFoundCondition(fmt.Sprintf("Invalid target reference, cannot find target %s %q", targetRef.Kind, key.String())))
		} else {
			return update.AddCondition(invalidCondition(fmt.Sprintf("Failed to get target %s %q: %s", targetRef.Kind, key, err)))
		}
	}

	policies := p.getSortedPolices(target)

	switch obj := target.(type) {
	case *gwv1.Gateway:
		if p.FindConflictPort != nil && len(policies[gwpkg.PolicyMatchTypePort]) > 0 {
			if conflict := p.FindConflictPort(obj, policy, policies[gwpkg.PolicyMatchTypePort]); conflict != nil {
				return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
			}
		}
	case *gwv1.HTTPRoute:
		if p.FindConflictedHostnamesBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeHostnames]) > 0 {
			info := route.NewRouteStatusHolder(
				obj,
				&obj.ObjectMeta,
				&obj.TypeMeta,
				obj.Spec.Hostnames,
				gwutils.ToSlicePtr(obj.Status.Parents),
			)

			if conflict := p.FindConflictedHostnamesBasedPolicy(info, obj.Spec.ParentRefs, policy, policies[gwpkg.PolicyMatchTypeHostnames]); conflict != nil {
				return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
			}
		}

		if p.FindConflictedHTTPRouteBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeHTTPRoute]) > 0 {
			if conflict := p.FindConflictedHTTPRouteBasedPolicy(obj, policy, policies[gwpkg.PolicyMatchTypeHTTPRoute]); conflict != nil {
				return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
			}
		}
	case *gwv1.GRPCRoute:
		if p.FindConflictedHostnamesBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeHostnames]) > 0 {
			info := route.NewRouteStatusHolder(
				obj,
				&obj.ObjectMeta,
				&obj.TypeMeta,
				obj.Spec.Hostnames,
				gwutils.ToSlicePtr(obj.Status.Parents),
			)

			if conflict := p.FindConflictedHostnamesBasedPolicy(info, obj.Spec.ParentRefs, policy, policies[gwpkg.PolicyMatchTypeHostnames]); conflict != nil {
				return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
			}
		}

		if p.FindConflictedGRPCRouteBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeGRPCRoute]) > 0 {
			if conflict := p.FindConflictedGRPCRouteBasedPolicy(obj, policy, policies[gwpkg.PolicyMatchTypeGRPCRoute]); conflict != nil {
				return update.AddCondition(conflictCondition(fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict)))
			}
		}
	}

	return update.AddCondition(acceptedCondition())
}

func (p *PolicyStatusProcessor) getSortedPolices(target client.Object) map[gwpkg.PolicyMatchType][]client.Object {
	policies := p.GetPolicies(target)

	// sort each type of by creation timestamp, then by namespace/name
	for matchType, ps := range policies {
		sort.Slice(ps, func(i, j int) bool {
			if ps[i].GetCreationTimestamp().Time.Equal(ps[j].GetCreationTimestamp().Time) {
				return client.ObjectKeyFromObject(ps[i]).String() < client.ObjectKeyFromObject(ps[j]).String()
			}

			return ps[i].GetCreationTimestamp().Time.Before(ps[j].GetCreationTimestamp().Time)
		})
		policies[matchType] = ps
	}

	return policies
}

func (p *PolicyStatusProcessor) getGatewayAPIObjectByGroupKind(group gwv1.Group, kind gwv1.Kind) client.Object {
	mapping := p.getGatewayAPIGroupKindObjectMapping()

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

func (p *PolicyStatusProcessor) getGatewayAPIGroupKindObjectMapping() map[string]map[string]client.Object {
	if len(p.GroupKindObjectMapping) > 0 {
		return p.GroupKindObjectMapping
	}

	return defaultGatewayAPIObjectMapping
}

func (p *PolicyStatusProcessor) supportedGroups() []string {
	groups := make([]string, 0)
	for g := range p.getGatewayAPIGroupKindObjectMapping() {
		if g == constants.KubernetesCoreGroup {
			groups = append(groups, "Kubernetes core")
			continue
		}

		groups = append(groups, g)
	}

	return groups
}

func (p *PolicyStatusProcessor) supportedKinds() []string {
	kinds := make([]string, 0)
	for _, g := range p.getGatewayAPIGroupKindObjectMapping() {
		for k := range g {
			kinds = append(kinds, k)
		}
	}

	return kinds
}
