package v1alpha2

import (
	"context"

	"k8s.io/apimachinery/pkg/util/sets"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	"github.com/rs/zerolog/log"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type retryPolicyReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *retryPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewRetryPolicyReconciler returns a new RetryPolicy Reconciler
func NewRetryPolicyReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	r := &retryPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("RetryPolicy"),
		fctx:     ctx,
		webhook:  webhook,
	}

	return r
}

// Reconcile reads that state of the cluster for a RetryPolicy object and makes changes based on the state read
func (r *retryPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha2.RetryPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha2.RetryPolicy{
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

	r.fctx.GatewayEventHandler.OnAdd(policy, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *retryPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwpav1alpha2.RetryPolicy{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha2.RetryPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r); err != nil {
		return err
	}

	return addRetryPolicyIndexer(context.Background(), mgr)
}

func addRetryPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha2.RetryPolicy{}, constants.ServicePolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwpav1alpha2.RetryPolicy)

		var targets []string
		for _, targetRef := range policy.Spec.TargetRefs {
			if targetRef.Kind == constants.KubernetesServiceKind {
				targets = append(targets, types.NamespacedName{
					Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
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

//func (r *retryPolicyReconciler) getAttachedRetryPolicies(svc client.Object) ([]client.Object, *metav1.Condition) {
//	c := r.fctx.Manager.GetCache()
//	key := client.ObjectKeyFromObject(svc).String()
//	selector := fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, key)
//
//	return gwutils.GetRetries(c, selector), nil
//}
//
//func (r *retryPolicyReconciler) findConflict(retryPolicy client.Object, allRetryPolicies []client.Object, port int32) *types.NamespacedName {
//	currentPolicy := retryPolicy.(*gwpav1alpha2.RetryPolicy)
//
//	for _, policy := range allRetryPolicies {
//		policy := policy.(*gwpav1alpha2.RetryPolicy)
//
//		c1 := retry.GetRetryConfigIfPortMatchesPolicy(port, *policy)
//		if c1 == nil {
//			continue
//		}
//
//		c2 := retry.GetRetryConfigIfPortMatchesPolicy(port, *currentPolicy)
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

func (r *retryPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	namespaces := sets.New[string]()
	for _, from := range refGrant.Spec.From {
		if from.Group == gwpav1alpha2.GroupName && from.Kind == constants.RetryPolicyKind {
			namespaces.Insert(string(from.Namespace))
		}
	}

	c := r.fctx.Manager.GetCache()
	items := make([]gwpav1alpha2.RetryPolicy, 0)
	for ns := range namespaces {
		list := &gwpav1alpha2.RetryPolicyList{}
		if err := c.List(context.Background(), list, client.InNamespace(ns)); err != nil {
			log.Error().Msgf("Failed to list RetryPolicyList: %v", err)
			continue
		}

		items = append(items, list.Items...)
	}

	requests := make([]reconcile.Request, 0)
	for _, policy := range gwutils.ToSlicePtr(items) {
		if isConcernedPolicy(policy, policy.Spec.TargetRefs, refGrant) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      policy.Name,
					Namespace: policy.Namespace,
				},
			})
		}
	}

	return requests
}
