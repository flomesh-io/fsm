package route

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/constants"
)

// --- DefaultRouteStatusObject ---

type DefaultRouteStatusObject struct {
	objectMeta          *metav1.ObjectMeta
	typeMeta            *metav1.TypeMeta
	routeParentStatuses []*gwv1.RouteParentStatus
	hostnames           []gwv1.Hostname
	resource            client.Object
	transitionTime      metav1.Time
	fullName            types.NamespacedName
	generation          int64
}

func (r *DefaultRouteStatusObject) GetObjectMeta() *metav1.ObjectMeta {
	return r.objectMeta
}

func (r *DefaultRouteStatusObject) GetTypeMeta() *metav1.TypeMeta {
	return r.typeMeta
}

func (r *DefaultRouteStatusObject) GetRouteParentStatuses() []*gwv1.RouteParentStatus {
	return r.routeParentStatuses
}

func (r *DefaultRouteStatusObject) GetHostnames() []gwv1.Hostname {
	return r.hostnames
}

func (r *DefaultRouteStatusObject) GetResource() client.Object {
	return r.resource
}

func (r *DefaultRouteStatusObject) GetTransitionTime() metav1.Time {
	return r.transitionTime
}

func (r *DefaultRouteStatusObject) GetFullName() types.NamespacedName {
	return r.fullName
}

func (r *DefaultRouteStatusObject) GetGeneration() int64 {
	return r.generation
}

func (r *DefaultRouteStatusObject) Mutate(obj client.Object) client.Object {
	return obj
}

func (r *DefaultRouteStatusObject) StatusUpdateFor(parentRef gwv1.ParentReference) status.RouteParentStatusObject {
	return &DefaultRouteParentStatusObject{
		DefaultRouteStatusObject: r,
		ParentRef:                parentRef,
	}
}

func (r *DefaultRouteStatusObject) ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition {
	for _, rps := range r.routeParentStatuses {
		if cmp.Equal(rps.ParentRef, parentRef) {
			return rps.Conditions
		}
	}

	return nil
}

// --- DefaultRouteParentStatusObject ---

type DefaultRouteParentStatusObject struct {
	*DefaultRouteStatusObject
	ParentRef gwv1.ParentReference
}

func (r *DefaultRouteParentStatusObject) GetRouteStatusObject() status.RouteStatusObject {
	return r.DefaultRouteStatusObject
}

func (r *DefaultRouteParentStatusObject) GetParentRef() gwv1.ParentReference {
	return r.ParentRef
}

func (r *DefaultRouteParentStatusObject) AddCondition(conditionType gwv1.RouteConditionType, status metav1.ConditionStatus, reason gwv1.RouteConditionReason, message string) metav1.Condition {
	var rps *gwv1.RouteParentStatus

	for _, v := range r.routeParentStatuses {
		if cmp.Equal(v.ParentRef, r.ParentRef) {
			rps = v
			break
		}
	}

	if rps == nil {
		rps = &gwv1.RouteParentStatus{
			ParentRef:      r.ParentRef,
			ControllerName: constants.GatewayController,
		}

		r.routeParentStatuses = append(r.routeParentStatuses, rps)
	}

	//msg := message
	//if cond := metautil.FindStatusCondition(rps.Conditions, string(conditionType)); cond != nil {
	//	msg = cond.Message + ", " + message
	//}

	cond := metav1.Condition{
		Reason:             string(reason),
		Status:             status,
		Type:               string(conditionType),
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: r.generation,
	}

	metautil.SetStatusCondition(&rps.Conditions, cond)

	return cond
}

func (r *DefaultRouteParentStatusObject) ConditionExists(conditionType gwv1.RouteConditionType) bool {
	for _, c := range r.ConditionsForParentRef(r.ParentRef) {
		if c.Type == string(conditionType) {
			return true
		}
	}
	return false
}

// --- RouteStatusUpdate ---

type RouteStatusUpdate struct {
	*DefaultRouteStatusObject
}

func NewRouteStatusUpdate(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, hostnames []gwv1.Hostname, routeParentStatuses []*gwv1.RouteParentStatus) status.RouteStatusObject {
	return &RouteStatusUpdate{
		DefaultRouteStatusObject: &DefaultRouteStatusObject{
			objectMeta:          meta,
			typeMeta:            typeMeta,
			routeParentStatuses: routeParentStatuses,
			resource:            resource,
			hostnames:           hostnames,
			transitionTime:      metav1.Time{Time: time.Now()},
			fullName:            types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
			generation:          meta.Generation,
		},
	}
}

func (r *RouteStatusUpdate) Mutate(obj client.Object) client.Object {
	var newRouteParentStatuses []gwv1.RouteParentStatus

	for _, rps := range r.routeParentStatuses {
		for i := range rps.Conditions {
			cond := &rps.Conditions[i]

			cond.ObservedGeneration = r.generation
			cond.LastTransitionTime = r.transitionTime
		}

		newRouteParentStatuses = append(newRouteParentStatuses, *rps)
	}

	switch o := obj.(type) {
	case *gwv1.HTTPRoute:
		route := o.DeepCopy()
		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1.GRPCRoute:
		route := o.DeepCopy()
		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.TLSRoute:
		route := o.DeepCopy()
		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.TCPRoute:
		route := o.DeepCopy()
		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.UDPRoute:
		route := o.DeepCopy()
		route.Status.Parents = newRouteParentStatuses

		return route
	default:
		panic(fmt.Sprintf("Unsupported %T object %s/%s in RouteConditionsUpdate status mutator", obj, r.fullName.Namespace, r.fullName.Name))
	}
}

// --- RouteStatusHolder ---

type RouteStatusHolder struct {
	*DefaultRouteStatusObject
}

func NewRouteStatusHolder(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, hostnames []gwv1.Hostname, routeParentStatuses []*gwv1.RouteParentStatus) status.RouteStatusObject {
	return &RouteStatusHolder{
		DefaultRouteStatusObject: &DefaultRouteStatusObject{objectMeta: meta,
			typeMeta:            typeMeta,
			routeParentStatuses: routeParentStatuses,
			resource:            resource,
			hostnames:           hostnames,
			transitionTime:      metav1.Time{Time: time.Now()},
			fullName:            types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
			generation:          meta.Generation,
		},
	}
}

func (r *RouteStatusHolder) StatusUpdateFor(parentRef gwv1.ParentReference) status.RouteParentStatusObject {
	return &RouteParentStatusHolder{
		DefaultRouteParentStatusObject: &DefaultRouteParentStatusObject{
			DefaultRouteStatusObject: r.DefaultRouteStatusObject,
			ParentRef:                parentRef,
		},
	}
}

// --- RouteParentStatusHolder ---

type RouteParentStatusHolder struct {
	*DefaultRouteParentStatusObject
}

func (r *RouteParentStatusHolder) AddCondition(_ gwv1.RouteConditionType, _ metav1.ConditionStatus, _ gwv1.RouteConditionReason, _ string) metav1.Condition {
	return metav1.Condition{}
}
