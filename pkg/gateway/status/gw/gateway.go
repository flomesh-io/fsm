package gw

import (
	"fmt"
	"sort"
	"time"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayStatusUpdate struct {
	conditions         map[gwv1.GatewayConditionType]metav1.Condition
	existingConditions map[gwv1.GatewayConditionType]metav1.Condition
	listenerStatus     map[string]*gwv1.ListenerStatus
	objectMeta         *metav1.ObjectMeta
	typeMeta           *metav1.TypeMeta
	resource           client.Object
	transitionTime     metav1.Time
	fullName           types.NamespacedName
	generation         int64
	addresses          []gwv1.GatewayStatusAddress
	existingAddresses  []gwv1.GatewayStatusAddress
}

func NewGatewayStatusUpdate(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, gs *gwv1.GatewayStatus) *GatewayStatusUpdate {
	return &GatewayStatusUpdate{
		objectMeta:         meta,
		typeMeta:           typeMeta,
		resource:           resource,
		transitionTime:     metav1.Time{Time: time.Now()},
		fullName:           types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
		generation:         meta.Generation,
		existingConditions: getGatewayConditions(gs),
		existingAddresses:  gs.Addresses,
	}
}

// AddCondition returns a metav1.Condition for a given GatewayConditionType.
func (g *GatewayStatusUpdate) AddCondition(
	conditionType gwv1.GatewayConditionType,
	status metav1.ConditionStatus,
	reason gwv1.GatewayConditionReason,
	message string,
) metav1.Condition {
	if g.conditions == nil {
		g.conditions = make(map[gwv1.GatewayConditionType]metav1.Condition)
	}

	newCond := metav1.Condition{
		Reason:             string(reason),
		Status:             status,
		Type:               string(conditionType),
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: g.generation,
	}

	g.conditions[conditionType] = newCond

	return newCond
}

func (g *GatewayStatusUpdate) ConditionExists(conditionType gwv1.GatewayConditionType) bool {
	_, ok := g.conditions[conditionType]
	return ok
}

func (g *GatewayStatusUpdate) IsStatusConditionTrue(conditionType gwv1.GatewayConditionType) bool {
	cond, ok := g.conditions[conditionType]
	if !ok {
		return false
	}

	return cond.Status == metav1.ConditionTrue
}

func (g *GatewayStatusUpdate) SetAddresses(addresses []gwv1.GatewayStatusAddress) {
	g.addresses = addresses
}

func (g *GatewayStatusUpdate) SetListenerSupportedKinds(listenerName string, groupKinds []gwv1.RouteGroupKind) {
	if g.listenerStatus == nil {
		g.listenerStatus = map[string]*gwv1.ListenerStatus{}
	}

	if g.listenerStatus[listenerName] == nil {
		g.listenerStatus[listenerName] = &gwv1.ListenerStatus{
			Name: gwv1.SectionName(listenerName),
		}
	}

	g.listenerStatus[listenerName].SupportedKinds = append(g.listenerStatus[listenerName].SupportedKinds, groupKinds...)
}

func (g *GatewayStatusUpdate) SetListenerAttachedRoutes(listenerName string, numRoutes int) {
	if g.listenerStatus == nil {
		g.listenerStatus = map[string]*gwv1.ListenerStatus{}
	}
	if g.listenerStatus[listenerName] == nil {
		g.listenerStatus[listenerName] = &gwv1.ListenerStatus{
			Name: gwv1.SectionName(listenerName),
		}
	}

	g.listenerStatus[listenerName].AttachedRoutes = int32(numRoutes)
}

// AddListenerCondition adds a Condition for the specified listener.
func (g *GatewayStatusUpdate) AddListenerCondition(
	listenerName string,
	cond gwv1.ListenerConditionType,
	status metav1.ConditionStatus,
	reason gwv1.ListenerConditionReason,
	message string,
) metav1.Condition {
	if g.listenerStatus == nil {
		g.listenerStatus = map[string]*gwv1.ListenerStatus{}
	}

	if g.listenerStatus[listenerName] == nil {
		g.listenerStatus[listenerName] = &gwv1.ListenerStatus{
			Name: gwv1.SectionName(listenerName),
		}
	}

	listenerStatus := g.listenerStatus[listenerName]

	condition := metav1.Condition{
		Reason:             string(reason),
		Status:             status,
		Type:               string(cond),
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: g.generation,
	}

	metautil.SetStatusCondition(&listenerStatus.Conditions, condition)

	return condition
}

func getGatewayConditions(gs *gwv1.GatewayStatus) map[gwv1.GatewayConditionType]metav1.Condition {
	conditions := make(map[gwv1.GatewayConditionType]metav1.Condition)
	for _, cond := range gs.Conditions {
		if _, ok := conditions[gwv1.GatewayConditionType(cond.Type)]; !ok {
			conditions[gwv1.GatewayConditionType(cond.Type)] = cond
		}
	}
	return conditions
}

func (g *GatewayStatusUpdate) GetListenerStatus(listenerName string) *gwv1.ListenerStatus {
	if g.listenerStatus == nil {
		return nil
	}

	return g.listenerStatus[listenerName]
}

func (g *GatewayStatusUpdate) Mutate(obj client.Object) client.Object {
	o, ok := obj.(*gwv1.Gateway)
	if !ok {
		panic(fmt.Sprintf("Unsupported %T object %s/%s in GatewayStatusUpdate status mutator",
			obj, g.fullName.Namespace, g.fullName.Name,
		))
	}

	updated := o.DeepCopy()

	var conditionsToWrite []metav1.Condition

	for _, cond := range g.conditions {
		// Set the Condition's observed generation based on
		// the generation of the gateway we looked at.
		cond.ObservedGeneration = g.generation
		cond.LastTransitionTime = g.transitionTime

		// is there a newer Condition on the gateway matching
		// this condition's type? If so, our observation is stale,
		// so don't write it, keep the newer one instead.
		var newerConditionExists bool
		for _, existingCond := range g.existingConditions {
			if existingCond.Type != cond.Type {
				continue
			}

			if existingCond.ObservedGeneration > cond.ObservedGeneration {
				conditionsToWrite = append(conditionsToWrite, existingCond)
				newerConditionExists = true
				break
			}
		}

		// if we didn't find a newer version of the Condition on the
		// gateway, then write the one we computed.
		if !newerConditionExists {
			conditionsToWrite = append(conditionsToWrite, cond)
		}
	}

	sort.Slice(conditionsToWrite, func(i, j int) bool {
		if conditionsToWrite[i].Type == conditionsToWrite[j].Type {
			return conditionsToWrite[i].Reason < conditionsToWrite[j].Reason
		}
		return conditionsToWrite[i].Type < conditionsToWrite[j].Type
	})
	updated.Status.Conditions = conditionsToWrite

	// Overwrite all listener statuses since we re-compute all of them
	// for each Gateway status update.
	var listenerStatusToWrite []gwv1.ListenerStatus
	for _, status := range g.listenerStatus {
		if status.Conditions == nil {
			// Conditions is a required field so we have to specify an empty slice here
			status.Conditions = []metav1.Condition{}
		}
		if status.SupportedKinds == nil {
			// SupportedKinds is a required field so we have to specify an empty slice here
			status.SupportedKinds = []gwv1.RouteGroupKind{}
		}
		listenerStatusToWrite = append(listenerStatusToWrite, *status)
	}

	sort.Slice(listenerStatusToWrite, func(i, j int) bool {
		return listenerStatusToWrite[i].Name < listenerStatusToWrite[j].Name
	})
	updated.Status.Listeners = listenerStatusToWrite

	// Gateway addresses
	var addressesToWrite []gwv1.GatewayStatusAddress
	if len(g.addresses) > 0 {
		addressesToWrite = g.addresses
	} else {
		addressesToWrite = g.existingAddresses
	}

	sort.Slice(addressesToWrite, func(i, j int) bool {
		if addressesToWrite[i].Type == nil || addressesToWrite[j].Type == nil {
			return addressesToWrite[i].Value < addressesToWrite[j].Value
		}

		if *addressesToWrite[i].Type == *addressesToWrite[j].Type {
			return addressesToWrite[i].Value < addressesToWrite[j].Value
		}

		return *addressesToWrite[i].Type < *addressesToWrite[j].Type
	})
	updated.Status.Addresses = addressesToWrite

	return updated
}
