package v1alpha1

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

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

type loadBalancerPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
}

// NewLoadBalancerPolicyReconciler returns a new LoadBalancerPolicy Reconciler
func NewLoadBalancerPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &loadBalancerPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("LoadBalancerPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a LoadBalancerPolicy object and makes changes based on the state read
func (r *loadBalancerPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.LoadBalancerPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.LoadBalancerPolicy{
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
func (r *loadBalancerPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.LoadBalancerPolicy{}).
		Complete(r)
}

func (r *loadBalancerPolicyReconciler) getStatusCondition(ctx context.Context, policy *gwpav1alpha1.LoadBalancerPolicy) metav1.Condition {
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

		loadBalancerPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().LoadBalancerPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list LoadBalancerPolicies: %s", err),
			}
		}

		loadBalancers := make([]gwpav1alpha1.LoadBalancerPolicy, 0)
		for _, p := range loadBalancerPolicyList.Items {
			p := p
			if gwutils.IsAcceptedLoadBalancerPolicy(&p) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, svc) {
				loadBalancers = append(loadBalancers, p)
			}
		}

		sort.Slice(loadBalancers, func(i, j int) bool {
			if loadBalancers[i].CreationTimestamp.Time.Equal(loadBalancers[j].CreationTimestamp.Time) {
				return loadBalancers[i].Name < loadBalancers[j].Name
			}

			return loadBalancers[i].CreationTimestamp.Time.Before(loadBalancers[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPolicyByService(policy, loadBalancers, svc); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with LoadBalancerPolicy: %s", conflict),
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

		loadBalancerPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().LoadBalancerPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list LoadBalancerPolicies: %s", err),
			}
		}

		loadBalancers := make([]gwpav1alpha1.LoadBalancerPolicy, 0)
		for _, p := range loadBalancerPolicyList.Items {
			p := p
			if gwutils.IsAcceptedLoadBalancerPolicy(&p) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, svcimp) {
				loadBalancers = append(loadBalancers, p)
			}
		}

		sort.Slice(loadBalancers, func(i, j int) bool {
			if loadBalancers[i].CreationTimestamp.Time.Equal(loadBalancers[j].CreationTimestamp.Time) {
				return loadBalancers[i].Name < loadBalancers[j].Name
			}

			return loadBalancers[i].CreationTimestamp.Time.Before(loadBalancers[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPolicyByServiceImport(policy, loadBalancers, svcimp); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with LoadBalancerPolicy: %s", conflict),
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

func (r *loadBalancerPolicyReconciler) getConflictedPolicyByService(loadBalancerPolicy *gwpav1alpha1.LoadBalancerPolicy, allLoadBalancerPolicies []gwpav1alpha1.LoadBalancerPolicy, svc *corev1.Service) *types.NamespacedName {
	for _, p := range allLoadBalancerPolicies {
		p := p
		if gwutils.LoadBalancerPolicyMatchesService(&p, svc) &&
			reflect.DeepEqual(p.Spec, loadBalancerPolicy.Spec) {
			continue
		}

		return &types.NamespacedName{
			Name:      p.Name,
			Namespace: p.Namespace,
		}
	}

	return nil
}

func (r *loadBalancerPolicyReconciler) getConflictedPolicyByServiceImport(loadBalancerPolicy *gwpav1alpha1.LoadBalancerPolicy, allLoadBalancerPolicies []gwpav1alpha1.LoadBalancerPolicy, svcimp *mcsv1alpha1.ServiceImport) *types.NamespacedName {
	for _, p := range allLoadBalancerPolicies {
		p := p
		if gwutils.LoadBalancerPolicyMatchesServiceImport(&p, svcimp) &&
			reflect.DeepEqual(p.Spec, loadBalancerPolicy.Spec) {
			continue
		}

		return &types.NamespacedName{
			Name:      p.Name,
			Namespace: p.Namespace,
		}
	}

	return nil
}
