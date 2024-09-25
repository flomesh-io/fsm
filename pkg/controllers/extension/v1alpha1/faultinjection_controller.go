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

type faultInjectionReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *faultInjectionReconciler) NeedLeaderElection() bool {
	return true
}

// NewFaultInjectionReconciler returns a new FaultInjection Reconciler
func NewFaultInjectionReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &faultInjectionReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("FaultInjection"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a FaultInjection object and makes changes based on the state read
func (r *faultInjectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	faultInjection := &extv1alpha1.FaultInjection{}
	err := r.fctx.Get(ctx, req.NamespacedName, faultInjection)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.FaultInjection{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if faultInjection.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(faultInjection)
		return ctrl.Result{}, nil
	}

	// As FaultInjection has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(faultInjection, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *faultInjectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.FaultInjection{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.FaultInjection{}).
		Complete(r); err != nil {
		return err
	}

	return addFaultInjectionIndexers(context.Background(), mgr)
}

func addFaultInjectionIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerFaultInjection{}, constants.GatewayListenerFaultInjectionIndex, func(obj client.Object) []string {
	//	faultInjection := obj.(*extv1alpha1.ListenerFaultInjection)
	//
	//	var gateways []string
	//	for _, targetRef := range faultInjection.Spec.TargetRefs {
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
