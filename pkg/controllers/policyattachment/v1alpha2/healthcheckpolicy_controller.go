package v1alpha2

import (
	"context"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/util/sets"

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

type healthCheckPolicyReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	webhook  whtypes.Register
}

func (r *healthCheckPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewHealthCheckPolicyReconciler returns a new HealthCheckPolicy Reconciler
func NewHealthCheckPolicyReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	r := &healthCheckPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("HealthCheckPolicy"),
		fctx:     ctx,
		webhook:  webhook,
	}

	return r
}

// Reconcile reads that state of the cluster for a HealthCheckPolicy object and makes changes based on the state read
func (r *healthCheckPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha2.HealthCheckPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha2.HealthCheckPolicy{
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
func (r *healthCheckPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwpav1alpha2.HealthCheckPolicy{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha2.HealthCheckPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r); err != nil {
		return err
	}

	return addHealthCheckPolicyIndexer(context.Background(), mgr)
}

func addHealthCheckPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha2.HealthCheckPolicy{}, constants.ServicePolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwpav1alpha2.HealthCheckPolicy)

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

func (r *healthCheckPolicyReconciler) referenceGrantToPolicyAttachment(ctx context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	namespaces := sets.New[string]()
	for _, from := range refGrant.Spec.From {
		if from.Group == gwpav1alpha2.GroupName && from.Kind == constants.HealthCheckPolicyKind {
			namespaces.Insert(string(from.Namespace))
		}
	}

	c := r.fctx.Manager.GetCache()
	items := make([]gwpav1alpha2.HealthCheckPolicy, 0)
	for ns := range namespaces {
		list := &gwpav1alpha2.HealthCheckPolicyList{}
		if err := c.List(ctx, list, client.InNamespace(ns)); err != nil {
			log.Error().Msgf("Failed to list HealthCheckPolicyList: %v", err)
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

func isConcernedPolicy(policy client.Object, targetRefs []gwv1alpha2.NamespacedPolicyTargetReference, refGrant *gwv1beta1.ReferenceGrant) bool {
	for _, targetRef := range targetRefs {
		if gwutils.HasAccessToTargetRef(policy, targetRef, []*gwv1beta1.ReferenceGrant{refGrant}) {
			return true
		}
	}

	return false
}
