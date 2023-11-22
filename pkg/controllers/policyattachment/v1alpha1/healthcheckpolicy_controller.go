package v1alpha1

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/healthcheck"

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

type healthCheckPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
}

func (r *healthCheckPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewHealthCheckPolicyReconciler returns a new HealthCheckPolicy Reconciler
func NewHealthCheckPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &healthCheckPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("HealthCheckPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a HealthCheckPolicy object and makes changes based on the state read
func (r *healthCheckPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.HealthCheckPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.HealthCheckPolicy{
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
func (r *healthCheckPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.HealthCheckPolicy{}).
		Complete(r)
}

func (r *healthCheckPolicyReconciler) getStatusCondition(ctx context.Context, policy *gwpav1alpha1.HealthCheckPolicy) metav1.Condition {
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

		healthCheckPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().HealthCheckPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list HealthCheckPolicies: %s", err),
			}
		}

		healthChecks := make([]gwpav1alpha1.HealthCheckPolicy, 0)
		for _, p := range healthCheckPolicyList.Items {
			if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, svc) {
				healthChecks = append(healthChecks, p)
			}
		}

		sort.Slice(healthChecks, func(i, j int) bool {
			if healthChecks[i].CreationTimestamp.Time.Equal(healthChecks[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&healthChecks[i]).String() < client.ObjectKeyFromObject(&healthChecks[j]).String()
			}

			return healthChecks[i].CreationTimestamp.Time.Before(healthChecks[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPolicyByService(policy, healthChecks, svc); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with HealthCheckPolicy: %s", conflict),
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

		healthCheckPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().HealthCheckPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list HealthCheckPolicies: %s", err),
			}
		}

		healthCheckPolicies := make([]gwpav1alpha1.HealthCheckPolicy, 0)
		for _, p := range healthCheckPolicyList.Items {
			if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, svcimp) {
				healthCheckPolicies = append(healthCheckPolicies, p)
			}
		}

		sort.Slice(healthCheckPolicies, func(i, j int) bool {
			if healthCheckPolicies[i].CreationTimestamp.Time.Equal(healthCheckPolicies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&healthCheckPolicies[i]).String() < client.ObjectKeyFromObject(&healthCheckPolicies[j]).String()
			}

			return healthCheckPolicies[i].CreationTimestamp.Time.Before(healthCheckPolicies[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPolicyByServiceImport(policy, healthCheckPolicies, svcimp); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with HealthCheckPolicy: %s", conflict),
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

func (r *healthCheckPolicyReconciler) getConflictedPolicyByService(healthCheckPolicy *gwpav1alpha1.HealthCheckPolicy, allHealthCheckPolicies []gwpav1alpha1.HealthCheckPolicy, svc *corev1.Service) *types.NamespacedName {
	for _, port := range svc.Spec.Ports {
		if conflict := r.findConflict(healthCheckPolicy, allHealthCheckPolicies, svc, port.Port); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *healthCheckPolicyReconciler) getConflictedPolicyByServiceImport(healthCheckPolicy *gwpav1alpha1.HealthCheckPolicy, allHealthCheckPolicies []gwpav1alpha1.HealthCheckPolicy, svcimp *mcsv1alpha1.ServiceImport) *types.NamespacedName {
	for _, port := range svcimp.Spec.Ports {
		if conflict := r.findConflict(healthCheckPolicy, allHealthCheckPolicies, svcimp, port.Port); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *healthCheckPolicyReconciler) findConflict(healthCheckPolicy *gwpav1alpha1.HealthCheckPolicy, allHealthCheckPolicies []gwpav1alpha1.HealthCheckPolicy, svc client.Object, port int32) *types.NamespacedName {
	for _, policy := range allHealthCheckPolicies {
		if !gwutils.IsRefToTarget(policy.Spec.TargetRef, svc) {
			continue
		}

		c1 := healthcheck.GetHealthCheckConfigIfPortMatchesPolicy(port, policy)
		if c1 == nil {
			continue
		}

		c2 := healthcheck.GetHealthCheckConfigIfPortMatchesPolicy(port, *healthCheckPolicy)
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
