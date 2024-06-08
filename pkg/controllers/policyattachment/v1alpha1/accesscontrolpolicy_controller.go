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

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
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

type accessControlPolicyReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *policystatus.PolicyStatusProcessor
}

func (r *accessControlPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewAccessControlPolicyReconciler returns a new AccessControlPolicy Reconciler
func NewAccessControlPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &accessControlPolicyReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("AccessControlPolicy"),
		fctx:     ctx,
	}

	r.statusProcessor = &policystatus.PolicyStatusProcessor{
		Client:                             r.fctx.Client,
		Informer:                           r.fctx.InformerCollection,
		GetPolicies:                        r.getAccessControls,
		FindConflictPort:                   r.getConflictedPort,
		FindConflictedHostnamesBasedPolicy: r.getConflictedHostnamesBasedAccessControlPolicy,
		FindConflictedHTTPRouteBasedPolicy: r.getConflictedHTTPRouteBasedAccessControlPolicy,
		FindConflictedGRPCRouteBasedPolicy: r.getConflictedGRPCRouteBasedAccessControlPolicy,
	}

	return r
}

// Reconcile reads that state of the cluster for a AccessControlPolicy object and makes changes based on the state read
func (r *accessControlPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.AccessControlPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwpav1alpha1.AccessControlPolicy{
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

func (r *accessControlPolicyReconciler) getAccessControls(target client.Object) map[gwpkg.PolicyMatchType][]client.Object {
	c := r.fctx.Manager.GetCache()
	policies := make(map[gwpkg.PolicyMatchType][]client.Object)

	for _, param := range []struct {
		matchType gwpkg.PolicyMatchType
		fn        func(cache.Cache, fields.Selector) []client.Object
		selector  fields.Selector
	}{
		{
			matchType: gwpkg.PolicyMatchTypePort,
			fn:        gwutils.GetAccessControlsMatchTypePort,
			selector:  fields.OneTermEqualSelector(constants.PortPolicyAttachmentIndex, client.ObjectKeyFromObject(target).String()),
		},
		{
			matchType: gwpkg.PolicyMatchTypeHostnames,
			fn:        gwutils.GetAccessControlsMatchTypeHostname,
			selector:  fields.OneTermEqualSelector(constants.HostnamePolicyAttachmentIndex, fmt.Sprintf("%s/%s/%s", target.GetObjectKind().GroupVersionKind().Kind, target.GetNamespace(), target.GetName())),
		},
		{
			matchType: gwpkg.PolicyMatchTypeHTTPRoute,
			fn:        gwutils.GetAccessControlsMatchTypeHTTPRoute,
			selector:  fields.OneTermEqualSelector(constants.HTTPRoutePolicyAttachmentIndex, client.ObjectKeyFromObject(target).String()),
		},
		{
			matchType: gwpkg.PolicyMatchTypeGRPCRoute,
			fn:        gwutils.GetAccessControlsMatchTypeGRPCRoute,
			selector:  fields.OneTermEqualSelector(constants.GRPCRoutePolicyAttachmentIndex, client.ObjectKeyFromObject(target).String()),
		},
	} {
		if result := param.fn(c, param.selector); len(result) > 0 {
			policies[param.matchType] = result
		}
	}

	return policies
}

func (r *accessControlPolicyReconciler) getConflictedHostnamesBasedAccessControlPolicy(route status.RouteStatusObject, parentRefs []gwv1.ParentReference, accessControlPolicy client.Object, hostnamesAccessControls []client.Object) *types.NamespacedName {
	currentPolicy := accessControlPolicy.(*gwpav1alpha1.AccessControlPolicy)

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
					for _, hr := range hostnamesAccessControls {
						hr := hr.(*gwpav1alpha1.AccessControlPolicy)

						r1 := accesscontrol.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, hr)
						if r1 == nil {
							continue
						}

						r2 := accesscontrol.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, currentPolicy)
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

func (r *accessControlPolicyReconciler) getConflictedHTTPRouteBasedAccessControlPolicy(route *gwv1.HTTPRoute, accessControlPolicy client.Object, routeAccessControls []client.Object) *types.NamespacedName {
	currentPolicy := accessControlPolicy.(*gwpav1alpha1.AccessControlPolicy)

	if len(currentPolicy.Spec.HTTPAccessControls) == 0 {
		return nil
	}

	for _, rule := range route.Spec.Rules {
		for _, m := range rule.Matches {
			for _, routePolicy := range routeAccessControls {
				routePolicy := routePolicy.(*gwpav1alpha1.AccessControlPolicy)

				if len(routePolicy.Spec.HTTPAccessControls) == 0 {
					continue
				}

				r1 := accesscontrol.GetAccessControlConfigIfHTTPRouteMatchesPolicy(m, routePolicy)
				if r1 == nil {
					continue
				}

				r2 := accesscontrol.GetAccessControlConfigIfHTTPRouteMatchesPolicy(m, currentPolicy)
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

func (r *accessControlPolicyReconciler) getConflictedGRPCRouteBasedAccessControlPolicy(route *gwv1.GRPCRoute, accessControlPolicy client.Object, routeAccessControls []client.Object) *types.NamespacedName {
	currentPolicy := accessControlPolicy.(*gwpav1alpha1.AccessControlPolicy)

	if len(currentPolicy.Spec.GRPCAccessControls) == 0 {
		return nil
	}

	for _, rule := range route.Spec.Rules {
		for _, m := range rule.Matches {
			for _, routePolicy := range routeAccessControls {
				routePolicy := routePolicy.(*gwpav1alpha1.AccessControlPolicy)

				if len(routePolicy.Spec.GRPCAccessControls) == 0 {
					continue
				}

				r1 := accesscontrol.GetAccessControlConfigIfGRPCRouteMatchesPolicy(m, routePolicy)
				if r1 == nil {
					continue
				}

				r2 := accesscontrol.GetAccessControlConfigIfGRPCRouteMatchesPolicy(m, currentPolicy)
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

func (r *accessControlPolicyReconciler) getConflictedPort(gateway *gwv1.Gateway, accessControlPolicy client.Object, allAccessControls []client.Object) *types.NamespacedName {
	currentPolicy := accessControlPolicy.(*gwpav1alpha1.AccessControlPolicy)

	if len(currentPolicy.Spec.Ports) == 0 {
		return nil
	}

	validListeners := gwutils.GetValidListenersForGateway(gateway)
	for _, accessControl := range allAccessControls {
		accessControl := accessControl.(*gwpav1alpha1.AccessControlPolicy)

		if len(accessControl.Spec.Ports) > 0 {
			for _, listener := range validListeners {
				r1 := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(listener.Port, accessControl)
				if r1 == nil {
					continue
				}

				r2 := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(listener.Port, currentPolicy)
				if r2 == nil {
					continue
				}

				if reflect.DeepEqual(r1, r2) {
					continue
				}

				return &types.NamespacedName{
					Name:      accessControl.Name,
					Namespace: accessControl.Namespace,
				}
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *accessControlPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.AccessControlPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r); err != nil {
		return err
	}

	return addAccessControlPolicyIndexer(context.Background(), mgr)
}

func addAccessControlPolicyIndexer(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.AccessControlPolicy{}, constants.PortPolicyAttachmentIndex, addAccessControlPortIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.AccessControlPolicy{}, constants.HostnamePolicyAttachmentIndex, addAccessControlHostnameIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.AccessControlPolicy{}, constants.HTTPRoutePolicyAttachmentIndex, addAccessControlHTTPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwpav1alpha1.AccessControlPolicy{}, constants.GRPCRoutePolicyAttachmentIndex, addAccessControlGRPCRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func addAccessControlPortIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.AccessControlPolicy)
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

func addAccessControlHostnameIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.AccessControlPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if (gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) || gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK)) && len(policy.Spec.Hostnames) > 0 {
		targets = append(targets, fmt.Sprintf("%s/%s/%s", targetRef.Kind, gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace), string(targetRef.Name)))
	}

	return targets
}

func addAccessControlHTTPRouteIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.AccessControlPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if gwutils.IsTargetRefToGVK(targetRef, constants.HTTPRouteGVK) && len(policy.Spec.HTTPAccessControls) > 0 {
		targets = append(targets, types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
			Name:      string(targetRef.Name),
		}.String())
	}

	return targets
}

func addAccessControlGRPCRouteIndexFunc(obj client.Object) []string {
	policy := obj.(*gwpav1alpha1.AccessControlPolicy)
	targetRef := policy.Spec.TargetRef

	var targets []string
	if gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) && len(policy.Spec.GRPCAccessControls) > 0 {
		targets = append(targets, types.NamespacedName{
			Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.Namespace),
			Name:      string(targetRef.Name),
		}.String())
	}

	return targets
}

func (r *accessControlPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	list := &gwpav1alpha1.AccessControlPolicyList{}
	if err := c.List(context.Background(), list); err != nil {
		log.Error().Msgf("Failed to list AccessControlPolicies: %v", err)
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
