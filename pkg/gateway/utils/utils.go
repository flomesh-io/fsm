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
	"reflect"
	"strings"
	"time"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/gobwas/glob"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/apis/gateway"
	"github.com/flomesh-io/fsm/pkg/constants"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

// IsAcceptedGatewayClass returns true if the gateway class is accepted
func IsAcceptedGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gwv1beta1.GatewayClassConditionStatusAccepted))
}

// IsActiveGatewayClass returns true if the gateway class is active
func IsActiveGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gateway.GatewayClassConditionStatusActive))
}

// IsEffectiveGatewayClass returns true if the gateway class is effective
func IsEffectiveGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return IsAcceptedGatewayClass(gatewayClass) && IsActiveGatewayClass(gatewayClass)
}

// IsAcceptedGateway returns true if the gateway is accepted
func IsAcceptedGateway(gateway *gwv1beta1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1beta1.GatewayConditionAccepted))
}

// IsActiveGateway returns true if the gateway is active
func IsActiveGateway(gateway *gwv1beta1.Gateway) bool {
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
func IsListenerProgrammed(listenerStatus gwv1beta1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1beta1.ListenerConditionAccepted))
}

// IsListenerAccepted returns true if the listener is accepted
func IsListenerAccepted(listenerStatus gwv1beta1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1beta1.ListenerConditionAccepted))
}

func IsAcceptedRateLimitPolicy(policy *gwpav1alpha1.RateLimitPolicy) bool {
	return metautil.IsStatusConditionTrue(policy.Status.Conditions, string(gwv1alpha2.PolicyConditionAccepted))
}

func IsAcceptedSessionStickyPolicy(policy *gwpav1alpha1.SessionStickyPolicy) bool {
	return metautil.IsStatusConditionTrue(policy.Status.Conditions, string(gwv1alpha2.PolicyConditionAccepted))
}

// IsRefToGateway returns true if the parent reference is to the gateway
func IsRefToGateway(parentRef gwv1beta1.ParentReference, gateway client.ObjectKey) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gwv1beta1.GroupName {
		return false
	}

	if parentRef.Kind != nil && string(*parentRef.Kind) != constants.GatewayKind {
		return false
	}

	if parentRef.Namespace != nil && string(*parentRef.Namespace) != gateway.Namespace {
		return false
	}

	return string(parentRef.Name) == gateway.Name
}

// IsRefToTarget returns true if the target reference is to the target object
func IsRefToTarget(targetRef gwv1alpha2.PolicyTargetReference, object client.Object) bool {
	gvk := object.GetObjectKind().GroupVersionKind()

	if string(targetRef.Group) != gvk.Group {
		return false
	}

	if string(targetRef.Kind) != gvk.Kind {
		return false
	}

	if targetRef.Namespace != nil && string(*targetRef.Namespace) != object.GetNamespace() {
		return false
	}

	return string(targetRef.Name) == object.GetName()
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
func GroupPointer(group string) *gwv1beta1.Group {
	result := gwv1beta1.Group(group)

	return &result
}

// GetValidListenersFromGateway returns the valid listeners from the gateway
func GetValidListenersFromGateway(gw *gwv1beta1.Gateway) []gwtypes.Listener {
	listeners := make(map[gwv1beta1.SectionName]gwv1beta1.Listener)
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

// GetAllowedListenersAndSetStatus returns the allowed listeners and set status
func GetAllowedListenersAndSetStatus(
	parentRef gwv1beta1.ParentReference,
	routeGvk schema.GroupVersionKind,
	routeGeneration int64,
	validListeners []gwtypes.Listener,
	routeParentStatus gwv1beta1.RouteParentStatus,
) []gwtypes.Listener {
	var selectedListeners []gwtypes.Listener
	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}

	if len(selectedListeners) == 0 {
		metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
			Type:               string(gwv1beta1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: routeGeneration,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1beta1.RouteReasonNoMatchingParent),
			Message:            fmt.Sprintf("No listeners match parent ref %s/%s", *parentRef.Namespace, parentRef.Name),
		})

		return nil
	}

	var allowedListeners []gwtypes.Listener
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(routeGvk) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
			Type:               string(gwv1beta1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: routeGeneration,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1beta1.RouteReasonNotAllowedByListeners),
			Message:            fmt.Sprintf("No matched listeners of parent ref %s/%s", *parentRef.Namespace, parentRef.Name),
		})

		return nil
	}

	return allowedListeners
}

// GetAllowedListeners returns the allowed listeners
func GetAllowedListeners(
	parentRef gwv1beta1.ParentReference,
	routeGvk schema.GroupVersionKind,
	routeGeneration int64,
	validListeners []gwtypes.Listener,
) []gwtypes.Listener {
	var selectedListeners []gwtypes.Listener
	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}

	if len(selectedListeners) == 0 {
		return nil
	}

	var allowedListeners []gwtypes.Listener
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(routeGvk) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		return nil
	}

	return allowedListeners
}

// GetValidHostnames returns the valid hostnames
func GetValidHostnames(listenerHostname *gwv1beta1.Hostname, routeHostnames []gwv1beta1.Hostname) []string {
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

// GetRateLimitIfRouteHostnameMatchesPolicy returns the rate limit config if the route hostname matches the policy
func GetRateLimitIfRouteHostnameMatchesPolicy(routeHostname string, rateLimitPolicy gwpav1alpha1.RateLimitPolicy) *gwpav1alpha1.L7RateLimit {
	if len(rateLimitPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for i := range rateLimitPolicy.Spec.Hostnames {
		hostname := string(rateLimitPolicy.Spec.Hostnames[i].Hostname)
		rateLimit := rateLimitPolicy.Spec.Hostnames[i].RateLimit
		if rateLimit == nil {
			rateLimit = rateLimitPolicy.Spec.DefaultL7RateLimit
		}

		switch {
		case routeHostname == hostname:
			return rateLimit

		case strings.HasPrefix(routeHostname, "*"):
			if HostnameMatchesWildcardHostname(hostname, routeHostname) {
				return rateLimit
			}

		case strings.HasPrefix(hostname, "*"):
			if HostnameMatchesWildcardHostname(routeHostname, hostname) {
				return rateLimit
			}
		}
	}

	return nil
}

// GetRateLimitIfHTTPRouteMatchesPolicy returns the rate limit config if the HTTP route matches the policy
func GetRateLimitIfHTTPRouteMatchesPolicy(routeMatch gwv1beta1.HTTPRouteMatch, rateLimitPolicy gwpav1alpha1.RateLimitPolicy) *gwpav1alpha1.L7RateLimit {
	if len(rateLimitPolicy.Spec.HTTPRateLimits) == 0 {
		return nil
	}

	for _, hr := range rateLimitPolicy.Spec.HTTPRateLimits {
		if reflect.DeepEqual(routeMatch, hr.Match) {
			if hr.RateLimit != nil {
				return hr.RateLimit
			}

			return rateLimitPolicy.Spec.DefaultL7RateLimit
		}
	}

	return nil
}

// GetRateLimitIfGRPCRouteMatchesPolicy returns the rate limit config if the GRPC route matches the policy
func GetRateLimitIfGRPCRouteMatchesPolicy(routeMatch gwv1alpha2.GRPCRouteMatch, rateLimitPolicy gwpav1alpha1.RateLimitPolicy) *gwpav1alpha1.L7RateLimit {
	if len(rateLimitPolicy.Spec.GRPCRateLimits) == 0 {
		return nil
	}

	for _, gr := range rateLimitPolicy.Spec.GRPCRateLimits {
		if reflect.DeepEqual(routeMatch, gr.Match) {
			if gr.RateLimit != nil {
				return gr.RateLimit
			}

			return rateLimitPolicy.Spec.DefaultL7RateLimit
		}
	}

	return nil
}

// GetRateLimitIfPortMatchesPolicy returns true if the port matches the rate limit policy
func GetRateLimitIfPortMatchesPolicy(port gwv1beta1.PortNumber, rateLimitPolicy gwpav1alpha1.RateLimitPolicy) *int64 {
	if len(rateLimitPolicy.Spec.Ports) == 0 {
		return nil
	}

	for _, policyPort := range rateLimitPolicy.Spec.Ports {
		if port == policyPort.Port {
			if policyPort.BPS != nil {
				return policyPort.BPS
			}

			return rateLimitPolicy.Spec.DefaultBPS
		}
	}

	return nil
}

// SessionStickyPolicyMatchesService returns true if the service matches the policy
func SessionStickyPolicyMatchesService(policy *gwpav1alpha1.SessionStickyPolicy, svc *corev1.Service) bool {
	if !IsRefToTarget(policy.Spec.TargetRef, svc) {
		return false
	}

	for _, p := range svc.Spec.Ports {
		if int32(policy.Spec.Port) == p.Port {
			return true
		}
	}

	return false
}

// SessionStickyPolicyMatchesServiceImport returns true if the ServiceImport matches the policy
func SessionStickyPolicyMatchesServiceImport(policy *gwpav1alpha1.SessionStickyPolicy, svcimp *mcsv1alpha1.ServiceImport) bool {
	if !IsRefToTarget(policy.Spec.TargetRef, svcimp) {
		return false
	}

	for _, p := range svcimp.Spec.Ports {
		if int32(policy.Spec.Port) == p.Port {
			return true
		}
	}

	return false
}
