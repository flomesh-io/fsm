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

type dnsModifierReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *dnsModifierReconciler) NeedLeaderElection() bool {
	return true
}

// NewDNSModifierReconciler returns a new DNSModifier Reconciler
func NewDNSModifierReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &dnsModifierReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("DNSModifier"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a DNSModifier object and makes changes based on the state read
func (r *dnsModifierReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	dnsModifier := &extv1alpha1.DNSModifier{}
	err := r.fctx.Get(ctx, req.NamespacedName, dnsModifier)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.DNSModifier{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if dnsModifier.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(dnsModifier)
		return ctrl.Result{}, nil
	}

	// As DNSModifier has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(dnsModifier, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *dnsModifierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.DNSModifier{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.DNSModifier{}).
		Complete(r); err != nil {
		return err
	}

	return addDNSModifierIndexers(context.Background(), mgr)
}

func addDNSModifierIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerDNSModifier{}, constants.GatewayListenerDNSModifierIndex, func(obj client.Object) []string {
	//	dnsModifier := obj.(*extv1alpha1.ListenerDNSModifier)
	//
	//	var gateways []string
	//	for _, targetRef := range dnsModifier.Spec.TargetRefs {
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
