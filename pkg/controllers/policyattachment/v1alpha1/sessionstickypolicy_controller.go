package v1alpha1

import (
	"context"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"

	policyAttachmentApiClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
)

type sessionStickyPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
}

// NewSessionStickyPolicyReconciler returns a new SessionStickyPolicy Reconciler
func NewSessionStickyPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &sessionStickyPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("SessionStickyPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a SessionStickyPolicy object and makes changes based on the state read
func (r *sessionStickyPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.SessionStickyPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.SessionStickyPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if policy.DeletionTimestamp != nil {
		r.fctx.EventHandler.OnDelete(policy)
		return ctrl.Result{}, nil
	}

	metautil.SetStatusCondition(&policy.Status.Conditions, r.getStatusCondition(ctx, policy))
	if err := r.fctx.Status().Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.EventHandler.OnAdd(policy)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *sessionStickyPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.SessionStickyPolicy{}).
		Complete(r)
}
