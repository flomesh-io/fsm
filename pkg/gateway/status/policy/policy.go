package policy

import (
	"fmt"
	"time"

	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
)

type PolicyUpdate struct {
	objectMeta     *metav1.ObjectMeta
	typeMeta       *metav1.TypeMeta
	targetRef      gwv1alpha2.NamespacedPolicyTargetReference
	resource       client.Object
	transitionTime metav1.Time
	fullName       types.NamespacedName
	generation     int64
	conditions     []metav1.Condition
}

func NewPolicyUpdate(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, targetRef gwv1alpha2.NamespacedPolicyTargetReference, conditions []metav1.Condition) *PolicyUpdate {
	return &PolicyUpdate{
		objectMeta:     meta,
		typeMeta:       typeMeta,
		resource:       resource,
		transitionTime: metav1.Now(),
		fullName:       types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
		generation:     meta.Generation,
		targetRef:      targetRef,
		conditions:     conditions,
	}
}

func (r *PolicyUpdate) AddCondition(condition metav1.Condition) metav1.Condition {
	//msg := condition.Message
	//if cond := metautil.FindStatusCondition(r.conditions, condition.Type); cond != nil {
	//	msg = cond.Message + ", " + msg
	//}

	cond := metav1.Condition{
		Reason:             condition.Reason,
		Status:             condition.Status,
		Type:               condition.Type,
		Message:            condition.Message,
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: r.generation,
	}

	metautil.SetStatusCondition(&r.conditions, cond)

	return cond
}

func (r *PolicyUpdate) ConditionExists(conditionType gwv1alpha2.PolicyConditionType) bool {
	for _, c := range r.conditions {
		if c.Type == string(conditionType) {
			return true
		}
	}
	return false
}

func (r *PolicyUpdate) Conditions() []metav1.Condition {
	return r.conditions
}

func (r *PolicyUpdate) GetTargetRef() gwv1alpha2.NamespacedPolicyTargetReference {
	return r.targetRef
}

func (r *PolicyUpdate) GetResource() client.Object {
	return r.resource
}

func (r *PolicyUpdate) Mutate(obj client.Object) client.Object {
	switch o := obj.(type) {
	case *gwpav1alpha1.AccessControlPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.RateLimitPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.FaultInjectionPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.SessionStickyPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.CircuitBreakingPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.LoadBalancerPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.HealthCheckPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.RetryPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	case *gwpav1alpha1.UpstreamTLSPolicy:
		policy := o.DeepCopy()
		policy.Status.Conditions = r.conditions
		return policy
	default:
		panic(fmt.Sprintf("Unsupported %T object %s/%s in PolicyUpdate status mutator", obj, r.fullName.Namespace, r.fullName.Name))
	}
}
