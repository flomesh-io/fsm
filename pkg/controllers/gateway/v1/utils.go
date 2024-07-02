package v1

import (
	"fmt"

	"github.com/flomesh-io/fsm/pkg/gateway/status/gw"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func hasTCP(gateway *gwv1.Gateway) bool {
	for _, listener := range gateway.Spec.Listeners {
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.TCPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
			return true
		}
	}

	return false
}

func hasUDP(gateway *gwv1.Gateway) bool {
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol == gwv1.UDPProtocolType {
			return true
		}
	}

	return false
}

func invalidateListeners(listeners []gwv1.Listener) map[gwv1.SectionName]metav1.Condition {
	conflictCondition := func(msg string) metav1.Condition {
		return metav1.Condition{
			Type:    string(gwv1.ListenerConditionConflicted),
			Status:  metav1.ConditionTrue,
			Reason:  string(gwv1.ListenerReasonHostnameConflict),
			Message: msg,
		}
	}

	invalidListenerConditions := map[gwv1.SectionName]metav1.Condition{}

	for i, listener := range listeners {
		// Check for a valid hostname.
		if hostname := ptr.Deref(listener.Hostname, ""); len(hostname) > 0 {
			if err := gwutils.IsValidHostname(string(hostname)); err != nil {
				invalidListenerConditions[listener.Name] = metav1.Condition{
					Type:    string(gwv1.ListenerConditionProgrammed),
					Status:  metav1.ConditionFalse,
					Reason:  string(gwv1.ListenerReasonInvalid),
					Message: fmt.Sprintf("Invalid hostname %q: %v", hostname, err),
				}
				continue
			}
		}

		// Check for a supported protocol.
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType, gwv1.TCPProtocolType, gwv1.UDPProtocolType:
		default:
			invalidListenerConditions[listener.Name] = conflictCondition(fmt.Sprintf("Listener protocol %q is unsupported, must be one of HTTP, HTTPS, TLS, TCP or UDP", listener.Protocol))
			continue
		}

		if constants.ReservedGatewayPorts.Has(int32(listener.Port)) {
			invalidListenerConditions[listener.Name] = conflictCondition(fmt.Sprintf("Listener port %d is reserved, please use other port instead", listener.Port))
			continue
		}

		func() {
			for j := range i {
				otherListener := listeners[j]

				if listener.Port != otherListener.Port {
					continue
				}

				if listener.Protocol != otherListener.Protocol {
					// same port, different protocol, not allowed
					invalidListenerConditions[listener.Name] = metav1.Condition{
						Type:    string(gwv1.ListenerConditionConflicted),
						Status:  metav1.ConditionTrue,
						Reason:  string(gwv1.ListenerReasonProtocolConflict),
						Message: "All Listener protocols for a given port must be the same",
					}

					return
				}

				switch listener.Protocol {
				case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
					// Hostname conflict
					if ptr.Deref(listener.Hostname, "") == ptr.Deref(otherListener.Hostname, "") {
						invalidListenerConditions[listener.Name] = metav1.Condition{
							Type:    string(gwv1.ListenerConditionConflicted),
							Status:  metav1.ConditionTrue,
							Reason:  string(gwv1.ListenerReasonHostnameConflict),
							Message: "All Listener hostnames for a given port must be unique",
						}
						return
					}
				}
			}
		}()
	}

	return invalidListenerConditions
}

func supportedRouteGroupKinds(_ *gwv1.Gateway, listener gwv1.Listener, update *gw.GatewayStatusUpdate) []gwv1.RouteGroupKind {
	if len(listener.AllowedRoutes.Kinds) == 0 {
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPIHTTPRouteKind,
				},
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPIGRPCRouteKind,
				},
			}
		case gwv1.TLSProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPITLSRouteKind,
				},
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPITCPRouteKind,
				},
			}
		case gwv1.TCPProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPITCPRouteKind,
				},
			}
		case gwv1.UDPProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPIUDPRouteKind,
				},
			}
		}
	}

	kinds := make([]gwv1.RouteGroupKind, 0)

	for _, routeKind := range listener.AllowedRoutes.Kinds {
		if routeKind.Group != nil && *routeKind.Group != constants.GatewayAPIGroup {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("Group %q is not supported, group must be %q", *routeKind.Group, gwv1.GroupName),
			)

			continue
		}

		if routeKind.Kind != constants.GatewayAPIHTTPRouteKind &&
			routeKind.Kind != constants.GatewayAPITLSRouteKind &&
			routeKind.Kind != constants.GatewayAPIGRPCRouteKind &&
			routeKind.Kind != constants.GatewayAPITCPRouteKind &&
			routeKind.Kind != constants.GatewayAPIUDPRouteKind {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("Kind %q is not supported, kind must be %q, %q, %q, %q or %q", routeKind.Kind, constants.GatewayAPIHTTPRouteKind, constants.GatewayAPIGRPCRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPITCPRouteKind, constants.GatewayAPIUDPRouteKind),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPIHTTPRouteKind && listener.Protocol != gwv1.HTTPProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("HTTPRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPIGRPCRouteKind && listener.Protocol != gwv1.HTTPProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("GRPCRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPITLSRouteKind && listener.Protocol != gwv1.TLSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("TLSRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPITCPRouteKind && listener.Protocol != gwv1.TCPProtocolType && listener.Protocol != gwv1.TLSProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("TCPRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		if routeKind.Kind == constants.GatewayAPIUDPRouteKind && listener.Protocol != gwv1.UDPProtocolType {
			update.AddListenerCondition(
				string(listener.Name),
				gwv1.ListenerConditionResolvedRefs,
				metav1.ConditionFalse,
				gwv1.ListenerReasonInvalidRouteKinds,
				fmt.Sprintf("UDPRoutes are incompatible with listener protocol %q", listener.Protocol),
			)
			continue
		}

		kinds = append(kinds, gwv1.RouteGroupKind{
			Group: routeKind.Group,
			Kind:  routeKind.Kind,
		})
	}

	return kinds
}

func gatewayServiceName(activeGateway *gwv1.Gateway) string {
	if hasTCP(activeGateway) {
		return fmt.Sprintf("fsm-gateway-%s-%s-tcp", activeGateway.Namespace, activeGateway.Name)
	}

	if hasUDP(activeGateway) {
		return fmt.Sprintf("fsm-gateway-%s-%s-udp", activeGateway.Namespace, activeGateway.Name)
	}

	return ""
}
