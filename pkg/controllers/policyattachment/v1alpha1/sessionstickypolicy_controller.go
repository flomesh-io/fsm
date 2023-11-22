package v1alpha1

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/sessionsticky"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"

	policyAttachmentApiClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
)

type sessionStickyPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
}

// NewSessionStickyPolicyReconciler returns a new SessionStickyPolicy Reconciler
func NewSessionStickyPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &sessionStickyPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("SessionStickyPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a SessionStickyPolicy object and makes changes based on the state read
func (r *sessionStickyPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.SessionStickyPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.SessionStickyPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if policy.DeletionTimestamp != nil {
		r.fctx.EventHandler.OnDelete(policy)
		return ctrl.Result{}, nil
	}

	metautil.SetStatusCondition(&policy.Status.Conditions, r.getStatusCondition(ctx, policy))
	if err := r.fctx.Status().Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.EventHandler.OnAdd(policy)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *sessionStickyPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.SessionStickyPolicy{}).
		Complete(r)
}

func (r *sessionStickyPolicyReconciler) getStatusCondition(ctx context.Context, policy *gwpav1alpha1.SessionStickyPolicy) metav1.Condition {
	if policy.Spec.TargetRef.Group != constants.KubernetesCoreGroup && policy.Spec.TargetRef.Group != constants.FlomeshAPIGroup {
		return metav1.Condition{
			Type:               string(gwv1alpha2.PolicyConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: policy.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1alpha2.PolicyReasonInvalid),
			Message:            "Invalid target reference group, only kubernetes core or flomesh.io is supported",
		}
	}

	if policy.Spec.TargetRef.Group == constants.KubernetesCoreGroup && policy.Spec.TargetRef.Kind != constants.KubernetesServiceKind {
		return metav1.Condition{
			Type:               string(gwv1alpha2.PolicyConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: policy.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1alpha2.PolicyReasonInvalid),
			Message:            "Invalid target reference kind, only Service is supported for kubernetes core group",
		}
	}

	if policy.Spec.TargetRef.Group == constants.FlomeshAPIGroup && policy.Spec.TargetRef.Kind != constants.FlomeshAPIServiceImportKind {
		return metav1.Condition{
			Type:               string(gwv1alpha2.PolicyConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: policy.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1alpha2.PolicyReasonInvalid),
			Message:            "Invalid target reference kind, only ServiceImport is supported for flomesh.io group",
		}
	}

	if policy.Spec.TargetRef.Group == constants.KubernetesCoreGroup && policy.Spec.TargetRef.Kind == constants.KubernetesServiceKind {
		svc := &corev1.Service{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy, policy.Spec.TargetRef), Name: string(policy.Spec.TargetRef.Name)}, svc); err != nil {
			if errors.IsNotFound(err) {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonTargetNotFound),
					Message:            "Invalid target reference, cannot find target Service",
				}
			} else {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonInvalid),
					Message:            fmt.Sprintf("Failed to get target Service: %s", err),
				}
			}
		}

		sessionStickyPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().SessionStickyPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list SessionStickyPolicies: %s", err),
			}
		}

		sessionStickies := make([]gwpav1alpha1.SessionStickyPolicy, 0)
		for _, p := range sessionStickyPolicyList.Items {
			if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, svc) {
				sessionStickies = append(sessionStickies, p)
			}
		}

		sort.Slice(sessionStickies, func(i, j int) bool {
			if sessionStickies[i].CreationTimestamp.Time.Equal(sessionStickies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&sessionStickies[i]).String() < client.ObjectKeyFromObject(&sessionStickies[j]).String()
			}

			return sessionStickies[i].CreationTimestamp.Time.Before(sessionStickies[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPolicyByService(policy, sessionStickies, svc); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with SessionStickyPolicy: %s", conflict),
			}
		}
	}

	if policy.Spec.TargetRef.Group == constants.FlomeshAPIGroup && policy.Spec.TargetRef.Kind == constants.FlomeshAPIServiceImportKind {
		svcimp := &mcsv1alpha1.ServiceImport{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy, policy.Spec.TargetRef), Name: string(policy.Spec.TargetRef.Name)}, svcimp); err != nil {
			if errors.IsNotFound(err) {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonTargetNotFound),
					Message:            "Invalid target reference, cannot find target ServiceImport",
				}
			} else {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonInvalid),
					Message:            fmt.Sprintf("Failed to get target ServiceImport: %s", err),
				}
			}
		}

		sessionStickyPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().SessionStickyPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list SessionStickyPolicies: %s", err),
			}
		}

		sessionStickies := make([]gwpav1alpha1.SessionStickyPolicy, 0)
		for _, p := range sessionStickyPolicyList.Items {
			if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, svcimp) {
				sessionStickies = append(sessionStickies, p)
			}
		}

		sort.Slice(sessionStickies, func(i, j int) bool {
			if sessionStickies[i].CreationTimestamp.Time.Equal(sessionStickies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&sessionStickies[i]).String() < client.ObjectKeyFromObject(&sessionStickies[j]).String()
			}

			return sessionStickies[i].CreationTimestamp.Time.Before(sessionStickies[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPolicyByServiceImport(policy, sessionStickies, svcimp); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with SessionStickyPolicy: %s", conflict),
			}
		}
	}

	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: policy.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1alpha2.PolicyReasonAccepted),
		Message:            string(gwv1alpha2.PolicyReasonAccepted),
	}
}

func (r *sessionStickyPolicyReconciler) getConflictedPolicyByService(sessionStickyPolicy *gwpav1alpha1.SessionStickyPolicy, allSessionStickyPolicies []gwpav1alpha1.SessionStickyPolicy, svc *corev1.Service) *types.NamespacedName {
	for _, port := range svc.Spec.Ports {
		if conflict := r.findConflict(sessionStickyPolicy, allSessionStickyPolicies, svc, port.Port); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *sessionStickyPolicyReconciler) getConflictedPolicyByServiceImport(sessionStickyPolicy *gwpav1alpha1.SessionStickyPolicy, allSessionStickyPolicies []gwpav1alpha1.SessionStickyPolicy, svcimp *mcsv1alpha1.ServiceImport) *types.NamespacedName {
	for _, port := range svcimp.Spec.Ports {
		if conflict := r.findConflict(sessionStickyPolicy, allSessionStickyPolicies, svcimp, port.Port); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *sessionStickyPolicyReconciler) findConflict(sessionStickyPolicy *gwpav1alpha1.SessionStickyPolicy, allSessionStickyPolicies []gwpav1alpha1.SessionStickyPolicy, svc client.Object, port int32) *types.NamespacedName {
	for _, policy := range allSessionStickyPolicies {
		if !gwutils.IsRefToTarget(policy.Spec.TargetRef, svc) {
			continue
		}

		c1 := sessionsticky.GetSessionStickyConfigIfPortMatchesPolicy(port, policy)
		if c1 == nil {
			continue
		}

		c2 := sessionsticky.GetSessionStickyConfigIfPortMatchesPolicy(port, *sessionStickyPolicy)
		if c2 == nil {
			continue
		}

		if reflect.DeepEqual(c1, c2) {
			continue
		}

		return &types.NamespacedName{
			Name:      policy.Name,
			Namespace: policy.Namespace,
		}
	}

	return nil
}
