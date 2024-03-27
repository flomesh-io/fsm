package v1alpha1

import (
	"context"
	"fmt"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/loadbalancer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

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
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.LoadBalancerPolicy{}).
		Complete(r)
}

func (r *loadBalancerPolicyReconciler) getAttachedLoadBalancers(policy client.Object, svc client.Object) ([]client.Object, *metav1.Condition) {
	loadBalancerPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().LoadBalancerPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list LoadBalancerPolicies: %s", err)))
	}

	loadBalancers := make([]client.Object, 0)
	for _, p := range loadBalancerPolicyList.Items {
		p := p
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
			gwutils.IsRefToTarget(p.Spec.TargetRef, svc) {
			loadBalancers = append(loadBalancers, &p)
		}
	}

	return loadBalancers, nil
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
