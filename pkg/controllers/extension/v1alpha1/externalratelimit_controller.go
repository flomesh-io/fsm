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

type externalRateLimitReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *externalRateLimitReconciler) NeedLeaderElection() bool {
	return true
}

// NewExternalRateLimitReconciler returns a new ExternalRateLimit Reconciler
func NewExternalRateLimitReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &externalRateLimitReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ExternalRateLimit"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a ExternalRateLimit object and makes changes based on the state read
func (r *externalRateLimitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	externalRateLimit := &extv1alpha1.ExternalRateLimit{}
	err := r.fctx.Get(ctx, req.NamespacedName, externalRateLimit)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.ExternalRateLimit{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if externalRateLimit.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(externalRateLimit)
		return ctrl.Result{}, nil
	}

	// As ExternalRateLimit has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(externalRateLimit, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *externalRateLimitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.ExternalRateLimit{}).
		Complete(r); err != nil {
		return err
	}

	return addExternalRateLimitIndexers(context.Background(), mgr)
}

func addExternalRateLimitIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerExternalRateLimit{}, constants.GatewayListenerExternalRateLimitIndex, func(obj client.Object) []string {
	//	externalRateLimit := obj.(*extv1alpha1.ListenerExternalRateLimit)
	//
	//	var gateways []string
	//	for _, targetRef := range externalRateLimit.Spec.TargetRefs {
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
