package v1alpha1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/sessionsticky"

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

type sessionStickyPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
	statusProcessor           *status.ServicePolicyStatusProcessor
}

func (r *sessionStickyPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewSessionStickyPolicyReconciler returns a new SessionStickyPolicy Reconciler
func NewSessionStickyPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &sessionStickyPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("SessionStickyPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}

	r.statusProcessor = &status.ServicePolicyStatusProcessor{
		Client:              r.fctx.Client,
		GetAttachedPolicies: r.getAttachedSessionStickies,
		FindConflict:        r.findConflict,
	}

	return r
}

// Reconcile reads that state of the cluster for a SessionStickyPolicy object and makes changes based on the state read
func (r *sessionStickyPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.SessionStickyPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha1.SessionStickyPolicy{
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
func (r *sessionStickyPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.SessionStickyPolicy{}).
		Complete(r)
}

func (r *sessionStickyPolicyReconciler) getAttachedSessionStickies(policy client.Object, svc client.Object) ([]client.Object, *metav1.Condition) {
	sessionStickyPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().SessionStickyPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list SessionStickyPolicies: %s", err)))
	}

	sessionStickies := make([]client.Object, 0)
	for _, p := range sessionStickyPolicyList.Items {
		p := p
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
			gwutils.IsRefToTarget(p.Spec.TargetRef, svc) {
			sessionStickies = append(sessionStickies, &p)
		}
	}

	return sessionStickies, nil
}

func (r *sessionStickyPolicyReconciler) findConflict(sessionStickyPolicy client.Object, allSessionStickyPolicies []client.Object, port int32) *types.NamespacedName {
	currentPolicy := sessionStickyPolicy.(*gwpav1alpha1.SessionStickyPolicy)

	for _, policy := range allSessionStickyPolicies {
		policy := policy.(*gwpav1alpha1.SessionStickyPolicy)

		c1 := sessionsticky.GetSessionStickyConfigIfPortMatchesPolicy(port, *policy)
		if c1 == nil {
			continue
		}

		c2 := sessionsticky.GetSessionStickyConfigIfPortMatchesPolicy(port, *currentPolicy)
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
