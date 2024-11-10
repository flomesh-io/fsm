package v1

import (
	"github.com/flomesh-io/fsm/pkg/gateway/status/gw"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (r *gatewayReconciler) addInvalidListenerCondition(gateway *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, name gwv1.SectionName, cond metav1.Condition) {
	defer r.recorder.Eventf(gateway, corev1.EventTypeWarning, cond.Reason, cond.Message)

	gsu.AddListenerCondition(
		string(name),
		gwv1.ListenerConditionType(cond.Type),
		cond.Status,
		gwv1.ListenerConditionReason(cond.Reason),
		cond.Message,
	)
}

func (r *gatewayReconciler) addListenerNotProgrammedCondition(gateway *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, name gwv1.SectionName, reason gwv1.ListenerConditionReason, msg string) {
	defer r.recorder.Eventf(gateway, corev1.EventTypeWarning, string(reason), msg)

	gsu.AddListenerCondition(
		string(name),
		gwv1.ListenerConditionProgrammed,
		metav1.ConditionFalse,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addGatewayNotProgrammedCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, reason gwv1.GatewayConditionReason, msg string) {
	defer r.recorder.Eventf(gw, corev1.EventTypeWarning, string(reason), msg)

	gsu.AddCondition(
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionFalse,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addGatewayProgrammedCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, reason gwv1.GatewayConditionReason, msg string) {
	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, string(reason), msg)

	gsu.AddCondition(
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionTrue,
		reason,
		msg,
	)
}

func (r *gatewayReconciler) addGatewayNotAcceptedCondition(gw *gwv1.Gateway, gsu *gw.GatewayStatusUpdate, reason gwv1.GatewayConditionReason, msg string) {
	defer r.recorder.Eventf(gw, corev1.EventTypeWarning, string(reason), msg)

	gsu.AddCondition(
		gwv1.GatewayConditionAccepted,
		metav1.ConditionFalse,
		reason,
		msg,
	)
}
