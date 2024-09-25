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

type filterConfigReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *filterConfigReconciler) NeedLeaderElection() bool {
	return true
}

// NewFilterConfigReconciler returns a new FilterConfig Reconciler
func NewFilterConfigReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &filterConfigReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("FilterConfig"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a FilterConfig object and makes changes based on the state read
func (r *filterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	filterConfig := &extv1alpha1.FilterConfig{}
	err := r.fctx.Get(ctx, req.NamespacedName, filterConfig)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.FilterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if filterConfig.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(filterConfig)
		return ctrl.Result{}, nil
	}

	// As FilterConfig has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(filterConfig, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *filterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.FilterConfig{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.FilterConfig{}).
		Complete(r); err != nil {
		return err
	}

	return addFilterConfigIndexers(context.Background(), mgr)
}

func addFilterConfigIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.FilterConfig{}, constants.GatewayFilterConfigIndex, func(obj client.Object) []string {
	//	filterConfig := obj.(*extv1alpha1.FilterConfig)
	//
	//	scope := ptr.Deref(filterConfig.Spec.Scope, extv1alpha1.FilterConfigScopeRoute)
	//	if scope != extv1alpha1.FilterConfigScopeListener {
	//		return nil
	//	}
	//
	//	var gateways []string
	//	for _, targetRef := range filterConfig.Spec.TargetRefs {
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
