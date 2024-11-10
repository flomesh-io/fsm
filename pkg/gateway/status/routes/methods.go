package routes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

func (p *RouteStatusProcessor) addNotAcceptedCondition(route client.Object, rps status.RouteConditionAccessor, reason gwv1.RouteConditionReason, message string) {
	defer p.recorder.Eventf(route, corev1.EventTypeWarning, string(reason), message)

	rps.AddCondition(
		gwv1.RouteConditionAccepted,
		metav1.ConditionFalse,
		reason,
		message,
	)
}

func (p *RouteStatusProcessor) addNotResolvedRefsCondition(route client.Object, rps status.RouteConditionAccessor, reason gwv1.RouteConditionReason, message string) {
	defer p.recorder.Eventf(route, corev1.EventTypeWarning, string(reason), message)

	rps.AddCondition(
		gwv1.RouteConditionResolvedRefs,
		metav1.ConditionFalse,
		reason,
		message,
	)
}

func (p *RouteStatusProcessor) addResolvedRefsCondition(route client.Object, rps status.RouteConditionAccessor, reason gwv1.RouteConditionReason, message string) {
	defer p.recorder.Eventf(route, corev1.EventTypeNormal, string(reason), message)

	rps.AddCondition(
		gwv1.RouteConditionResolvedRefs,
		metav1.ConditionTrue,
		reason,
		message,
	)
}
