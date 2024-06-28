// Package status implements utility routines related to the status of the Gateway API resource.
package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type MetadataAccessor interface {
	GetObjectMeta() *metav1.ObjectMeta
	GetTypeMeta() *metav1.TypeMeta
	GetResource() client.Object
	GetTransitionTime() metav1.Time
	GetFullName() types.NamespacedName
	GetGeneration() int64
}

type RouteConditionAccessor interface {
	ConditionExists(conditionType gwv1.RouteConditionType) bool
	AddCondition(conditionType gwv1.RouteConditionType, status metav1.ConditionStatus, reason gwv1.RouteConditionReason, message string) metav1.Condition
}

type RouteStatusObject interface {
	Mutator
	MetadataAccessor
	GetRouteParentStatuses() []*gwv1.RouteParentStatus
	GetHostnames() []gwv1.Hostname
	StatusUpdateFor(parentRef gwv1.ParentReference) RouteParentStatusObject
	ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition
}

type RouteParentStatusObject interface {
	RouteConditionAccessor
	GetRouteStatusObject() RouteStatusObject
	GetParentRef() gwv1.ParentReference
}

type PolicyConditionAccessor interface {
	ConditionExists(conditionType gwv1alpha2.PolicyConditionType) bool
	AddCondition(conditionType gwv1alpha2.PolicyConditionType, status metav1.ConditionStatus, reason gwv1alpha2.PolicyConditionReason, message string) metav1.Condition
}

type PolicyStatusObject interface {
	Mutator
	MetadataAccessor
	StatusUpdateFor(parentRef gwv1.ParentReference) PolicyAncestorStatusObject
	ConditionsForAncestorRef(parentRef gwv1.ParentReference) []metav1.Condition
}

type PolicyAncestorStatusObject interface {
	PolicyConditionAccessor
	GetPolicyStatusObject() PolicyStatusObject
	GetAncestorRef() gwv1.ParentReference
}
