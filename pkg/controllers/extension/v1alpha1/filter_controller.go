package v1alpha1

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/flomesh-io/fsm/pkg/constants"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type filterReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *filterReconciler) NeedLeaderElection() bool {
	return true
}

// NewFilterReconciler returns a new Filter Reconciler
func NewFilterReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &filterReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Filter"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a Filter object and makes changes based on the state read
func (r *filterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	filter := &extv1alpha1.Filter{}
	err := r.fctx.Get(ctx, req.NamespacedName, filter)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.Filter{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if filter.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(filter)
		return ctrl.Result{}, nil
	}

	// As Filter has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(filter, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *filterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.Filter{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.Filter{}).
		Complete(r); err != nil {
		return err
	}

	return addFilterIndexers(context.Background(), mgr)
}

func addFilterIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.Filter{}, constants.FilterDefinitionFilterIndex, filterDefinitionFilterIndex); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.Filter{}, constants.ConfigFilterIndex, configFilterIndex); err != nil {
		return err
	}

	return nil
}

func filterDefinitionFilterIndex(obj client.Object) []string {
	filter := obj.(*extv1alpha1.Filter)

	var definitions []string

	if filter.Spec.DefinitionRef != nil &&
		filter.Spec.DefinitionRef.Group == extv1alpha1.GroupName &&
		filter.Spec.DefinitionRef.Kind == constants.GatewayAPIExtensionFilterDefinitionKind {
		definitions = append(definitions, fmt.Sprintf("%s/%s", filter.Namespace, filter.Spec.DefinitionRef.Name))
	}

	return definitions
}

func configFilterIndex(obj client.Object) []string {
	filter := obj.(*extv1alpha1.Filter)

	var configs []string

	if filter.Spec.ConfigRef != nil && filter.Spec.ConfigRef.Group == extv1alpha1.GroupName {
		configs = append(configs, fmt.Sprintf("%s/%s/%s", filter.Spec.ConfigRef.Kind, filter.Namespace, filter.Spec.ConfigRef.Name))
	}

	return configs
}
