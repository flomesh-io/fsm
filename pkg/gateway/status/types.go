// Package status implements utility routines related to the status of the Gateway API resource.
package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

//var (
//	log = logger.New("fsm-gateway/routestatus")
//)

type RouteStatusObject interface {
	Mutator
	GetObjectMeta() *metav1.ObjectMeta
	GetTypeMeta() *metav1.TypeMeta
	GetRouteParentStatuses() []*gwv1.RouteParentStatus
	GetHostnames() []gwv1.Hostname
	GetResource() client.Object
	GetTransitionTime() metav1.Time
	GetFullName() types.NamespacedName
	GetGeneration() int64
	StatusUpdateFor(parentRef gwv1.ParentReference) RouteParentStatusObject
}

type RouteParentStatusObject interface {
	GetRouteStatusObject() RouteStatusObject
	GetParentRef() gwv1.ParentReference
	AddCondition(conditionType gwv1.RouteConditionType, status metav1.ConditionStatus, reason gwv1.RouteConditionReason, message string) metav1.Condition
	ConditionExists(conditionType gwv1.RouteConditionType) bool
	ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition
}
