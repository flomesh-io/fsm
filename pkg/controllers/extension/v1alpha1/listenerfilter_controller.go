package v1alpha1

import (
	"context"
	"fmt"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"

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

type listenerListenerFilterReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *listenerListenerFilterReconciler) NeedLeaderElection() bool {
	return true
}

// NewListenerFilterReconciler returns a new ListenerFilter Reconciler
func NewListenerFilterReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &listenerListenerFilterReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ListenerFilter"),
		fctx:     ctx,
		webhook:  webhook,
	}
}

// Reconcile reads that state of the cluster for a ListenerFilter object and makes changes based on the state read
func (r *listenerListenerFilterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	listenerListenerFilter := &extv1alpha1.ListenerFilter{}
	err := r.fctx.Get(ctx, req.NamespacedName, listenerListenerFilter)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&extv1alpha1.ListenerFilter{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if listenerListenerFilter.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(listenerListenerFilter)
		return ctrl.Result{}, nil
	}

	// As ListenerFilter has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(listenerListenerFilter, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *listenerListenerFilterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&extv1alpha1.ListenerFilter{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&extv1alpha1.ListenerFilter{}).
		Complete(r); err != nil {
		return err
	}

	return addListenerFilterIndexers(context.Background(), mgr)
}

func addListenerFilterIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerFilter{}, constants.GatewayListenerFilterIndex, gatewayPortListenerFilterIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerFilter{}, constants.FilterDefinitionListenerFilterIndex, filterDefinitionListenerFilterIndex); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &extv1alpha1.ListenerFilter{}, constants.ConfigListenerFilterIndex, configListenerFilterIndex); err != nil {
		return err
	}

	return nil
}

func configListenerFilterIndex(obj client.Object) []string {
	filter := obj.(*extv1alpha1.ListenerFilter)

	var configs []string

	if filter.Spec.ConfigRef != nil && filter.Spec.ConfigRef.Group == extv1alpha1.GroupName {
		configs = append(configs, fmt.Sprintf("%s/%s/%s", filter.Spec.ConfigRef.Kind, filter.Namespace, filter.Spec.ConfigRef.Name))
	}

	return configs
}

func filterDefinitionListenerFilterIndex(obj client.Object) []string {
	filter := obj.(*extv1alpha1.ListenerFilter)

	var definitions []string

	if filter.Spec.DefinitionRef != nil && filter.Spec.DefinitionRef.Group == extv1alpha1.GroupName &&
		filter.Spec.DefinitionRef.Kind == constants.GatewayAPIExtensionFilterDefinitionKind {
		definitions = append(definitions, fmt.Sprintf("%s/%s", filter.Namespace, filter.Spec.DefinitionRef.Name))
	}

	return definitions
}

func gatewayPortListenerFilterIndexFunc(obj client.Object) []string {
	filter := obj.(*extv1alpha1.ListenerFilter)

	var gateways []string
	for _, targetRef := range filter.Spec.TargetRefs {
		if string(targetRef.Kind) == constants.GatewayAPIGatewayKind &&
			string(targetRef.Group) == gwv1.GroupName {
			gateways = append(gateways, fmt.Sprintf("%s/%d", string(targetRef.Name), targetRef.Port))
		}
	}

	return gateways
}
