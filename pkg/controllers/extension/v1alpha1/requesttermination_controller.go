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

type requestTerminationReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *requestTerminationReconciler) NeedLeaderElection() bool {
	return true
}

// NewRequestTerminationReconciler returns a new RequestTermination Reconciler
func NewRequestTerminationReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &requestTerminationReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("RequestTermination"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a RequestTermination object and makes changes based on the state read
func (r *requestTerminationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	requestTermination := &extv1alpha1.RequestTermination{}
	err := r.fctx.Get(ctx, req.NamespacedName, requestTermination)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.RequestTermination{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if requestTermination.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(requestTermination)
		return ctrl.Result{}, nil
	}

	// As RequestTermination has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(requestTermination, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *requestTerminationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.RequestTermination{}).
		Complete(r); err != nil {
		return err
	}

	return addRequestTerminationIndexers(context.Background(), mgr)
}

func addRequestTerminationIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerRequestTermination{}, constants.GatewayListenerRequestTerminationIndex, func(obj client.Object) []string {
	//	requestTermination := obj.(*extv1alpha1.ListenerRequestTermination)
	//
	//	var gateways []string
	//	for _, targetRef := range requestTermination.Spec.TargetRefs {
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
