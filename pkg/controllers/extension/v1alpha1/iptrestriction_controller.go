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

type ipRestrictionReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *ipRestrictionReconciler) NeedLeaderElection() bool {
	return true
}

// NewIPRestrictionReconciler returns a new IPRestriction Reconciler
func NewIPRestrictionReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &ipRestrictionReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("IPRestriction"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a IPRestriction object and makes changes based on the state read
func (r *ipRestrictionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ipRestriction := &extv1alpha1.IPRestriction{}
	err := r.fctx.Get(ctx, req.NamespacedName, ipRestriction)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.IPRestriction{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if ipRestriction.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(ipRestriction)
		return ctrl.Result{}, nil
	}

	// As IPRestriction has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(ipRestriction, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ipRestrictionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.IPRestriction{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.IPRestriction{}).
		Complete(r); err != nil {
		return err
	}

	return addIPRestrictionIndexers(context.Background(), mgr)
}

func addIPRestrictionIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerIPRestriction{}, constants.GatewayListenerIPRestrictionIndex, func(obj client.Object) []string {
	//	ipRestriction := obj.(*extv1alpha1.ListenerIPRestriction)
	//
	//	var gateways []string
	//	for _, targetRef := range ipRestriction.Spec.TargetRefs {
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
