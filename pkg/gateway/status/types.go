// Package status implements utility routines related to the status of the Gateway API resource.
package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// MetadataAccessor is an interface to access metadata of a resource.
type MetadataAccessor interface {
	GetObjectMeta() *metav1.ObjectMeta
	GetTypeMeta() *metav1.TypeMeta
	GetResource() client.Object
	GetTransitionTime() metav1.Time
	GetFullName() types.NamespacedName
	GetGeneration() int64
}

// RouteConditionAccessor is an interface to access conditions of a Route.
type RouteConditionAccessor interface {
	ConditionExists(conditionType gwv1.RouteConditionType) bool
	AddCondition(conditionType gwv1.RouteConditionType, status metav1.ConditionStatus, reason gwv1.RouteConditionReason, message string) metav1.Condition
}

// RouteStatusObject is an interface to access the status of a Route.
type RouteStatusObject interface {
	Mutator
	MetadataAccessor
	GetRouteParentStatuses() []*gwv1.RouteParentStatus
	GetHostnames() []gwv1.Hostname
	StatusUpdateFor(parentRef gwv1.ParentReference) RouteParentStatusObject
	ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition
}

// RouteParentStatusObject is an interface to access the status of a RouteParent.
type RouteParentStatusObject interface {
	RouteConditionAccessor
	GetRouteStatusObject() RouteStatusObject
	GetParentRef() gwv1.ParentReference
}

// PolicyConditionAccessor is an interface to access conditions of a Policy.
type PolicyConditionAccessor interface {
	ConditionExists(conditionType gwv1alpha2.PolicyConditionType) bool
	AddCondition(conditionType gwv1alpha2.PolicyConditionType, status metav1.ConditionStatus, reason gwv1alpha2.PolicyConditionReason, message string) metav1.Condition
}

// PolicyStatusObject is an interface to access the status of a Policy.
type PolicyStatusObject interface {
	Mutator
	MetadataAccessor
	StatusUpdateFor(parentRef gwv1.ParentReference) PolicyAncestorStatusObject
	ConditionsForAncestorRef(parentRef gwv1.ParentReference) []metav1.Condition
}

// PolicyAncestorStatusObject is an interface to access the status of a Policy Ancestor.
type PolicyAncestorStatusObject interface {
	PolicyConditionAccessor
	GetPolicyStatusObject() PolicyStatusObject
	GetAncestorRef() gwv1.ParentReference
}

//var (
//	log = logger.NewPretty("fsm-gateway/status")
//)
