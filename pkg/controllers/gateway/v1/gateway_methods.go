package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status/gw"
)

func (r *gatewayReconciler) addInvalidListenerCondition(gateway *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, name gwv1.SectionName, cond metav1.Condition) {
	r.addListenerCondition(
		gateway,
		gsu,
		name,
		corev1.EventTypeWarning,
		gwv1.ListenerConditionType(cond.Type),
		cond.Status,
		gwv1.ListenerConditionReason(cond.Reason),
		cond.Message,
	)
}

func (r *gatewayReconciler) addListenerNotProgrammedCondition(gateway *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, name gwv1.SectionName, reason gwv1.ListenerConditionReason, msg string) {
	r.addListenerCondition(
		gateway,
		gsu,
		name,
		corev1.EventTypeWarning,
		gwv1.ListenerConditionProgrammed,
		metav1.ConditionFalse,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addGatewayNotProgrammedCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, reason gwv1.GatewayConditionReason, msg string) {
	r.addCondition(
		gw,
		gsu,
		corev1.EventTypeWarning,
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionFalse,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addGatewayProgrammedCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, reason gwv1.GatewayConditionReason, msg string) {
	r.addCondition(
		gw,
		gsu,
		corev1.EventTypeNormal,
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionTrue,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addGatewayNotAcceptedCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, reason gwv1.GatewayConditionReason, msg string) {
	r.addCondition(
		gw,
		gsu,
		corev1.EventTypeWarning,
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, eventType string, conditionType gwv1.GatewayConditionType, status metav1.ConditionStatus, reason gwv1.GatewayConditionReason, message string) {
	gsu.AddCondition(
		conditionType,
		status,
		reason,
		message,
	)

	r.recorder.Event(gw, eventType, string(reason), message)
}

func (r *gatewayReconciler) addListenerCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, name gwv1.SectionName, eventType string, conditionType gwv1.ListenerConditionType, status metav1.ConditionStatus, reason gwv1.ListenerConditionReason, message string) {
	gsu.AddListenerCondition(
		string(name),
		conditionType,
		status,
		reason,
		message,
	)

	r.recorder.Event(gw, eventType, string(reason), message)
}
