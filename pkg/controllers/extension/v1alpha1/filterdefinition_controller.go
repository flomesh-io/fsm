package v1alpha1

import (
	"context"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

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

type filterDefinitionReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *filterDefinitionReconciler) NeedLeaderElection() bool {
	return true
}

// NewFilterDefinitionReconciler returns a new FilterDefinition Reconciler
func NewFilterDefinitionReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &filterDefinitionReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("FilterDefinition"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a FilterDefinition object and makes changes based on the state read
func (r *filterDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	filterDefinition := &extv1alpha1.FilterDefinition{}
	err := r.fctx.Get(ctx, req.NamespacedName, filterDefinition)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.FilterDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if filterDefinition.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(filterDefinition)
		return ctrl.Result{}, nil
	}

	// As FilterDefinition has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(filterDefinition, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *filterDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.FilterDefinition{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.FilterDefinition{}).
		Complete(r); err != nil {
		return err
	}

	return addFilterDefinitionIndexers(context.Background(), mgr)
}

func addFilterDefinitionIndexers(ctx context.Context, mgr manager.Manager) error {
	return nil
}
