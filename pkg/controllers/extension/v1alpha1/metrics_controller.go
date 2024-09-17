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

type metricsReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *metricsReconciler) NeedLeaderElection() bool {
	return true
}

// NewMetricsReconciler returns a new Metrics Reconciler
func NewMetricsReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &metricsReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Metrics"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a Metrics object and makes changes based on the state read
func (r *metricsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	metrics := &extv1alpha1.Metrics{}
	err := r.fctx.Get(ctx, req.NamespacedName, metrics)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.Metrics{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if metrics.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(metrics)
		return ctrl.Result{}, nil
	}

	// As Metrics has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(metrics, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *metricsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.Metrics{}).
		Complete(r); err != nil {
		return err
	}

	return addMetricsIndexers(context.Background(), mgr)
}

func addMetricsIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerMetrics{}, constants.GatewayListenerMetricsIndex, func(obj client.Object) []string {
	//	metrics := obj.(*extv1alpha1.ListenerMetrics)
	//
	//	var gateways []string
	//	for _, targetRef := range metrics.Spec.TargetRefs {
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
