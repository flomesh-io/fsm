package v2

import (
	"fmt"
	"time"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"k8s.io/utils/ptr"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	"github.com/google/go-cmp/cmp"

	"github.com/flomesh-io/fsm/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type PolicyTargetReference = gwv1alpha2.NamespacedPolicyTargetReference

type DefaultPolicyStatusObject struct {
	objectMeta             *metav1.ObjectMeta
	typeMeta               *metav1.TypeMeta
	targetRefs             []PolicyTargetReference
	policyAncestorStatuses []*gwv1alpha2.PolicyAncestorStatus
	resource               client.Object
	transitionTime         metav1.Time
	fullName               types.NamespacedName
	generation             int64
}

func (p *DefaultPolicyStatusObject) GetObjectMeta() *metav1.ObjectMeta {
	return p.objectMeta
}

func (p *DefaultPolicyStatusObject) GetTypeMeta() *metav1.TypeMeta {
	return p.typeMeta
}

func (p *DefaultPolicyStatusObject) GetResource() client.Object {
	return p.resource
}

func (p *DefaultPolicyStatusObject) GetTransitionTime() metav1.Time {
	return p.transitionTime
}

func (p *DefaultPolicyStatusObject) GetFullName() types.NamespacedName {
	return p.fullName
}

func (p *DefaultPolicyStatusObject) GetGeneration() int64 {
	return p.generation
}

func (p *DefaultPolicyStatusObject) StatusUpdateFor(ancestorRef gwv1.ParentReference) status.PolicyAncestorStatusObject {
	return &DefaultPolicyAncestorStatusUpdate{
		DefaultPolicyStatusObject: p,
		ancestorRef:               ancestorRef,
	}
}

func (p *DefaultPolicyStatusObject) ConditionsForAncestorRef(ancestorRef gwv1.ParentReference) []metav1.Condition {
	for _, pas := range p.policyAncestorStatuses {
		if cmp.Equal(pas.AncestorRef, ancestorRef) {
			return pas.Conditions
		}
	}

	return nil
}

func (p *DefaultPolicyStatusObject) Mutate(obj client.Object) client.Object {
	return obj
}

// ---------------------------------------------------------------------------

type DefaultPolicyAncestorStatusUpdate struct {
	*DefaultPolicyStatusObject
	ancestorRef gwv1.ParentReference
}

func (p *DefaultPolicyAncestorStatusUpdate) ConditionExists(conditionType gwv1alpha2.PolicyConditionType) bool {
	for _, c := range p.ConditionsForAncestorRef(p.ancestorRef) {
		if c.Type == string(conditionType) {
			return true
		}
	}
	return false
}

func (p *DefaultPolicyAncestorStatusUpdate) AddCondition(conditionType gwv1alpha2.PolicyConditionType, status metav1.ConditionStatus, reason gwv1alpha2.PolicyConditionReason, message string) metav1.Condition {
	var pas *gwv1alpha2.PolicyAncestorStatus

	for _, v := range p.policyAncestorStatuses {
		if cmp.Equal(v.AncestorRef, p.ancestorRef) {
			pas = v
			break
		}
	}

	if pas == nil {
		pas = &gwv1alpha2.PolicyAncestorStatus{
			AncestorRef:    p.ancestorRef,
			ControllerName: constants.GatewayController,
		}

		p.policyAncestorStatuses = append(p.policyAncestorStatuses, pas)
	}

	cond := metav1.Condition{
		Reason:             string(reason),
		Status:             status,
		Type:               string(conditionType),
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: p.generation,
	}

	metautil.SetStatusCondition(&pas.Conditions, cond)

	return cond
}

func (p *DefaultPolicyAncestorStatusUpdate) GetPolicyStatusObject() status.PolicyStatusObject {
	return p.DefaultPolicyStatusObject
}

func (p *DefaultPolicyAncestorStatusUpdate) GetAncestorRef() gwv1.ParentReference {
	return p.ancestorRef
}

// ---------------------------------------------------------------------------

type PolicyStatusUpdate struct {
	DefaultPolicyStatusObject
}

func (p *PolicyStatusUpdate) Mutate(obj client.Object) client.Object {
	var newPolicyAncestorStatuses []gwv1alpha2.PolicyAncestorStatus
	for _, pas := range p.policyAncestorStatuses {
		for i := range pas.Conditions {
			cond := &pas.Conditions[i]

			cond.ObservedGeneration = p.generation
			cond.LastTransitionTime = p.transitionTime
		}

		newPolicyAncestorStatuses = append(newPolicyAncestorStatuses, *pas)
	}

	switch o := obj.(type) {
	case *gwv1alpha3.BackendTLSPolicy:
		policy := o.DeepCopy()
		policy.Status.Ancestors = newPolicyAncestorStatuses
		return policy
	case *gwv1alpha2.BackendLBPolicy:
		policy := o.DeepCopy()
		policy.Status.Ancestors = newPolicyAncestorStatuses
		return policy
	case *gwpav1alpha2.RetryPolicy:
		policy := o.DeepCopy()
		policy.Status.Ancestors = newPolicyAncestorStatuses
		return policy
	case *gwpav1alpha2.HealthCheckPolicy:
		policy := o.DeepCopy()
		policy.Status.Ancestors = newPolicyAncestorStatuses
		return policy
	default:
		panic(fmt.Sprintf("Unsupported %T object %s/%s in status mutator", obj, p.fullName.Namespace, p.fullName.Name))
	}
}

func NewPolicyStatusUpdateWithLocalPolicyTargetReference(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, targetRefs []gwv1alpha2.LocalPolicyTargetReference, policyAncestorStatuses []*gwv1alpha2.PolicyAncestorStatus) *PolicyStatusUpdate {
	refs := make([]PolicyTargetReference, len(targetRefs))
	for i, ref := range targetRefs {
		refs[i] = PolicyTargetReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Name:      ref.Name,
			Namespace: ptr.To(gwv1.Namespace(meta.Namespace)),
		}
	}

	return newPolicyStatusUpdate(resource, meta, typeMeta, refs, policyAncestorStatuses)
}

func NewPolicyStatusUpdateWithLocalPolicyTargetReferenceWithSectionName(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, targetRefs []gwv1alpha2.LocalPolicyTargetReferenceWithSectionName, policyAncestorStatuses []*gwv1alpha2.PolicyAncestorStatus) *PolicyStatusUpdate {
	refs := make([]PolicyTargetReference, len(targetRefs))
	for i, ref := range targetRefs {
		refs[i] = PolicyTargetReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Name:      ref.Name,
			Namespace: ptr.To(gwv1.Namespace(meta.Namespace)),
		}
	}

	return newPolicyStatusUpdate(resource, meta, typeMeta, refs, policyAncestorStatuses)
}

func NewPolicyStatusUpdateWithNamespacedPolicyTargetReference(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, targetRefs []gwv1alpha2.NamespacedPolicyTargetReference, policyAncestorStatuses []*gwv1alpha2.PolicyAncestorStatus) *PolicyStatusUpdate {
	return newPolicyStatusUpdate(resource, meta, typeMeta, targetRefs, policyAncestorStatuses)
}

func newPolicyStatusUpdate(resource client.Object, meta *metav1.ObjectMeta, typeMeta *metav1.TypeMeta, targetRefs []PolicyTargetReference, policyAncestorStatuses []*gwv1alpha2.PolicyAncestorStatus) *PolicyStatusUpdate {
	return &PolicyStatusUpdate{
		DefaultPolicyStatusObject: DefaultPolicyStatusObject{
			objectMeta:             meta,
			typeMeta:               typeMeta,
			targetRefs:             targetRefs,
			policyAncestorStatuses: policyAncestorStatuses,
			resource:               resource,
			transitionTime:         metav1.Time{Time: time.Now()},
			fullName:               types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name},
			generation:             meta.Generation,
		},
	}
}

// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
