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

type circuitBreakerReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *circuitBreakerReconciler) NeedLeaderElection() bool {
	return true
}

// NewCircuitBreakerReconciler returns a new CircuitBreaker Reconciler
func NewCircuitBreakerReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &circuitBreakerReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("CircuitBreaker"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a CircuitBreaker object and makes changes based on the state read
func (r *circuitBreakerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	circuitBreaker := &extv1alpha1.CircuitBreaker{}
	err := r.fctx.Get(ctx, req.NamespacedName, circuitBreaker)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.CircuitBreaker{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if circuitBreaker.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(circuitBreaker)
		return ctrl.Result{}, nil
	}

	// As CircuitBreaker has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(circuitBreaker, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *circuitBreakerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.CircuitBreaker{}).
		Complete(r); err != nil {
		return err
	}

	return addCircuitBreakerIndexers(context.Background(), mgr)
}

func addCircuitBreakerIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerCircuitBreaker{}, constants.GatewayListenerCircuitBreakerIndex, func(obj client.Object) []string {
	//	circuitBreaker := obj.(*extv1alpha1.ListenerCircuitBreaker)
	//
	//	var gateways []string
	//	for _, targetRef := range circuitBreaker.Spec.TargetRefs {
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
