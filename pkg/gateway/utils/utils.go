/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package utils contains utility functions for gateway
package utils

import (
	"sort"
	"strings"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/webhook"

	"github.com/jinzhu/copier"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"

	"github.com/flomesh-io/fsm/pkg/logger"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/gobwas/glob"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

var (
	log = logger.New("fsm-gateway/utils")
)

// NamespaceDerefOr returns the namespace if it is not nil, otherwise returns the default namespace
func NamespaceDerefOr(ns *gwv1.Namespace, defaultNs string) string {
	if ns == nil {
		return defaultNs
	}

	return string(*ns)
}

// IsAcceptedPolicyAttachment returns true if the policy attachment is accepted
func IsAcceptedPolicyAttachment(conditions []metav1.Condition) bool {
	return metautil.IsStatusConditionTrue(conditions, string(gwv1alpha2.PolicyConditionAccepted))
}

// IsRefToGateway returns true if the parent reference is to the gateway
func IsRefToGateway(parentRef gwv1.ParentReference, gateway types.NamespacedName) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gwv1.GroupName {
		return false
	}

	if parentRef.Kind != nil && string(*parentRef.Kind) != constants.GatewayAPIGatewayKind {
		return false
	}

	if parentRef.Namespace != nil && string(*parentRef.Namespace) != gateway.Namespace {
		return false
	}

	return string(parentRef.Name) == gateway.Name
}

func IsLocalObjRefToGateway(targetRef gwv1.LocalObjectReference, gateway types.NamespacedName) bool {
	if string(targetRef.Group) != gwv1.GroupName {
		return false
	}

	if string(targetRef.Kind) != constants.GatewayAPIGatewayKind {
		return false
	}

	return string(targetRef.Name) == gateway.Name
}

// IsTargetRefToTarget returns true if the target reference is to the target resource
func IsTargetRefToTarget(policy client.Object, targetRef gwv1alpha2.NamespacedPolicyTargetReference, target client.Object) bool {
	gvk := target.GetObjectKind().GroupVersionKind()

	if string(targetRef.Group) != gvk.Group {
		return false
	}

	if string(targetRef.Kind) != gvk.Kind {
		return false
	}

	if ns := NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()); ns != target.GetNamespace() {
		return false
	}

	return string(targetRef.Name) == target.GetName()
}

// HasAccessToTargetRef returns true if the policy has access to the target reference
func HasAccessToTargetRef(policy client.Object, ref gwv1alpha2.NamespacedPolicyTargetReference, referenceGrants []*gwv1beta1.ReferenceGrant) bool {
	if ref.Namespace != nil && string(*ref.Namespace) != policy.GetNamespace() && !ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     policy.GetObjectKind().GroupVersionKind().Group,
			Kind:      policy.GetObjectKind().GroupVersionKind().Kind,
			Namespace: policy.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(ref.Group),
			Kind:      string(ref.Kind),
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
		referenceGrants,
	) {
		return false
	}

	return true
}

// IsTargetRefToGVK returns true if the target reference is to the given group version kind
func IsTargetRefToGVK(targetRef gwv1alpha2.NamespacedPolicyTargetReference, gvk schema.GroupVersionKind) bool {
	return string(targetRef.Group) == gvk.Group && string(targetRef.Kind) == gvk.Kind
}

// GroupPointer returns a pointer to the given group
func GroupPointer(group string) *gwv1.Group {
	result := gwv1.Group(group)

	return &result
}

// GetValidHostnames returns the valid hostnames
func GetValidHostnames(listenerHostname *gwv1.Hostname, routeHostnames []gwv1.Hostname) []string {
	if len(routeHostnames) == 0 {
		if listenerHostname != nil {
			return []string{string(*listenerHostname)}
		}

		return []string{"*"}
	}

	hostnames := sets.New[string]()
	for i := range routeHostnames {
		routeHostname := string(routeHostnames[i])

		switch {
		case listenerHostname == nil:
			hostnames.Insert(routeHostname)

		case string(*listenerHostname) == routeHostname:
			hostnames.Insert(routeHostname)

		case strings.HasPrefix(string(*listenerHostname), "*"):
			if HostnameMatchesWildcardHostname(routeHostname, string(*listenerHostname)) {
				hostnames.Insert(routeHostname)
			}

		case strings.HasPrefix(routeHostname, "*"):
			if HostnameMatchesWildcardHostname(string(*listenerHostname), routeHostname) {
				hostnames.Insert(string(*listenerHostname))
			}
		}
	}

	if len(hostnames) == 0 {
		return []string{}
	}

	return hostnames.UnsortedList()
}

// HostnameMatchesWildcardHostname returns true if the hostname matches the wildcard hostname
func HostnameMatchesWildcardHostname(hostname, wildcardHostname string) bool {
	g := glob.MustCompile(wildcardHostname, '.')
	return g.Match(hostname)
}

// ValidCrossNamespaceRef returns if the reference is valid across namespaces based on the reference grants
func ValidCrossNamespaceRef(from gwtypes.CrossNamespaceFrom, to gwtypes.CrossNamespaceTo, referenceGrants []*gwv1beta1.ReferenceGrant) bool {
	if len(referenceGrants) == 0 {
		return false
	}

	for _, refGrant := range referenceGrants {
		log.Debug().Msgf("Evaluating ReferenceGrant: %s/%s", refGrant.GetNamespace(), refGrant.GetName())

		if refGrant.Namespace != to.Namespace {
			log.Debug().Msgf("ReferenceGrant namespace %s does not match to namespace %s", refGrant.Namespace, to.Namespace)
			continue
		}

		var fromAllowed bool
		for _, refGrantFrom := range refGrant.Spec.From {
			if string(refGrantFrom.Namespace) == from.Namespace && string(refGrantFrom.Group) == from.Group && string(refGrantFrom.Kind) == from.Kind {
				fromAllowed = true
				log.Debug().Msgf("ReferenceGrant from %s/%s/%s is allowed", from.Group, from.Kind, from.Namespace)
				break
			}
		}

		if !fromAllowed {
			log.Debug().Msgf("ReferenceGrant from %s/%s/%s is NOT allowed", from.Group, from.Kind, from.Namespace)
			continue
		}

		var toAllowed bool
		for _, refGrantTo := range refGrant.Spec.To {
			if string(refGrantTo.Group) == to.Group && string(refGrantTo.Kind) == to.Kind && (refGrantTo.Name == nil || *refGrantTo.Name == "" || string(*refGrantTo.Name) == to.Name) {
				toAllowed = true
				log.Debug().Msgf("ReferenceGrant to %s/%s/%s/%s is allowed", to.Group, to.Kind, to.Namespace, to.Name)
				break
			}
		}

		if !toAllowed {
			log.Debug().Msgf("ReferenceGrant to %s/%s/%s/%s is NOT allowed", to.Group, to.Kind, to.Namespace, to.Name)
			continue
		}

		log.Debug().Msgf("ReferenceGrant from %s/%s/%s to %s/%s/%s/%s is allowed", from.Group, from.Kind, from.Namespace, to.Group, to.Kind, to.Namespace, to.Name)
		return true
	}

	log.Debug().Msgf("ReferenceGrant from %s/%s/%s to %s/%s/%s/%s is NOT allowed", from.Group, from.Kind, from.Namespace, to.Group, to.Kind, to.Namespace, to.Name)
	return false
}

// FindMatchedReferenceGrant returns if the reference is valid across namespaces based on the reference grants
func FindMatchedReferenceGrant(from gwtypes.CrossNamespaceFrom, to gwtypes.CrossNamespaceTo, referenceGrants []*gwv1beta1.ReferenceGrant) *gwv1beta1.ReferenceGrant {
	if len(referenceGrants) == 0 {
		return nil
	}

	for _, refGrant := range referenceGrants {
		if refGrant.Namespace != to.Namespace {
			continue
		}

		var fromAllowed bool
		for _, refGrantFrom := range refGrant.Spec.From {
			if string(refGrantFrom.Namespace) == from.Namespace && string(refGrantFrom.Group) == from.Group && string(refGrantFrom.Kind) == from.Kind {
				fromAllowed = true
				break
			}
		}

		if !fromAllowed {
			continue
		}

		var toAllowed bool
		for _, refGrantTo := range refGrant.Spec.To {
			if string(refGrantTo.Group) == to.Group && string(refGrantTo.Kind) == to.Kind && (refGrantTo.Name == nil || *refGrantTo.Name == "" || string(*refGrantTo.Name) == to.Name) {
				toAllowed = true
				break
			}
		}

		if !toAllowed {
			continue
		}

		return refGrant
	}

	return nil
}

// SortResources sorts the resources by creation timestamp and name
func SortResources[T client.Object](resources []T) []T {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].GetCreationTimestamp().Time.Equal(resources[j].GetCreationTimestamp().Time) {
			return client.ObjectKeyFromObject(resources[i]).String() < client.ObjectKeyFromObject(resources[j]).String()
		}

		return resources[i].GetCreationTimestamp().Time.Before(resources[j].GetCreationTimestamp().Time)
	})

	return resources
}

func ToSlicePtr[T any](slice []T) []*T {
	ptrs := make([]*T, len(slice))
	for i, v := range slice {
		v := v
		ptrs[i] = &v
	}
	return ptrs
}

// SortFilterRefs sorts the resources by creation timestamp and name
func SortFilterRefs(filterRefs []gwpav1alpha2.LocalFilterReference) []gwpav1alpha2.LocalFilterReference {
	sort.Slice(filterRefs, func(i, j int) bool {
		if filterRefs[i].Priority != nil && filterRefs[j].Priority != nil {
			if *filterRefs[i].Priority == *filterRefs[j].Priority {
				return filterRefs[i].Name < filterRefs[j].Name
			}

			return *filterRefs[i].Priority < *filterRefs[j].Priority
		}

		return filterRefs[i].Name < filterRefs[j].Name
	})

	return filterRefs
}

// IsValidRefToGroupKindOfCA returns true if the reference is to a ConfigMap or Secret in the core group
func IsValidRefToGroupKindOfCA(ref gwv1.ObjectReference) bool {
	if ref.Group != corev1.GroupName {
		return false
	}

	if ref.Kind == constants.KubernetesSecretKind || ref.Kind == constants.KubernetesConfigMapKind {
		return true
	}

	return false
}

// IsValidBackendRefToGroupKindOfService returns true if the reference is to a Service in the core group
func IsValidBackendRefToGroupKindOfService(ref gwv1.BackendObjectReference) bool {
	if ref.Group == nil {
		return false
	}

	if ref.Kind == nil {
		return false
	}

	if (string(*ref.Kind) == constants.KubernetesServiceKind && string(*ref.Group) == constants.KubernetesCoreGroup) ||
		(string(*ref.Kind) == constants.FlomeshAPIServiceImportKind && string(*ref.Group) == constants.FlomeshMCSAPIGroup) {
		return true
	}

	return false
}

// IsValidRefToGroupKindOfSecret returns true if the reference is to a Secret in the core group
func IsValidRefToGroupKindOfSecret(ref gwv1.SecretObjectReference) bool {
	if ref.Group == nil {
		return false
	}

	if ref.Kind == nil {
		return false
	}

	if string(*ref.Group) == constants.KubernetesCoreGroup && string(*ref.Kind) == constants.KubernetesSecretKind {
		return true
	}

	return false
}

// IsValidTargetRefToGroupKindOfService checks if the target reference is valid to the group kind of service
func IsValidTargetRefToGroupKindOfService(ref gwv1alpha2.NamespacedPolicyTargetReference) bool {
	if (ref.Kind == constants.KubernetesServiceKind && ref.Group == constants.KubernetesCoreGroup) ||
		(ref.Kind == constants.FlomeshAPIServiceImportKind && ref.Group == constants.FlomeshMCSAPIGroup) {
		return true
	}

	return false
}

// DeepCopy copy all fields from source to destination
func DeepCopy(dst any, src any) error {
	return copier.CopyWithOption(dst, src, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}

var IsValidHostname = webhook.IsValidHostname
