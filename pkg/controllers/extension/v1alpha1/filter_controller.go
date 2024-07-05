package v1alpha1

import (
	"context"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type filterReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *filterReconciler) NeedLeaderElection() bool {
	return true
}

// NewFilterReconciler returns a new Filter Reconciler
func NewFilterReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &filterReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Filter"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a Filter object and makes changes based on the state read
func (r *filterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	filter := &extv1alpha1.Filter{}
	err := r.fctx.Get(ctx, req.NamespacedName, filter)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.Filter{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if filter.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(filter)
		return ctrl.Result{}, nil
	}

	// As Filter has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(filter, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *filterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.Filter{}).
		Complete(r); err != nil {
		return err
	}

	return addFilterIndexers(context.Background(), mgr)
}

func addFilterIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.Filter{}, constants.GatewayFilterIndex, func(obj client.Object) []string {
		filter := obj.(*extv1alpha1.Filter)

		var gateways []string
		for _, targetRef := range filter.Spec.TargetRefs {
			if string(targetRef.Kind) == constants.GatewayAPIGatewayKind &&
				string(targetRef.Group) == gwv1.GroupName {
				gateways = append(gateways,
					types.NamespacedName{
						Namespace: filter.Namespace,
						Name:      string(targetRef.Name),
					}.String(),
				)
			}
		}

		return gateways
	}); err != nil {
		return err
	}

	return nil
}
