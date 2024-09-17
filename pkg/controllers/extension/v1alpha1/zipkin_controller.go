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

type zipkinReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *zipkinReconciler) NeedLeaderElection() bool {
	return true
}

// NewZipkinReconciler returns a new Zipkin Reconciler
func NewZipkinReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &zipkinReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Zipkin"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a Zipkin object and makes changes based on the state read
func (r *zipkinReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	zipkin := &extv1alpha1.Zipkin{}
	err := r.fctx.Get(ctx, req.NamespacedName, zipkin)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.Zipkin{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if zipkin.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(zipkin)
		return ctrl.Result{}, nil
	}

	// As Zipkin has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(zipkin, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *zipkinReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.Zipkin{}).
		Complete(r); err != nil {
		return err
	}

	return addZipkinIndexers(context.Background(), mgr)
}

func addZipkinIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerZipkin{}, constants.GatewayListenerZipkinIndex, func(obj client.Object) []string {
	//	zipkin := obj.(*extv1alpha1.ListenerZipkin)
	//
	//	var gateways []string
	//	for _, targetRef := range zipkin.Spec.TargetRefs {
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
