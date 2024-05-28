package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/loadbalancer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"k8s.io/apimachinery/pkg/types"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"

	policyAttachmentApiClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
)

type loadBalancerPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
	statusProcessor           *status.ServicePolicyStatusProcessor
}

func (r *loadBalancerPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewLoadBalancerPolicyReconciler returns a new LoadBalancerPolicy Reconciler
func NewLoadBalancerPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &loadBalancerPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("LoadBalancerPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}

	r.statusProcessor = &status.ServicePolicyStatusProcessor{
		Client:              r.fctx.Client,
		Informer:            r.fctx.InformerCollection,
		GetAttachedPolicies: r.getAttachedLoadBalancers,
		FindConflict:        r.findConflict,
	}

	return r
}

// Reconcile reads that state of the cluster for a LoadBalancerPolicy object and makes changes based on the state read
func (r *loadBalancerPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.LoadBalancerPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha1.LoadBalancerPolicy{
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

	metautil.SetStatusCondition(
		&policy.Status.Conditions,
		r.statusProcessor.Process(ctx, policy, policy.Spec.TargetRef),
	)
	if err := r.fctx.Status().Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.GatewayEventHandler.OnAdd(policy, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *loadBalancerPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.LoadBalancerPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r); err != nil {
		return err
	}

	return addLoadBalancerPolicyIndexer(context.Background(), mgr)
}

func addLoadBalancerPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.LoadBalancerPolicy{}, constants.ServicePolicyAttachmentIndex, func(obj client.Object) []string {
		policy := obj.(*gwpav1alpha1.LoadBalancerPolicy)
		targetRef := policy.Spec.TargetRef
		var targets []string
		if targetRef.Kind == constants.KubernetesServiceKind {
			targets = append(targets, types.NamespacedName{
				Namespace: gwutils.Namespace(targetRef.Namespace, policy.Namespace),
				Name:      string(targetRef.Name),
			}.String())
		}

		return targets
	}); err != nil {
		return err
	}

	return nil
}

func (r *loadBalancerPolicyReconciler) getAttachedLoadBalancers(svc client.Object) ([]client.Object, *metav1.Condition) {
	c := r.fctx.Manager.GetCache()
	key := client.ObjectKeyFromObject(svc).String()
	selector := fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, key)

	return gwutils.GetLoadBalancers(c, selector), nil

	//loadBalancerPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().LoadBalancerPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	//if err != nil {
	//	return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list LoadBalancerPolicies: %s", err)))
	//}
	//
	//loadBalancers := make([]client.Object, 0)
	//referenceGrants := r.fctx.InformerCollection.GetGatewayResourcesFromCache(informers.ReferenceGrantResourceType, false)
	//
	//for _, p := range loadBalancerPolicyList.Items {
	//	p := p
	//	if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
	//		gwutils.HasAccessToTarget(referenceGrants, &p, p.Spec.TargetRef, svc) {
	//		loadBalancers = append(loadBalancers, &p)
	//	}
	//}
	//
	//return loadBalancers, nil
	//
	//c := r.fctx.Manager.GetCache()
	//
	//referenceGrants, cond := gwutils.getServiceRefGrants(c, policy)
	//if cond != nil {
	//	return nil, cond
	//}
	//
	//policyList := &gwpav1alpha1.LoadBalancerPolicyList{}
	//if err := c.List(context.Background(), policyList, &client.ListOptions{
	//	FieldSelector: fields.OneTermEqualSelector(constants.ServicePolicyAttachmentIndex, client.ObjectKeyFromObject(svc).String()),
	//}); err != nil {
	//	return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list LoadBalancerPolicyList: %s", err)))
	//}
	//
	//return gwutils.filterValidPolicies(
	//	gwutils.toClientObjects(gwutils.ToSlicePtr(policyList.Items)),
	//	svc,
	//	referenceGrants,
	//	func(policy client.Object) bool {
	//		p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
	//		return gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions)
	//	},
	//	func(policy client.Object) bool {
	//		return false
	//	},
	//	func(policy client.Object, target client.Object) bool {
	//		p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
	//		return gwutils.IsTargetRefToTarget(p.Spec.TargetRef, target)
	//	},
	//	func(policy client.Object, refGrants []*gwv1beta1.ReferenceGrant) bool {
	//		p := policy.(*gwpav1alpha1.LoadBalancerPolicy)
	//		return gwutils.HasAccessToTargetRef(p, p.Spec.TargetRef, refGrants)
	//	},
	//), nil
}

func (r *loadBalancerPolicyReconciler) findConflict(loadBalancerPolicy client.Object, allSessionStickyPolicies []client.Object, port int32) *types.NamespacedName {
	currentPolicy := loadBalancerPolicy.(*gwpav1alpha1.LoadBalancerPolicy)

	for _, policy := range allSessionStickyPolicies {
		policy := policy.(*gwpav1alpha1.LoadBalancerPolicy)

		t1 := loadbalancer.GetLoadBalancerTypeIfPortMatchesPolicy(port, *policy)
		if t1 == nil {
			continue
		}

		t2 := loadbalancer.GetLoadBalancerTypeIfPortMatchesPolicy(port, *currentPolicy)
		if t2 == nil {
			continue
		}

		if *t1 == *t2 {
			continue
		}

		return &types.NamespacedName{
			Name:      policy.Name,
			Namespace: policy.Namespace,
		}
	}

	return nil
}

func (r *loadBalancerPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	list := &gwpav1alpha1.LoadBalancerPolicyList{}
	if err := c.List(context.Background(), list); err != nil {
		log.Error().Msgf("Failed to list LoadBalancerPolicyList: %v", err)
		return nil
	}
	policies := gwutils.ToSlicePtr(list.Items)

	requests := make([]reconcile.Request, 0)
	//policies := r.fctx.InformerCollection.GetGatewayResourcesFromCache(informers.LoadBalancerPoliciesResourceType, false)

	for _, policy := range policies {
		//policy := p.(*gwpav1alpha1.LoadBalancerPolicy)

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
