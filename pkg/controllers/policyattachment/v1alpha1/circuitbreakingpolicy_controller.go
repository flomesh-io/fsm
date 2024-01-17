package v1alpha1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/circuitbreaking"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

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

type circuitBreakingPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
	statusProcessor           *status.ServicePolicyStatusProcessor
}

func (r *circuitBreakingPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewCircuitBreakingPolicyReconciler returns a new CircuitBreakingPolicy Reconciler
func NewCircuitBreakingPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &circuitBreakingPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("CircuitBreakingPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}

	r.statusProcessor = &status.ServicePolicyStatusProcessor{
		Client:              r.fctx.Client,
		GetAttachedPolicies: r.getAttachedCircuitBreakings,
		FindConflict:        r.findConflict,
	}

	return r
}

// Reconcile reads that state of the cluster for a CircuitBreakingPolicy object and makes changes based on the state read
func (r *circuitBreakingPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.CircuitBreakingPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.CircuitBreakingPolicy{
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

	metautil.SetStatusCondition(
		&policy.Status.Conditions,
		r.statusProcessor.Process(ctx, policy, policy.Spec.TargetRef),
	)
	if err := r.fctx.Status().Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.EventHandler.OnAdd(policy, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *circuitBreakingPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.CircuitBreakingPolicy{}).
		Complete(r)
}

func (r *circuitBreakingPolicyReconciler) getAttachedCircuitBreakings(policy client.Object, svc client.Object) ([]client.Object, *metav1.Condition) {
	circuitBreakingPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().CircuitBreakingPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list CircuitBreakingPolicies: %s", err)))
	}

	circuitBreakings := make([]client.Object, 0)
	for _, p := range circuitBreakingPolicyList.Items {
		p := p
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
			gwutils.IsRefToTarget(p.Spec.TargetRef, svc) {
			circuitBreakings = append(circuitBreakings, &p)
		}
	}

	return circuitBreakings, nil
}

func (r *circuitBreakingPolicyReconciler) findConflict(circuitBreakingPolicy client.Object, allCircuitBreakingPolicies []client.Object, port int32) *types.NamespacedName {
	currentPolicy := circuitBreakingPolicy.(*gwpav1alpha1.CircuitBreakingPolicy)

	for _, policy := range allCircuitBreakingPolicies {
		policy := policy.(*gwpav1alpha1.CircuitBreakingPolicy)

		c1 := circuitbreaking.GetCircuitBreakingConfigIfPortMatchesPolicy(port, *policy)
		if c1 == nil {
			continue
		}

		c2 := circuitbreaking.GetCircuitBreakingConfigIfPortMatchesPolicy(port, *currentPolicy)
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
