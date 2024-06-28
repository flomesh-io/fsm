package v1alpha3

import (
	"context"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

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

type backendTLSPolicyReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
	//statusProcessor *policystatus.ServicePolicyStatusProcessor
}

func (r *backendTLSPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewBackendTLSPolicyReconciler returns a new BackendTLSPolicy Reconciler
func NewBackendTLSPolicyReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	r := &backendTLSPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("BackendTLSPolicy"),
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

// Reconcile reads that state of the cluster for a BackendTLSPolicy object and makes changes based on the state read
func (r *backendTLSPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwv1alpha3.BackendTLSPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1alpha3.BackendTLSPolicy{
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
func (r *backendTLSPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&gwv1alpha3.BackendTLSPolicy{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha3.BackendTLSPolicy{}).
		Complete(r); err != nil {
		return err
	}

	return addBackendTLSPolicyIndexer(context.Background(), mgr)
}

func addBackendTLSPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha3.BackendTLSPolicy{}, constants.ServicePolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwv1alpha3.BackendTLSPolicy)

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

//func (r *backendTLSPolicyReconciler) getAttachedRetryPolicies(svc client.Object) ([]client.Object, *metav1.Condition) {
//	c := r.fctx.Manager.GetCache()
//	key := client.ObjectKeyFromObject(svc).String()
//	selector := fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, key)
//
//	return gwutils.GetRetries(c, selector), nil
//}
//
//func (r *backendTLSPolicyReconciler) findConflict(backendTLSPolicy client.Object, allRetryPolicies []client.Object, port int32) *types.NamespacedName {
//	currentPolicy := backendTLSPolicy.(*gwpav1alpha1.BackendTLSPolicy)
//
//	for _, policy := range allRetryPolicies {
//		policy := policy.(*gwpav1alpha1.BackendTLSPolicy)
//
//		c1 := backendTLS.GetRetryConfigIfPortMatchesPolicy(port, *policy)
//		if c1 == nil {
//			continue
//		}
//
//		c2 := backendTLS.GetRetryConfigIfPortMatchesPolicy(port, *currentPolicy)
//		if c2 == nil {
//			continue
//		}
//
//		if reflect.DeepEqual(c1, c2) {
//			continue
//		}
//
//		return &types.NamespacedName{
//			Name:      policy.Name,
//			Namespace: policy.Namespace,
//		}
//	}
//
//	return nil
//}
