package status

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
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
type GetPoliciesFunc func(policy client.Object, target client.Object) (map[gwpkg.PolicyMatchType][]client.Object, *metav1.Condition)

// FindConflictPortFunc finds the conflicted port level policy
type FindConflictPortFunc func(gateway *gwv1.Gateway, policy client.Object, allPortLevelPolicies []client.Object) *types.NamespacedName

// FindConflictedHostnamesBasedPolicyFunc finds the conflicted hostnames based policy
type FindConflictedHostnamesBasedPolicyFunc func(route *gwtypes.RouteContext, policy client.Object, allHostnamesBasedPolicies []client.Object) *types.NamespacedName

// FindConflictedHTTPRouteBasedPolicyFunc finds the conflicted HTTPRoute based policy
type FindConflictedHTTPRouteBasedPolicyFunc func(route *gwv1.HTTPRoute, policy client.Object, allRouteBasedPolicies []client.Object) *types.NamespacedName

// FindConflictedGRPCRouteBasedPolicyFunc finds the conflicted GRPCRoute based policy
type FindConflictedGRPCRouteBasedPolicyFunc func(route *gwv1.GRPCRoute, policy client.Object, allRouteBasedPolicies []client.Object) *types.NamespacedName

// Process processes the policy status of port, hostnames and route level
func (p *PolicyStatusProcessor) Process(ctx context.Context, policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference) metav1.Condition {
	_, ok := p.getGatewayAPIGroupKindObjectMapping()[string(targetRef.Group)]
	if !ok {
		return InvalidCondition(policy, fmt.Sprintf("Invalid target reference group, only %q is/are supported", strings.Join(p.supportedGroups(), ",")))
	}

	target := p.getGatewayAPIObjectByGroupKind(targetRef.Group, targetRef.Kind)
	if target == nil {
		return InvalidCondition(policy, fmt.Sprintf("Invalid target reference kind, only %q are supported", strings.Join(p.supportedKinds(), ",")))
	}

	referenceGrants := p.Informer.GetGatewayResourcesFromCache(informers.ReferenceGrantResourceType, false)
	if !gwutils.HasAccessToTargetRef(policy, targetRef, referenceGrants) {
		return NoAccessCondition(policy, fmt.Sprintf("Cross namespace reference to target %s/%s/%s is not allowed", targetRef.Kind, ns(targetRef.Namespace), targetRef.Name))
	}

	key := types.NamespacedName{
		Namespace: gwutils.Namespace(targetRef.Namespace, policy.GetNamespace()),
		Name:      string(targetRef.Name),
	}

	if err := p.Get(ctx, key, target); err != nil {
		if errors.IsNotFound(err) {
			return NotFoundCondition(policy, fmt.Sprintf("Invalid target reference, cannot find target %s %q", targetRef.Kind, key.String()))
		} else {
			return InvalidCondition(policy, fmt.Sprintf("Failed to get target %s %q: %s", targetRef.Kind, key, err))
		}
	}

	policies, condition := p.getSortedPolices(policy, target)
	if condition != nil {
		return *condition
	}

	switch obj := target.(type) {
	case *gwv1.Gateway:
		if p.FindConflictPort != nil && len(policies[gwpkg.PolicyMatchTypePort]) > 0 {
			if conflict := p.FindConflictPort(obj, policy, policies[gwpkg.PolicyMatchTypePort]); conflict != nil {
				return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
			}
		}
	case *gwv1.HTTPRoute:
		if p.FindConflictedHostnamesBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeHostnames]) > 0 {
			info := gwutils.ToRouteContext(obj)

			if conflict := p.FindConflictedHostnamesBasedPolicy(info, policy, policies[gwpkg.PolicyMatchTypeHostnames]); conflict != nil {
				return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
			}
		}

		if p.FindConflictedHTTPRouteBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeHTTPRoute]) > 0 {
			if conflict := p.FindConflictedHTTPRouteBasedPolicy(obj, policy, policies[gwpkg.PolicyMatchTypeHTTPRoute]); conflict != nil {
				return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
			}
		}
	case *gwv1.GRPCRoute:
		if p.FindConflictedHostnamesBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeHostnames]) > 0 {
			info := gwutils.ToRouteContext(obj)

			if conflict := p.FindConflictedHostnamesBasedPolicy(info, policy, policies[gwpkg.PolicyMatchTypeHostnames]); conflict != nil {
				return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
			}
		}

		if p.FindConflictedGRPCRouteBasedPolicy != nil && len(policies[gwpkg.PolicyMatchTypeGRPCRoute]) > 0 {
			if conflict := p.FindConflictedGRPCRouteBasedPolicy(obj, policy, policies[gwpkg.PolicyMatchTypeGRPCRoute]); conflict != nil {
				return ConflictCondition(policy, fmt.Sprintf("Conflict with %s: %s", policy.GetObjectKind().GroupVersionKind().Kind, conflict))
			}
		}
	}

	return AcceptedCondition(policy)
}

func (p *PolicyStatusProcessor) getSortedPolices(policy client.Object, target client.Object) (map[gwpkg.PolicyMatchType][]client.Object, *metav1.Condition) {
	policies, condition := p.GetPolicies(policy, target)
	if condition != nil {
		return nil, condition
	}

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

	return policies, nil
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

func ns(namespace *gwv1.Namespace) string {
	if namespace == nil {
		return ""
	}

	return string(*namespace)
}
