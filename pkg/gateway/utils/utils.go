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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/flomesh-io/fsm/pkg/logger"

	v1 "k8s.io/client-go/listers/core/v1"

	"k8s.io/apimachinery/pkg/labels"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/gobwas/glob"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/apis/gateway"
	"github.com/flomesh-io/fsm/pkg/constants"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

var (
	log = logger.New("fsm-gateway/utils")
)

// Namespace returns the namespace if it is not nil, otherwise returns the default namespace
func Namespace(ns *gwv1.Namespace, defaultNs string) string {
	if ns == nil {
		return defaultNs
	}

	return string(*ns)
}

// IsAcceptedGatewayClass returns true if the gateway class is accepted
func IsAcceptedGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gwv1.GatewayClassConditionStatusAccepted))
}

// IsActiveGatewayClass returns true if the gateway class is active
func IsActiveGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gateway.GatewayClassConditionStatusActive))
}

// IsEffectiveGatewayClass returns true if the gateway class is effective
func IsEffectiveGatewayClass(gatewayClass *gwv1.GatewayClass) bool {
	return IsAcceptedGatewayClass(gatewayClass) && IsActiveGatewayClass(gatewayClass)
}

// IsAcceptedGateway returns true if the gateway is accepted
func IsAcceptedGateway(gateway *gwv1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1.GatewayConditionAccepted))
}

// IsActiveGateway returns true if the gateway is active
func IsActiveGateway(gateway *gwv1.Gateway) bool {
	hasValidListener := false

	for _, listenerStatus := range gateway.Status.Listeners {
		if IsListenerAccepted(listenerStatus) && IsListenerProgrammed(listenerStatus) {
			hasValidListener = true
			break
		}
	}

	return IsAcceptedGateway(gateway) && hasValidListener
}

// IsListenerProgrammed returns true if the listener is programmed
func IsListenerProgrammed(listenerStatus gwv1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1.ListenerConditionProgrammed))
}

// IsListenerAccepted returns true if the listener is accepted
func IsListenerAccepted(listenerStatus gwv1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1.ListenerConditionAccepted))
}

// IsAcceptedPolicyAttachment returns true if the policy attachment is accepted
func IsAcceptedPolicyAttachment(conditions []metav1.Condition) bool {
	return metautil.IsStatusConditionTrue(conditions, string(gwv1alpha2.PolicyConditionAccepted))
}

// IsRefToGateway returns true if the parent reference is to the gateway
func IsRefToGateway(parentRef gwv1.ParentReference, gateway client.ObjectKey) bool {
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

// IsRefToTarget returns true if the target reference is to the target object
func IsRefToTarget(referenceGrants []client.Object, targetRef gwv1alpha2.PolicyTargetReference, object client.Object) bool {
	gvk := object.GetObjectKind().GroupVersionKind()

	if string(targetRef.Group) != gvk.Group {
		return false
	}

	if string(targetRef.Kind) != gvk.Kind {
		return false
	}

	if string(targetRef.Name) != object.GetName() {
		return false
	}

	if targetRef.Namespace != nil && string(*targetRef.Namespace) != object.GetNamespace() && !ValidCrossNamespaceRef(
		referenceGrants,
		gwtypes.CrossNamespaceFrom{
			Group:     gvk.Group,
			Kind:      gvk.Kind,
			Namespace: object.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     string(targetRef.Group),
			Kind:      string(targetRef.Kind),
			Namespace: string(*targetRef.Namespace),
			Name:      string(targetRef.Name),
		},
	) {
		return false
	}

	return true
}

// IsTargetRefToGVK returns true if the target reference is to the given group version kind
func IsTargetRefToGVK(targetRef gwv1alpha2.PolicyTargetReference, gvk schema.GroupVersionKind) bool {
	return string(targetRef.Group) == gvk.Group && string(targetRef.Kind) == gvk.Kind
}

// ObjectKey returns the object key for the given object
func ObjectKey(obj client.Object) client.ObjectKey {
	ns := obj.GetNamespace()
	if ns == "" {
		ns = metav1.NamespaceDefault
	}

	return client.ObjectKey{Namespace: ns, Name: obj.GetName()}
}

// GroupPointer returns a pointer to the given group
func GroupPointer(group string) *gwv1.Group {
	result := gwv1.Group(group)

	return &result
}

// GetValidListenersFromGateway returns the valid listeners from the gateway
func GetValidListenersFromGateway(gw *gwv1.Gateway) []gwtypes.Listener {
	listeners := make(map[gwv1.SectionName]gwv1.Listener)
	for _, listener := range gw.Spec.Listeners {
		listeners[listener.Name] = listener
	}

	validListeners := make([]gwtypes.Listener, 0)
	for _, status := range gw.Status.Listeners {
		if IsListenerAccepted(status) && IsListenerProgrammed(status) {
			l, ok := listeners[status.Name]
			if !ok {
				continue
			}
			validListeners = append(validListeners, gwtypes.Listener{
				Listener:       l,
				SupportedKinds: status.SupportedKinds,
			})
		}
	}

	return validListeners
}

// GetAllowedListeners returns the allowed listeners
func GetAllowedListeners(
	nsLister v1.NamespaceLister,
	gw *gwv1.Gateway,
	parentRef gwv1.ParentReference,
	params *gwtypes.RouteContext,
	validListeners []gwtypes.Listener,
) ([]gwtypes.Listener, []metav1.Condition) {
	routeGvk := params.GVK
	routeGeneration := params.Generation
	routeNs := params.Namespace

	selectedListeners := make([]gwtypes.Listener, 0)
	invalidListenerConditions := make([]metav1.Condition, 0)

	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}

	if len(selectedListeners) == 0 {
		invalidListenerConditions = append(invalidListenerConditions, metav1.Condition{
			Type:               string(gwv1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: routeGeneration,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1.RouteReasonNoMatchingParent),
			Message:            fmt.Sprintf("No listeners match parent ref %s", types.NamespacedName{Namespace: Namespace(parentRef.Namespace, routeNs), Name: string(parentRef.Name)}),
		})
		return nil, invalidListenerConditions
	}

	allowedListeners := make([]gwtypes.Listener, 0)
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(routeGvk) {
			continue
		}

		// Check if the route is in a namespace that the listener allows.
		if !NamespaceMatches(nsLister, selectedListener.AllowedRoutes.Namespaces, gw.Namespace, routeNs) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		invalidListenerConditions = append(invalidListenerConditions, metav1.Condition{
			Type:               string(gwv1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: routeGeneration,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1.RouteReasonNotAllowedByListeners),
			Message:            fmt.Sprintf("No matched listeners of parent ref %s", types.NamespacedName{Namespace: Namespace(parentRef.Namespace, routeNs), Name: string(parentRef.Name)}),
		})
		return nil, invalidListenerConditions
	}

	return allowedListeners, invalidListenerConditions
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

// NamespaceMatches returns true if the namespace matches
func NamespaceMatches(nsLister v1.NamespaceLister, namespaces *gwv1.RouteNamespaces, gatewayNamespace, routeNamespace string) bool {
	if namespaces == nil || namespaces.From == nil {
		return true
	}

	switch *namespaces.From {
	case gwv1.NamespacesFromAll:
		return true
	case gwv1.NamespacesFromSame:
		return gatewayNamespace == routeNamespace
	case gwv1.NamespacesFromSelector:
		namespaceSelector, err := metav1.LabelSelectorAsSelector(namespaces.Selector)
		if err != nil {
			log.Error().Msgf("failed to convert namespace selector: %v", err)
			return false
		}

		ns, err := nsLister.Get(routeNamespace)
		if err != nil {
			log.Error().Msgf("failed to get namespace %s: %v", routeNamespace, err)
			return false
		}

		return namespaceSelector.Matches(labels.Set(ns.Labels))
	}

	return true
}

func ToRouteContext(route client.Object) *gwtypes.RouteContext {
	switch route := route.(type) {
	case *gwv1.HTTPRoute:
		return &gwtypes.RouteContext{
			Meta:         route.GetObjectMeta(),
			ParentRefs:   route.Spec.ParentRefs,
			GVK:          route.GroupVersionKind(),
			Generation:   route.GetGeneration(),
			Hostnames:    route.Spec.Hostnames,
			Namespace:    route.GetNamespace(),
			ParentStatus: route.Status.Parents,
		}
	case *gwv1alpha2.GRPCRoute:
		return &gwtypes.RouteContext{
			Meta:         route.GetObjectMeta(),
			ParentRefs:   route.Spec.ParentRefs,
			GVK:          route.GroupVersionKind(),
			Generation:   route.GetGeneration(),
			Hostnames:    route.Spec.Hostnames,
			Namespace:    route.GetNamespace(),
			ParentStatus: route.Status.Parents,
		}
	case *gwv1alpha2.TLSRoute:
		return &gwtypes.RouteContext{
			Meta:         route.GetObjectMeta(),
			ParentRefs:   route.Spec.ParentRefs,
			GVK:          route.GroupVersionKind(),
			Generation:   route.GetGeneration(),
			Hostnames:    route.Spec.Hostnames,
			Namespace:    route.GetNamespace(),
			ParentStatus: route.Status.Parents,
		}
	case *gwv1alpha2.TCPRoute:
		return &gwtypes.RouteContext{
			Meta:         route.GetObjectMeta(),
			ParentRefs:   route.Spec.ParentRefs,
			GVK:          route.GroupVersionKind(),
			Generation:   route.GetGeneration(),
			Hostnames:    nil,
			Namespace:    route.GetNamespace(),
			ParentStatus: route.Status.Parents,
		}
	case *gwv1alpha2.UDPRoute:
		return &gwtypes.RouteContext{
			Meta:         route.GetObjectMeta(),
			ParentRefs:   route.Spec.ParentRefs,
			GVK:          route.GroupVersionKind(),
			Generation:   route.GetGeneration(),
			Hostnames:    nil,
			Namespace:    route.GetNamespace(),
			ParentStatus: route.Status.Parents,
		}
	default:
		log.Warn().Msgf("Unsupported route type: %T", route)
		return nil
	}
}

func ValidCrossNamespaceRef(referenceGrants []client.Object, from gwtypes.CrossNamespaceFrom, to gwtypes.CrossNamespaceTo) bool {
	for _, referenceGrant := range referenceGrants {
		refGrant := referenceGrant.(*gwv1beta1.ReferenceGrant)

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

		return true
	}

	return false
}
