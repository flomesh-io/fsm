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

type filterDefinitionReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *filterDefinitionReconciler) NeedLeaderElection() bool {
	return true
}

// NewFilterDefinitionReconciler returns a new FilterDefinition Reconciler
func NewFilterDefinitionReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &filterDefinitionReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("FilterDefinition"),
		fctx:     ctx,
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
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.FilterDefinition{}).
		Complete(r); err != nil {
		return err
	}

	return addFilterDefinitionIndexers(context.Background(), mgr)
}

func addFilterDefinitionIndexers(ctx context.Context, mgr manager.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.FilterDefinition{}, constants.GatewayFilterDefinitionIndex, func(obj client.Object) []string {
	//	filterDefinition := obj.(*extv1alpha1.FilterDefinition)
	//
	//	scope := ptr.Deref(filterDefinition.Spec.Scope, extv1alpha1.FilterDefinitionScopeRoute)
	//	if scope != extv1alpha1.FilterDefinitionScopeListener {
	//		return nil
	//	}
	//
	//	var gateways []string
	//	for _, targetRef := range filterDefinition.Spec.TargetRefs {
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
