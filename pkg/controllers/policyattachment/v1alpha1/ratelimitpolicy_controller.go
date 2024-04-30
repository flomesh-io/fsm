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

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"

	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

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

type rateLimitPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
	statusProcessor           *status.PolicyStatusProcessor
}

func (r *rateLimitPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewRateLimitPolicyReconciler returns a new RateLimitPolicy Reconciler
func NewRateLimitPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &rateLimitPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("RateLimitPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}

	r.statusProcessor = &status.PolicyStatusProcessor{
		Client:                             r.fctx.Client,
		Informer:                           r.fctx.InformerCollection,
		GetPolicies:                        r.getRateLimitPolices,
		FindConflictPort:                   r.getConflictedPort,
		FindConflictedHostnamesBasedPolicy: r.getConflictedHostnamesBasedRateLimitPolicy,
		FindConflictedHTTPRouteBasedPolicy: r.getConflictedHTTPRouteBasedRateLimitPolicy,
		FindConflictedGRPCRouteBasedPolicy: r.getConflictedGRPCRouteBasedRateLimitPolicy,
	}

	return r
}

// Reconcile reads that state of the cluster for a RateLimitPolicy object and makes changes based on the state read
func (r *rateLimitPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.RateLimitPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha1.RateLimitPolicy{
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

func (r *rateLimitPolicyReconciler) getRateLimitPolices(policy client.Object, target client.Object) (map[gwpkg.PolicyMatchType][]client.Object, *metav1.Condition) {
	rateLimitPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list rate limit policies: %s", err)))
	}

	policies := make(map[gwpkg.PolicyMatchType][]client.Object)
	referenceGrants := r.fctx.InformerCollection.GetGatewayResourcesFromCache(informers.ReferenceGrantResourceType, false)

	for _, p := range rateLimitPolicyList.Items {
		p := p
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) {
			spec := p.Spec
			targetRef := spec.TargetRef

			switch {
			case gwutils.IsTargetRefToGVK(targetRef, constants.GatewayGVK) &&
				gwutils.IsRefToTarget(referenceGrants, &p, targetRef, target) &&
				len(spec.Ports) > 0:
				policies[gwpkg.PolicyMatchTypePort] = append(policies[gwpkg.PolicyMatchTypePort], &p)
			case (gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK)) &&
				gwutils.IsRefToTarget(referenceGrants, &p, targetRef, target) &&
				len(spec.Hostnames) > 0:
				policies[gwpkg.PolicyMatchTypeHostnames] = append(policies[gwpkg.PolicyMatchTypeHostnames], &p)
			case gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) &&
				gwutils.IsRefToTarget(referenceGrants, &p, targetRef, target) &&
				len(spec.HTTPRateLimits) > 0:
				policies[gwpkg.PolicyMatchTypeHTTPRoute] = append(policies[gwpkg.PolicyMatchTypeHTTPRoute], &p)
			case gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) &&
				gwutils.IsRefToTarget(referenceGrants, &p, targetRef, target) &&
				len(spec.GRPCRateLimits) > 0:
				policies[gwpkg.PolicyMatchTypeGRPCRoute] = append(policies[gwpkg.PolicyMatchTypeGRPCRoute], &p)
			}
		}
	}

	return policies, nil
}

func (r *rateLimitPolicyReconciler) getConflictedHostnamesBasedRateLimitPolicy(route *gwtypes.RouteContext, rateLimitPolicy client.Object, hostnamesRateLimits []client.Object) *types.NamespacedName {
	currentPolicy := rateLimitPolicy.(*gwpav1alpha1.RateLimitPolicy)

	if len(currentPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for _, parent := range route.ParentStatus {
		if metautil.IsStatusConditionTrue(parent.Conditions, string(gwv1.RouteConditionAccepted)) {
			key := getRouteParentKey(route.Meta, parent)

			gateway := &gwv1.Gateway{}
			if err := r.fctx.Get(context.TODO(), key, gateway); err != nil {
				continue
			}

			validListeners := gwutils.GetValidListenersFromGateway(gateway)

			allowedListeners, _ := gwutils.GetAllowedListeners(r.fctx.InformerCollection.GetListers().Namespace, gateway, parent.ParentRef, route, validListeners)
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, route.Hostnames)
				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}
				for _, hostname := range hostnames {
					for _, hr := range hostnamesRateLimits {
						hr := hr.(*gwpav1alpha1.RateLimitPolicy)

						r1 := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, *hr)
						if r1 == nil {
							continue
						}

						r2 := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, *currentPolicy)
						if r2 == nil {
							continue
						}

						if reflect.DeepEqual(r1, r2) {
							continue
						}

						return &types.NamespacedName{
							Name:      hr.Name,
							Namespace: hr.Namespace,
						}
					}
				}
			}
		}
	}

	return nil
}

func (r *rateLimitPolicyReconciler) getConflictedHTTPRouteBasedRateLimitPolicy(route *gwv1.HTTPRoute, rateLimitPolicy client.Object, routeRateLimits []client.Object) *types.NamespacedName {
	currentPolicy := rateLimitPolicy.(*gwpav1alpha1.RateLimitPolicy)

	if len(currentPolicy.Spec.HTTPRateLimits) == 0 {
		return nil
	}

	for _, rule := range route.Spec.Rules {
		for _, m := range rule.Matches {
			for _, routePolicy := range routeRateLimits {
				routePolicy := routePolicy.(*gwpav1alpha1.RateLimitPolicy)

				if len(routePolicy.Spec.HTTPRateLimits) == 0 {
					continue
				}

				r1 := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(m, *routePolicy)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(m, *currentPolicy)
				if r2 == nil {
					continue
				}

				if reflect.DeepEqual(r1, r2) {
					continue
				}

				return &types.NamespacedName{
					Name:      routePolicy.Name,
					Namespace: routePolicy.Namespace,
				}
			}
		}
	}

	return nil
}

func (r *rateLimitPolicyReconciler) getConflictedGRPCRouteBasedRateLimitPolicy(route *gwv1alpha2.GRPCRoute, rateLimitPolicy client.Object, routeRateLimits []client.Object) *types.NamespacedName {
	currentPolicy := rateLimitPolicy.(*gwpav1alpha1.RateLimitPolicy)

	if len(currentPolicy.Spec.GRPCRateLimits) == 0 {
		return nil
	}

	for _, rule := range route.Spec.Rules {
		for _, m := range rule.Matches {
			for _, routePolicy := range routeRateLimits {
				routePolicy := routePolicy.(*gwpav1alpha1.RateLimitPolicy)

				if len(routePolicy.Spec.GRPCRateLimits) == 0 {
					continue
				}

				r1 := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(m, *routePolicy)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(m, *currentPolicy)
				if r2 == nil {
					continue
				}

				if reflect.DeepEqual(r1, r2) {
					continue
				}

				return &types.NamespacedName{
					Name:      routePolicy.Name,
					Namespace: routePolicy.Namespace,
				}
			}
		}
	}

	return nil
}

func (r *rateLimitPolicyReconciler) getConflictedPort(gateway *gwv1.Gateway, rateLimitPolicy client.Object, allRateLimitPolicies []client.Object) *types.NamespacedName {
	currentPolicy := rateLimitPolicy.(*gwpav1alpha1.RateLimitPolicy)

	if len(currentPolicy.Spec.Ports) == 0 {
		return nil
	}

	validListeners := gwutils.GetValidListenersFromGateway(gateway)
	for _, pr := range allRateLimitPolicies {
		pr := pr.(*gwpav1alpha1.RateLimitPolicy)

		if len(pr.Spec.Ports) > 0 {
			for _, listener := range validListeners {
				r1 := ratelimit.GetRateLimitIfPortMatchesPolicy(listener.Port, *pr)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfPortMatchesPolicy(listener.Port, *currentPolicy)
				if r2 == nil {
					continue
				}

				if *r1 == *r2 {
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
func (r *rateLimitPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.RateLimitPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r)
}

func (r *rateLimitPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	requests := make([]reconcile.Request, 0)
	policies := r.fctx.InformerCollection.GetGatewayResourcesFromCache(informers.RateLimitPoliciesResourceType, false)

	for _, p := range policies {
		policy := p.(*gwpav1alpha1.RateLimitPolicy)

		if gwutils.HasAccessToTargetRef(policy, policy.Spec.TargetRef, []client.Object{refGrant}) {
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
