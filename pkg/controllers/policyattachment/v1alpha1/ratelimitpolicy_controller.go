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

	policystatus "github.com/flomesh-io/fsm/pkg/gateway/status/policy"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type rateLimitPolicyReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *policystatus.PolicyStatusProcessor
}

func (r *rateLimitPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewRateLimitPolicyReconciler returns a new RateLimitPolicy Reconciler
func NewRateLimitPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &rateLimitPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("RateLimitPolicy"),
		fctx:     ctx,
	}

	r.statusProcessor = &policystatus.PolicyStatusProcessor{
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

func (r *rateLimitPolicyReconciler) getRateLimitPolices(target client.Object) map[gwpkg.PolicyMatchType][]client.Object {
	c := r.fctx.Manager.GetCache()
	policies := make(map[gwpkg.PolicyMatchType][]client.Object)

	for _, p := range []struct {
		matchType gwpkg.PolicyMatchType
		fn        func(cache.Cache, fields.Selector) []client.Object
		selector  fields.Selector
	}{
		{
			matchType: gwpkg.PolicyMatchTypePort,
			fn:        gwutils.GetRateLimitsMatchTypePort,
			selector:  fields.OneTermEqualSelector(constants.PortPolicyAttachmentIndex, client.ObjectKeyFromObject(target).String()),
		},
		{
			matchType: gwpkg.PolicyMatchTypeHostnames,
			fn:        gwutils.GetRateLimitsMatchTypeHostname,
			selector:  fields.OneTermEqualSelector(constants.HostnamePolicyAttachmentIndex, fmt.Sprintf("%s/%s/%s", target.GetObjectKind().GroupVersionKind().Kind, target.GetNamespace(), target.GetName())),
		},
		{
			matchType: gwpkg.PolicyMatchTypeHTTPRoute,
			fn:        gwutils.GetRateLimitsMatchTypeHTTPRoute,
			selector:  fields.OneTermEqualSelector(constants.HTTPRoutePolicyAttachmentIndex, client.ObjectKeyFromObject(target).String()),
		},
		{
			matchType: gwpkg.PolicyMatchTypeGRPCRoute,
			fn:        gwutils.GetRateLimitsMatchTypeGRPCRoute,
			selector:  fields.OneTermEqualSelector(constants.GRPCRoutePolicyAttachmentIndex, client.ObjectKeyFromObject(target).String()),
		},
	} {
		if result := p.fn(c, p.selector); len(result) > 0 {
			policies[p.matchType] = result
		}
	}

	return policies
}

func (r *rateLimitPolicyReconciler) getConflictedHostnamesBasedRateLimitPolicy(route status.RouteStatusObject, parentRefs []gwv1.ParentReference, rateLimitPolicy client.Object, hostnamesRateLimits []client.Object) *types.NamespacedName {
	currentPolicy := rateLimitPolicy.(*gwpav1alpha1.RateLimitPolicy)

	if len(currentPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for _, parentRef := range parentRefs {
		h := route.StatusUpdateFor(parentRef)

		if metautil.IsStatusConditionTrue(h.ConditionsForParentRef(parentRef), string(gwv1.RouteConditionAccepted)) {
			key := getRouteParentKey(route.GetObjectMeta(), parentRef)

			gateway := &gwv1.Gateway{}
			if err := r.fctx.Get(context.TODO(), key, gateway); err != nil {
				continue
			}

			allowedListeners := gwutils.GetAllowedListeners(r.fctx.Manager.GetCache(), gateway, h)

			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, route.GetHostnames())
				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}
				for _, hostname := range hostnames {
					for _, hr := range hostnamesRateLimits {
						hr := hr.(*gwpav1alpha1.RateLimitPolicy)

						r1 := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, hr)
						if r1 == nil {
							continue
						}

						r2 := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, currentPolicy)
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

				r1 := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(m, routePolicy)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(m, currentPolicy)
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

func (r *rateLimitPolicyReconciler) getConflictedGRPCRouteBasedRateLimitPolicy(route *gwv1.GRPCRoute, rateLimitPolicy client.Object, routeRateLimits []client.Object) *types.NamespacedName {
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

				r1 := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(m, routePolicy)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(m, currentPolicy)
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

	validListeners := gwutils.GetValidListenersForGateway(gateway)
	for _, pr := range allRateLimitPolicies {
		pr := pr.(*gwpav1alpha1.RateLimitPolicy)

		if len(pr.Spec.Ports) > 0 {
			for _, listener := range validListeners {
				r1 := ratelimit.GetRateLimitIfPortMatchesPolicy(listener.Port, pr)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfPortMatchesPolicy(listener.Port, currentPolicy)
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
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.RateLimitPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r); err != nil {
		return err
	}

	return addRateLimitPolicyIndexer(context.Background(), mgr)
}

func addRateLimitPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.RateLimitPolicy{}, constants.PortPolicyAttachmentIndex, addRateLimitPortIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.RateLimitPolicy{}, constants.HostnamePolicyAttachmentIndex, addRateLimitHostnameIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.RateLimitPolicy{}, constants.HTTPRoutePolicyAttachmentIndex, addRateLimitHTTPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.RateLimitPolicy{}, constants.GRPCRoutePolicyAttachmentIndex, addRateLimitGRPCRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func addRateLimitPortIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.RateLimitPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if gwutils.IsTargetRefToGVK(targetRef, constants.GatewayGVK) && len(policy.Spec.Ports) > 0 {
		targets = append(targets, types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
			Name:      string(targetRef.Name),
		}.String())
	}

	return targets
}

func addRateLimitHostnameIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.RateLimitPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if (gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK)) && len(policy.Spec.Hostnames) > 0 {
		targets = append(targets, fmt.Sprintf("%s/%s/%s", targetRef.Kind, gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace), string(targetRef.Name)))
	}

	return targets
}

func addRateLimitHTTPRouteIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.RateLimitPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) && len(policy.Spec.HTTPRateLimits) > 0 {
		targets = append(targets, types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
			Name:      string(targetRef.Name),
		}.String())
	}

	return targets
}

func addRateLimitGRPCRouteIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.RateLimitPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) && len(policy.Spec.GRPCRateLimits) > 0 {
		targets = append(targets, types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
			Name:      string(targetRef.Name),
		}.String())
	}

	return targets
}

func (r *rateLimitPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	list := &gwpav1alpha1.RateLimitPolicyList{}
	if err := c.List(context.Background(), list); err != nil {
		log.Error().Msgf("Failed to list RateLimitPolicyList: %v", err)
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
