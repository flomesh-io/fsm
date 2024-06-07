package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

// IsAcceptedGateway returns true if the gateway is accepted
func IsAcceptedGateway(gateway *gwv1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1.GatewayConditionAccepted))
}

// IsProgrammedGateway returns true if the gateway is programmed
func IsProgrammedGateway(gateway *gwv1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1.GatewayConditionProgrammed))
}

// IsActiveGateway returns true if the gateway is active, it stands for the gateway is accepted, programmed and has a valid listener
func IsActiveGateway(gateway *gwv1.Gateway) bool {
	return IsAcceptedGateway(gateway) && IsProgrammedGateway(gateway)
}

// IsListenerProgrammed returns true if the listener is programmed
func IsListenerProgrammed(listenerStatus gwv1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1.ListenerConditionProgrammed))
}

// IsListenerAccepted returns true if the listener is accepted
func IsListenerAccepted(listenerStatus gwv1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1.ListenerConditionAccepted))
}

// IsListenerConflicted returns true if the listener is conflicted
func IsListenerConflicted(listenerStatus gwv1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1.ListenerConditionConflicted))
}

// IsListenerResolvedRefs returns true if the listener is resolved refs
func IsListenerResolvedRefs(listenerStatus gwv1.ListenerStatus) bool {
	return metautil.IsStatusConditionTrue(listenerStatus.Conditions, string(gwv1.ListenerConditionResolvedRefs))
}

// IsListenerValid returns true if the listener is valid
func IsListenerValid(s gwv1.ListenerStatus) bool {
	return IsListenerAccepted(s) && IsListenerProgrammed(s) && !IsListenerConflicted(s) && IsListenerResolvedRefs(s)
}

func GetActiveGateways(cache cache.Cache) []*gwv1.Gateway {
	classes, err := findFSMGatewayClasses(cache)
	if err != nil {
		log.Error().Msgf("Failed to find GatewayClass: %v", err)
		return nil
	}

	gateways := make([]*gwv1.Gateway, 0)
	for _, cls := range classes {
		list := &gwv1.GatewayList{}
		if err := cache.List(context.Background(), list, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, cls.Name),
		}); err != nil {
			log.Error().Msgf("Failed to list Gateways: %v", err)
			continue
		}

		gateways = append(gateways, ToSlicePtr(list.Items)...)
	}

	return filterActiveGateways(gateways)
}

// filterActiveGateways returns the active gateways from the list of gateways
func filterActiveGateways(allGateways []*gwv1.Gateway) []*gwv1.Gateway {
	gateways := make([]*gwv1.Gateway, 0)

	for _, gw := range allGateways {
		if IsActiveGateway(gw) {
			gateways = append(gateways, gw)
		}
	}

	return gateways
}

// GetValidListenersForGateway returns the valid listeners from the gateway
func GetValidListenersForGateway(gw *gwv1.Gateway) []gwtypes.Listener {
	listeners := make(map[gwv1.SectionName]gwv1.Listener)
	for _, listener := range gw.Spec.Listeners {
		listeners[listener.Name] = listener
	}

	validListeners := make([]gwtypes.Listener, 0)
	for _, s := range gw.Status.Listeners {
		if IsListenerValid(s) {
			l, ok := listeners[s.Name]
			if !ok {
				continue
			}
			validListeners = append(validListeners, gwtypes.Listener{
				Listener:       l,
				SupportedKinds: s.SupportedKinds,
			})
		}
	}

	return validListeners
}

// GetAllowedListeners returns the allowed listeners
func GetAllowedListeners(client cache.Cache, gw *gwv1.Gateway, u status.RouteParentStatusObject) []gwtypes.Listener {
	routeGvk := u.GetRouteStatusObject().GetTypeMeta().GroupVersionKind()
	routeNs := u.GetRouteStatusObject().GetObjectMeta().Namespace
	parentRef := u.GetParentRef()
	validListeners := GetValidListenersForGateway(gw)

	selectedListeners := make([]gwtypes.Listener, 0)
	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}

	if len(selectedListeners) == 0 {
		u.AddCondition(
			gwv1.RouteConditionAccepted,
			metav1.ConditionFalse,
			gwv1.RouteReasonNoMatchingParent,
			fmt.Sprintf("No listeners match parent ref %s", types.NamespacedName{Namespace: NamespaceDerefOr(parentRef.Namespace, routeNs), Name: string(parentRef.Name)}),
		)

		return nil
	}

	allowedListeners := make([]gwtypes.Listener, 0)
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(routeGvk) {
			continue
		}

		// Check if the route is in a namespace that the listener allows.
		if !NamespaceMatches(client, selectedListener.AllowedRoutes.Namespaces, gw.Namespace, routeNs) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		u.AddCondition(
			gwv1.RouteConditionAccepted,
			metav1.ConditionFalse,
			gwv1.RouteReasonNotAllowedByListeners,
			fmt.Sprintf("No matched listeners of parent ref %s", types.NamespacedName{Namespace: NamespaceDerefOr(parentRef.Namespace, routeNs), Name: string(parentRef.Name)}),
		)

		return nil
	}

	return allowedListeners
}

// NamespaceMatches returns true if the namespace matches
func NamespaceMatches(client cache.Cache, namespaces *gwv1.RouteNamespaces, gatewayNamespace, routeNamespace string) bool {
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

		ns := &corev1.Namespace{}
		if err := client.Get(context.Background(), types.NamespacedName{Name: routeNamespace}, ns); err != nil {
			log.Error().Msgf("failed to get namespace %s: %v", routeNamespace, err)
			return false
		}

		return namespaceSelector.Matches(labels.Set(ns.Labels))
	}

	return true
}
