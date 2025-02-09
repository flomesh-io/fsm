package v1alpha2

import (
	"context"
	"fmt"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type routeRuleFilterPolicyReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *routeRuleFilterPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewRouteRuleFilterPolicyReconciler returns a new RouteRuleFilterPolicy Reconciler
func NewRouteRuleFilterPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &routeRuleFilterPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("RouteRuleFilterPolicy"),
		fctx:     ctx,
	}

	return r
}

// Reconcile reads that state of the cluster for a RouteRuleFilterPolicy object and makes changes based on the state read
func (r *routeRuleFilterPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha2.RouteRuleFilterPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha2.RouteRuleFilterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if policy.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(policy)
		return ctrl.Result{}, nil
	}

	r.fctx.GatewayEventHandler.OnAdd(policy, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *routeRuleFilterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha2.RouteRuleFilterPolicy{}).
		Complete(r); err != nil {
		return err
	}

	return addRouteRuleFilterPolicyIndexer(context.Background(), mgr)
}

func addRouteRuleFilterPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha2.RouteRuleFilterPolicy{}, constants.RouteRouteRuleFilterPolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwpav1alpha2.RouteRuleFilterPolicy)

		var targets []string
		for _, targetRef := range policy.Spec.TargetRefs {
			targets = append(targets, fmt.Sprintf("%s/%s/%s/%s", targetRef.Kind, policy.Namespace, targetRef.Name, targetRef.Rule))
		}

		return targets
	}); err != nil {
		return err
	}

	return nil
}
