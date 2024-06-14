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

type RouteStatusUpdate struct {
	objectMeta          *metav1.ObjectMeta
	typeMeta            *metav1.TypeMeta
	routeParentStatuses []*gwv1.RouteParentStatus
	hostnames           []gwv1.Hostname
	resource            client.Object
	transitionTime      metav1.Time
	fullName            types.NamespacedName
	generation          int64
}

func (r *RouteStatusUpdate) GetObjectMeta() *metav1.ObjectMeta {
	return r.objectMeta
}

func (r *RouteStatusUpdate) GetTypeMeta() *metav1.TypeMeta {
	return r.typeMeta
}

func (r *RouteStatusUpdate) GetRouteParentStatuses() []*gwv1.RouteParentStatus {
	return r.routeParentStatuses
}

func (r *RouteStatusUpdate) GetHostnames() []gwv1.Hostname {
	return r.hostnames
}

func (r *RouteStatusUpdate) GetResource() client.Object {
	return r.resource
}

func (r *RouteStatusUpdate) GetTransitionTime() metav1.Time {
	return r.transitionTime
}

func (r *RouteStatusUpdate) GetFullName() types.NamespacedName {
	return r.fullName
}

func (r *RouteStatusUpdate) GetGeneration() int64 {
	return r.generation
}

func NewRouteStatusUpdate(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, hostnames []gwv1.Hostname, routeParentStatuses []*gwv1.RouteParentStatus) status.RouteStatusObject {
	return &RouteStatusUpdate{
		objectMeta:          meta,
		typeMeta:            typeMeta,
		routeParentStatuses: routeParentStatuses,
		resource:            resource,
		hostnames:           hostnames,
		transitionTime:      metav1.Time{Time: time.Now()},
		fullName:            types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
		generation:          meta.Generation,
	}
}

type RouteParentStatusUpdate struct {
	*RouteStatusUpdate
	ParentRef gwv1.ParentReference
}

func (r *RouteParentStatusUpdate) GetRouteStatusObject() status.RouteStatusObject {
	return r.RouteStatusUpdate
}

func (r *RouteParentStatusUpdate) GetParentRef() gwv1.ParentReference {
	return r.ParentRef
}

func (r *RouteStatusUpdate) StatusUpdateFor(parentRef gwv1.ParentReference) status.RouteParentStatusObject {
	return &RouteParentStatusUpdate{
		RouteStatusUpdate: r,
		ParentRef:         parentRef,
	}
}

func (r *RouteParentStatusUpdate) AddCondition(conditionType gwv1.RouteConditionType, status metav1.ConditionStatus, reason gwv1.RouteConditionReason, message string) metav1.Condition {
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

func (r *RouteParentStatusUpdate) ConditionExists(conditionType gwv1.RouteConditionType) bool {
	for _, c := range r.ConditionsForParentRef(r.ParentRef) {
		if c.Type == string(conditionType) {
			return true
		}
	}
	return false
}

func (r *RouteStatusUpdate) ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition {
	for _, rps := range r.routeParentStatuses {
		if cmp.Equal(rps.ParentRef, parentRef) {
			return rps.Conditions
		}
	}

	return nil
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

		// Get all the RouteParentStatuses that are for other Gateways.
		//for _, rps := range o.Status.Parents {
		//if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
		//newRouteParentStatuses = append(newRouteParentStatuses, rps)
		//}
		//}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1.GRPCRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		//for _, rps := range o.Status.Parents {
		//	//if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
		//	newRouteParentStatuses = append(newRouteParentStatuses, rps)
		//	//}
		//}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.TLSRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		//for _, rps := range o.Status.Parents {
		//	//if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
		//	newRouteParentStatuses = append(newRouteParentStatuses, rps)
		//	//}
		//}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.TCPRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		//for _, rps := range o.Status.Parents {
		//	//if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
		//	newRouteParentStatuses = append(newRouteParentStatuses, rps)
		//	//}
		//}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.UDPRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		//for _, rps := range o.Status.Parents {
		//	//if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
		//	newRouteParentStatuses = append(newRouteParentStatuses, rps)
		//	//}
		//}

		route.Status.Parents = newRouteParentStatuses

		return route

	default:
		panic(fmt.Sprintf("Unsupported %T object %s/%s in RouteConditionsUpdate status mutator", obj, r.fullName.Namespace, r.fullName.Name))
	}
}

type RouteStatusHolder struct {
	objectMeta          *metav1.ObjectMeta
	typeMeta            *metav1.TypeMeta
	routeParentStatuses []*gwv1.RouteParentStatus
	hostnames           []gwv1.Hostname
	resource            client.Object
	transitionTime      metav1.Time
	fullName            types.NamespacedName
	generation          int64
}

func (r *RouteStatusHolder) Mutate(obj client.Object) client.Object {
	return obj
}

func (r *RouteStatusHolder) GetObjectMeta() *metav1.ObjectMeta {
	return r.objectMeta
}

func (r *RouteStatusHolder) GetTypeMeta() *metav1.TypeMeta {
	return r.typeMeta
}

func (r *RouteStatusHolder) GetRouteParentStatuses() []*gwv1.RouteParentStatus {
	return r.routeParentStatuses
}

func (r *RouteStatusHolder) GetHostnames() []gwv1.Hostname {
	return r.hostnames
}

func (r *RouteStatusHolder) GetResource() client.Object {
	return r.resource
}

func (r *RouteStatusHolder) GetTransitionTime() metav1.Time {
	return r.transitionTime
}

func (r *RouteStatusHolder) GetFullName() types.NamespacedName {
	return r.fullName
}

func (r *RouteStatusHolder) GetGeneration() int64 {
	return r.generation
}

func (r *RouteStatusHolder) StatusUpdateFor(parentRef gwv1.ParentReference) status.RouteParentStatusObject {
	return &RouteParentStatusHolder{
		RouteStatusHolder: r,
		ParentRef:         parentRef,
	}
}

func NewRouteStatusHolder(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, hostnames []gwv1.Hostname, routeParentStatuses []*gwv1.RouteParentStatus) status.RouteStatusObject {
	return &RouteStatusHolder{
		objectMeta:          meta,
		typeMeta:            typeMeta,
		routeParentStatuses: routeParentStatuses,
		resource:            resource,
		hostnames:           hostnames,
		transitionTime:      metav1.Time{Time: time.Now()},
		fullName:            types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
		generation:          meta.Generation,
	}
}

type RouteParentStatusHolder struct {
	*RouteStatusHolder
	ParentRef gwv1.ParentReference
}

func (r *RouteParentStatusHolder) AddCondition(_ gwv1.RouteConditionType, _ metav1.ConditionStatus, _ gwv1.RouteConditionReason, _ string) metav1.Condition {
	return metav1.Condition{}
}

func (r *RouteParentStatusHolder) ConditionExists(conditionType gwv1.RouteConditionType) bool {
	for _, c := range r.ConditionsForParentRef(r.ParentRef) {
		if c.Type == string(conditionType) {
			return true
		}
	}
	return false
}

func (r *RouteParentStatusHolder) ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition {
	for _, rps := range r.routeParentStatuses {
		if cmp.Equal(rps.ParentRef, parentRef) {
			return rps.Conditions
		}
	}

	return nil
}

func (r *RouteParentStatusHolder) GetRouteStatusObject() status.RouteStatusObject {
	return r.RouteStatusHolder
}

func (r *RouteParentStatusHolder) GetParentRef() gwv1.ParentReference {
	return r.ParentRef
}
