/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"
	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/gatewaytls"

	corev1 "k8s.io/api/core/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/apimachinery/pkg/types"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"

	policyAttachmentApiClientset "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
)

type gatewayTLSPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
	statusProcessor           *status.PolicyStatusProcessor
}

func (r *gatewayTLSPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewGatewayTLSPolicyReconciler returns a new GatewayTLSPolicy Reconciler
func NewGatewayTLSPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &gatewayTLSPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("GatewayTLSPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}

	r.statusProcessor = &status.PolicyStatusProcessor{
		Client:           r.fctx.Client,
		GetPolicies:      r.getGatewayTLSPolices,
		FindConflictPort: r.getConflictedPort,
		GroupKindObjectMapping: map[string]map[string]client.Object{
			constants.GatewayAPIGroup: {
				constants.GatewayAPIGatewayKind: &gwv1beta1.Gateway{},
			},
		},
	}

	return r
}

// Reconcile reads that state of the cluster for a GatewayTLSPolicy object and makes changes based on the state read
func (r *gatewayTLSPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.GatewayTLSPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.GatewayTLSPolicy{
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

	metautil.SetStatusCondition(
		&policy.Status.Conditions,
		r.statusProcessor.Process(ctx, policy, policy.Spec.TargetRef),
	)
	if err := r.fctx.Status().Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.EventHandler.OnAdd(policy)

	return ctrl.Result{}, nil
}

func (r *gatewayTLSPolicyReconciler) getGatewayTLSPolices(policy client.Object, target client.Object) (map[gwpkg.PolicyMatchType][]client.Object, *metav1.Condition) {
	gatewayTLSPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().GatewayTLSPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list GatewayTLSPolicies: %s", err)))
	}

	policies := make(map[gwpkg.PolicyMatchType][]client.Object)

	for _, p := range gatewayTLSPolicyList.Items {
		p := p
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
			spec := p.Spec
			targetRef := spec.TargetRef

			switch {
			case gwutils.IsTargetRefToGVK(targetRef, constants.GatewayGVK) &&
				gwutils.IsRefToTarget(targetRef, target) &&
				len(spec.Ports) > 0:
				policies[gwpkg.PolicyMatchTypePort] = append(policies[gwpkg.PolicyMatchTypePort], &p)
			}
		}
	}

	return policies, nil
}

func (r *gatewayTLSPolicyReconciler) getConflictedPort(gateway *gwv1beta1.Gateway, gatewayTLSPolicy client.Object, allGatewayTLSPolicies []client.Object) *types.NamespacedName {
	currentGatewayTLSPolicy := gatewayTLSPolicy.(*gwpav1alpha1.GatewayTLSPolicy)

	if len(currentGatewayTLSPolicy.Spec.Ports) == 0 {
		return nil
	}

	validListeners := gwutils.GetValidListenersFromGateway(gateway)
	for _, pr := range allGatewayTLSPolicies {
		pr := pr.(*gwpav1alpha1.GatewayTLSPolicy)

		if len(pr.Spec.Ports) > 0 {
			for _, listener := range validListeners {
				r1 := gatewaytls.GetGatewayTLSConfigIfPortMatchesPolicy(listener.Port, *pr)
				if r1 == nil {
					continue
				}

				r2 := gatewaytls.GetGatewayTLSConfigIfPortMatchesPolicy(listener.Port, *currentGatewayTLSPolicy)
				if r2 == nil {
					continue
				}

				if reflect.DeepEqual(r1, r2) {
					continue
				}

				return &types.NamespacedName{
					Name:      pr.Name,
					Namespace: pr.Namespace,
				}
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayTLSPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.GatewayTLSPolicy{}).
		Complete(r)
}
