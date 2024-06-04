package v1

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
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

func gatewayProgrammedCondition(gw *gwv1.Gateway, deployment *appsv1.Deployment) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionProgrammed),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gw.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayConditionProgrammed),
		Message:            fmt.Sprintf("Address assigned to the Gateway, %d/%d Deployment replicas available", deployment.Status.AvailableReplicas, deployment.Status.Replicas),
	}
}

func gatewayNoResourcesCondition(gw *gwv1.Gateway) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionProgrammed),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gw.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayReasonNoResources),
		Message:            "Deployment replicas unavailable",
	}
}

func gatewayAddressNotAssignedCondition(gw *gwv1.Gateway) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionProgrammed),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gw.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayReasonAddressNotAssigned),
		Message:            "No addresses have been assigned to the Gateway",
	}
}

func gatewayAcceptedCondition(gateway *gwv1.Gateway) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayReasonAccepted),
		Message:            fmt.Sprintf("Gateway %s/%s is accepted.", gateway.Namespace, gateway.Name),
	}
}

func gatewayUnacceptedCondition(gateway *gwv1.Gateway) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Unaccepted",
		Message:            fmt.Sprintf("Gateway %s/%s is not accepted as it's not the oldest one in namespace %q.", gateway.Namespace, gateway.Name, gateway.Namespace),
	}
}
