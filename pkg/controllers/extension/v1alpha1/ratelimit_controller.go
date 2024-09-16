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

type rateLimitReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *rateLimitReconciler) NeedLeaderElection() bool {
	return true
}

// NewRateLimitReconciler returns a new RateLimit Reconciler
func NewRateLimitReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &rateLimitReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("RateLimit"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a RateLimit object and makes changes based on the state read
func (r *rateLimitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rateLimit := &extv1alpha1.RateLimit{}
	err := r.fctx.Get(ctx, req.NamespacedName, rateLimit)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.RateLimit{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if rateLimit.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(rateLimit)
		return ctrl.Result{}, nil
	}

	// As RateLimit has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(rateLimit, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *rateLimitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.RateLimit{}).
		Complete(r); err != nil {
		return err
	}

	return addRateLimitIndexers(context.Background(), mgr)
}

func addRateLimitIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerRateLimit{}, constants.GatewayListenerRateLimitIndex, func(obj client.Object) []string {
	//	rateLimit := obj.(*extv1alpha1.ListenerRateLimit)
	//
	//	var gateways []string
	//	for _, targetRef := range rateLimit.Spec.TargetRefs {
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
