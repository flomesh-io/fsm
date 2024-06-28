package v1alpha2

import (
	"context"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type backendLBPolicyReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	//statusProcessor *policystatus.ServicePolicyStatusProcessor
	webhook whtypes.Register
}

func (r *backendLBPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewBackendLBPolicyReconciler returns a new BackendLBPolicy Reconciler
func NewBackendLBPolicyReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	r := &backendLBPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("BackendLBPolicy"),
		fctx:     ctx,
		webhook:  webhook,
	}

	//r.statusProcessor = &policystatus.ServicePolicyStatusProcessor{
	//	Client:              r.fctx.Client,
	//	Informer:            r.fctx.InformerCollection,
	//	GetAttachedPolicies: r.getAttachedRetryPolicies,
	//	FindConflict:        r.findConflict,
	//}

	return r
}

// Reconcile reads that state of the cluster for a BackendLBPolicy object and makes changes based on the state read
func (r *backendLBPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwv1alpha2.BackendLBPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1alpha2.BackendLBPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if policy.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(policy)
		return ctrl.Result{}, nil
	}

	//r.statusProcessor.Process(ctx, r.fctx.StatusUpdater, policystatus.NewPolicyUpdate(
	//	policy,
	//	&policy.ObjectMeta,
	//	&policy.TypeMeta,
	//	policy.Spec.TargetRef,
	//	policy.Status.Conditions,
	//))

	r.fctx.GatewayEventHandler.OnAdd(policy, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *backendLBPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwv1alpha2.BackendLBPolicy{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha2.BackendLBPolicy{}).
		Complete(r); err != nil {
		return err
	}

	return addBackendLBPolicyIndexer(context.Background(), mgr)
}

func addBackendLBPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.BackendLBPolicy{}, constants.ServicePolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwv1alpha2.BackendLBPolicy)

		var targets []string
		for _, targetRef := range policy.Spec.TargetRefs {
			if targetRef.Kind == constants.KubernetesServiceKind {
				targets = append(targets, types.NamespacedName{
					Namespace: policy.Namespace,
					Name:      string(targetRef.Name),
				}.String())
			}
		}

		return targets
	}); err != nil {
		return err
	}

	return nil
}
