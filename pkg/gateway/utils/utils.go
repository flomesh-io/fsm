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

package utils

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/apis/gateway"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway"
	"github.com/gobwas/glob"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"strings"
	"time"
)

func IsAcceptedGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gwv1beta1.GatewayClassConditionStatusAccepted))
}

func IsActiveGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gateway.GatewayClassConditionStatusActive))
}

func IsEffectiveGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return IsAcceptedGatewayClass(gatewayClass) && IsActiveGatewayClass(gatewayClass)
}

func IsAcceptedGateway(gateway *gwv1beta1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1beta1.GatewayConditionAccepted))
}

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

func IsListenerProgrammed(listenerStatus gwv1beta1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1beta1.ListenerConditionAccepted))
}

func IsListenerAccepted(listenerStatus gwv1beta1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1beta1.ListenerConditionAccepted))
}

func IsRefToGateway(parentRef gwv1beta1.ParentReference, gateway client.ObjectKey) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gwv1beta1.GroupName {
		return false
	}

	if parentRef.Kind != nil && string(*parentRef.Kind) != "Gateway" {
		return false
	}

	if parentRef.Namespace != nil && string(*parentRef.Namespace) != gateway.Namespace {
		return false
	}

	return string(parentRef.Name) == gateway.Name
}

func ObjectKey(obj client.Object) client.ObjectKey {
	ns := obj.GetNamespace()
	if ns == "" {
		ns = metav1.NamespaceDefault
	}

	return client.ObjectKey{Namespace: ns, Name: obj.GetName()}
}

func GroupPointer(group string) *gwv1beta1.Group {
	result := gwv1beta1.Group(group)

	return &result
}

func GetValidListenersFromGateway(gw *gwv1beta1.Gateway) []gwpkg.Listener {
	listeners := make(map[gwv1beta1.SectionName]gwv1beta1.Listener)
	for _, listener := range gw.Spec.Listeners {
		listeners[listener.Name] = listener
	}

	validListeners := make([]gwpkg.Listener, 0)
	for _, status := range gw.Status.Listeners {
		if IsListenerAccepted(status) && IsListenerProgrammed(status) {
			l, ok := listeners[status.Name]
			if !ok {
				continue
			}
			validListeners = append(validListeners, gwpkg.Listener{
				Listener:       l,
				SupportedKinds: status.SupportedKinds,
			})
		}
	}

	return validListeners
}

func GetAllowedListeners(
	parentRef gwv1beta1.ParentReference,
	routeGvk schema.GroupVersionKind,
	routeGeneration int64,
	validListeners []gwpkg.Listener,
	routeParentStatus gwv1beta1.RouteParentStatus,
) []gwpkg.Listener {
	var selectedListeners []gwpkg.Listener
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

	var allowedListeners []gwpkg.Listener
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

func HostnameMatchesWildcardHostname(hostname, wildcardHostname string) bool {
	g := glob.MustCompile(wildcardHostname, '.')
	return g.Match(hostname)
}
