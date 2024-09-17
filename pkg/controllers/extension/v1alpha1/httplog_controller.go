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

type httpLogReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *httpLogReconciler) NeedLeaderElection() bool {
	return true
}

// NewHTTPLogReconciler returns a new HTTPLog Reconciler
func NewHTTPLogReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &httpLogReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("HTTPLog"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a HTTPLog object and makes changes based on the state read
func (r *httpLogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	httpLog := &extv1alpha1.HTTPLog{}
	err := r.fctx.Get(ctx, req.NamespacedName, httpLog)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.HTTPLog{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if httpLog.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(httpLog)
		return ctrl.Result{}, nil
	}

	// As HTTPLog has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(httpLog, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *httpLogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.HTTPLog{}).
		Complete(r); err != nil {
		return err
	}

	return addHTTPLogIndexers(context.Background(), mgr)
}

func addHTTPLogIndexers(ctx context.Context, mgr manager.Manager) error {
	return nil
}
