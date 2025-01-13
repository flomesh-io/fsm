package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type concurrencyLimitReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *concurrencyLimitReconciler) NeedLeaderElection() bool {
	return true
}

// NewConcurrencyLimitReconciler returns a new ConcurrencyLimit Reconciler
func NewConcurrencyLimitReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &concurrencyLimitReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ConcurrencyLimit"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a ConcurrencyLimit object and makes changes based on the state read
func (r *concurrencyLimitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	concurrencyLimit := &extv1alpha1.ConcurrencyLimit{}
	err := r.fctx.Get(ctx, req.NamespacedName, concurrencyLimit)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.ConcurrencyLimit{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if concurrencyLimit.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(concurrencyLimit)
		return ctrl.Result{}, nil
	}

	// As ConcurrencyLimit has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(concurrencyLimit, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *concurrencyLimitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.ConcurrencyLimit{}).
		Complete(r); err != nil {
		return err
	}

	return addConcurrencyLimitIndexers(context.Background(), mgr)
}

func addConcurrencyLimitIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerConcurrencyLimit{}, constants.GatewayListenerConcurrencyLimitIndex, func(obj client.Object) []string {
	//	concurrencyLimit := obj.(*extv1alpha1.ListenerConcurrencyLimit)
	//
	//	var gateways []string
	//	for _, targetRef := range concurrencyLimit.Spec.TargetRefs {
	//		if string(targetRef.Kind) == constants.GatewayAPIGatewayKind &&
	//			string(targetRef.Group) == gwv1.GroupName {
	//			gateways = append(gateways, fmt.Sprintf("%s/%d", string(targetRef.Name), targetRef.Port))
	//		}
	//	}
	//
	//	return gateways
	//}); err != nil {
	//	return err
	//}

	return nil
}
