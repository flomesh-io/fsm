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

type proxyTagReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *proxyTagReconciler) NeedLeaderElection() bool {
	return true
}

// NewProxyTagReconciler returns a new ProxyTag Reconciler
func NewProxyTagReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &proxyTagReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ProxyTag"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a ProxyTag object and makes changes based on the state read
func (r *proxyTagReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	proxyTag := &extv1alpha1.ProxyTag{}
	err := r.fctx.Get(ctx, req.NamespacedName, proxyTag)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.ProxyTag{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if proxyTag.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(proxyTag)
		return ctrl.Result{}, nil
	}

	// As ProxyTag has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(proxyTag, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *proxyTagReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.ProxyTag{}).
		Complete(r); err != nil {
		return err
	}

	return addProxyTagIndexers(context.Background(), mgr)
}

func addProxyTagIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerProxyTag{}, constants.GatewayListenerProxyTagIndex, func(obj client.Object) []string {
	//	proxyTag := obj.(*extv1alpha1.ListenerProxyTag)
	//
	//	var gateways []string
	//	for _, targetRef := range proxyTag.Spec.TargetRefs {
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
