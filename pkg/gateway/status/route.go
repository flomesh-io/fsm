package status

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"time"
)

type RouteStatusUpdate struct {
	FullName            types.NamespacedName
	RouteParentStatuses []*gwv1.RouteParentStatus
	GatewayRef          types.NamespacedName
	Resource            client.Object
	Generation          int64
	TransitionTime      metav1.Time
}

type RouteParentStatusUpdate struct {
	*RouteStatusUpdate
	parentRef gwv1.ParentReference
}

func (r *RouteStatusUpdate) StatusUpdateFor(parentRef gwv1.ParentReference) *RouteParentStatusUpdate {
	return &RouteParentStatusUpdate{
		RouteStatusUpdate: r,
		parentRef:         parentRef,
	}
}

func (r *RouteParentStatusUpdate) AddCondition(conditionType gwv1.RouteConditionType, status metav1.ConditionStatus, reason gwv1.RouteConditionReason, message string) metav1.Condition {
	var rps *gwv1.RouteParentStatus

	for _, v := range r.RouteParentStatuses {
		if v.ParentRef == r.parentRef {
			rps = v
			break
		}
	}

	if rps == nil {
		rps = &gwv1.RouteParentStatus{
			ParentRef:      r.parentRef,
			ControllerName: constants.GatewayController,
		}

		r.RouteParentStatuses = append(r.RouteParentStatuses, rps)
	}

	msg := message
	if cond := metautil.FindStatusCondition(rps.Conditions, string(conditionType)); cond != nil {
		msg = cond.Message + ", " + message
	}

	cond := metav1.Condition{
		Reason:             string(reason),
		Status:             status,
		Type:               string(conditionType),
		Message:            msg,
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: r.Generation,
	}

	metautil.SetStatusCondition(&rps.Conditions, cond)

	return cond
}

func (r *RouteParentStatusUpdate) ConditionExists(conditionType gwv1.RouteConditionType) bool {
	for _, c := range r.ConditionsForParentRef(r.parentRef) {
		if c.Type == string(conditionType) {
			return true
		}
	}
	return false
}

func (r *RouteStatusUpdate) ConditionsForParentRef(parentRef gwv1.ParentReference) []metav1.Condition {
	for _, rps := range r.RouteParentStatuses {
		if rps.ParentRef == parentRef {
			return rps.Conditions
		}
	}

	return nil
}

func (r *RouteStatusUpdate) Mutate(obj client.Object) client.Object {
	var newRouteParentStatuses []gwv1.RouteParentStatus

	for _, rps := range r.RouteParentStatuses {
		for i := range rps.Conditions {
			cond := &rps.Conditions[i]

			cond.ObservedGeneration = r.Generation
			cond.LastTransitionTime = r.TransitionTime
		}

		newRouteParentStatuses = append(newRouteParentStatuses, *rps)
	}

	switch o := obj.(type) {
	case *gwv1.HTTPRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		for _, rps := range o.Status.Parents {
			if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
				newRouteParentStatuses = append(newRouteParentStatuses, rps)
			}
		}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1.GRPCRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		for _, rps := range o.Status.Parents {
			if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
				newRouteParentStatuses = append(newRouteParentStatuses, rps)
			}
		}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.TLSRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		for _, rps := range o.Status.Parents {
			if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
				newRouteParentStatuses = append(newRouteParentStatuses, rps)
			}
		}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.TCPRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		for _, rps := range o.Status.Parents {
			if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
				newRouteParentStatuses = append(newRouteParentStatuses, rps)
			}
		}

		route.Status.Parents = newRouteParentStatuses

		return route
	case *gwv1alpha2.UDPRoute:
		route := o.DeepCopy()

		// Get all the RouteParentStatuses that are for other Gateways.
		for _, rps := range o.Status.Parents {
			if !gwutils.IsRefToGateway(rps.ParentRef, r.GatewayRef) {
				newRouteParentStatuses = append(newRouteParentStatuses, rps)
			}
		}

		route.Status.Parents = newRouteParentStatuses

		return route

	default:
		panic(fmt.Sprintf("Unsupported %T object %s/%s in RouteConditionsUpdate status mutator", obj, r.FullName.Namespace, r.FullName.Name))
	}
}
