package v1alpha1

import (
	"context"
	"reflect"

	policystatus "github.com/flomesh-io/fsm/pkg/gateway/status/policy"

	"k8s.io/apimachinery/pkg/fields"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/upstreamtls"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type upstreamTLSPolicyReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *policystatus.ServicePolicyStatusProcessor
}

func (r *upstreamTLSPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewUpstreamTLSPolicyReconciler returns a new UpstreamTLSPolicy Reconciler
func NewUpstreamTLSPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &upstreamTLSPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("UpstreamTLSPolicy"),
		fctx:     ctx,
	}

	r.statusProcessor = &policystatus.ServicePolicyStatusProcessor{
		Client:              r.fctx.Client,
		Informer:            r.fctx.InformerCollection,
		GetAttachedPolicies: r.getAttachedUpstreamTLSPolices,
		FindConflict:        r.findConflict,
	}

	return r
}

// Reconcile reads that state of the cluster for a UpstreamTLSPolicy object and makes changes based on the state read
func (r *upstreamTLSPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.UpstreamTLSPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha1.UpstreamTLSPolicy{
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

	r.statusProcessor.Process(ctx, r.fctx.StatusUpdater, policystatus.NewPolicyUpdate(
		policy,
		&policy.ObjectMeta,
		&policy.TypeMeta,
		policy.Spec.TargetRef,
		policy.Status.Conditions,
	))

	r.fctx.GatewayEventHandler.OnAdd(policy, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *upstreamTLSPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.UpstreamTLSPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r); err != nil {
		return err
	}

	return addUpstreamTLSPolicyIndexer(context.Background(), mgr)
}

func addUpstreamTLSPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.UpstreamTLSPolicy{}, constants.ServicePolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwpav1alpha1.UpstreamTLSPolicy)
		targetRef := policy.Spec.TargetRef
		var targets []string
		if targetRef.Kind == constants.KubernetesServiceKind {
			targets = append(targets, types.NamespacedName{
				Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
				Name:      string(targetRef.Name),
			}.String())
		}

		return targets
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.UpstreamTLSPolicy{}, constants.SecretUpstreamTLSPolicyIndex, addSecretUpstreamTLSPolicyFunc); err != nil {
		return err
	}

	return nil
}

func addSecretUpstreamTLSPolicyFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.UpstreamTLSPolicy)
	secrets := make([]string, 0)

	if policy.Spec.DefaultConfig != nil {
		ref := policy.Spec.DefaultConfig.CertificateRef
		kind := ref.Kind
		if kind == nil || string(*kind) == constants.KubernetesSecretKind {
			secrets = append(secrets, types.NamespacedName{
				Namespace: gwutils.NamespaceDerefOr(ref.Namespace, policy.Namespace),
				Name:      string(ref.Name),
			}.String())
		}
	}

	if len(policy.Spec.Ports) > 0 {
		for _, port := range policy.Spec.Ports {
			if port.Config == nil {
				continue
			}

			ref := port.Config.CertificateRef
			kind := ref.Kind
			if kind == nil || string(*kind) == constants.KubernetesSecretKind {
				secrets = append(secrets, types.NamespacedName{
					Namespace: gwutils.NamespaceDerefOr(ref.Namespace, policy.Namespace),
					Name:      string(ref.Name),
				}.String())
			}
		}
	}

	return secrets
}

func (r *upstreamTLSPolicyReconciler) getAttachedUpstreamTLSPolices(svc client.Object) ([]client.Object, *metav1.Condition) {
	c := r.fctx.Manager.GetCache()
	key := client.ObjectKeyFromObject(svc).String()
	selector := fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, key)

	return gwutils.GetUpStreamTLSes(c, selector), nil
}

func (r *upstreamTLSPolicyReconciler) findConflict(upstreamTLSPolicy client.Object, allUpstreamTLSPolicies []client.Object, port int32) *types.NamespacedName {
	currentPolicy := upstreamTLSPolicy.(*gwpav1alpha1.UpstreamTLSPolicy)

	for _, policy := range allUpstreamTLSPolicies {
		policy := policy.(*gwpav1alpha1.UpstreamTLSPolicy)

		c1 := upstreamtls.GetUpstreamTLSConfigIfPortMatchesPolicy(port, *policy)
		if c1 == nil {
			continue
		}

		c2 := upstreamtls.GetUpstreamTLSConfigIfPortMatchesPolicy(port, *currentPolicy)
		if c2 == nil {
			continue
		}

		if reflect.DeepEqual(c1, c2) {
			continue
		}

		return &types.NamespacedName{
			Name:      policy.Name,
			Namespace: policy.Namespace,
		}
	}

	return nil
}

func (r *upstreamTLSPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	list := &gwpav1alpha1.UpstreamTLSPolicyList{}
	if err := c.List(context.Background(), list); err != nil {
		log.Error().Msgf("Failed to list UpstreamTLSPolicyList: %v", err)
		return nil
	}
	policies := gwutils.ToSlicePtr(list.Items)

	requests := make([]reconcile.Request, 0)
	for _, policy := range policies {
		if gwutils.HasAccessToTargetRef(policy, policy.Spec.TargetRef, []*gwv1beta1.ReferenceGrant{refGrant}) {
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
