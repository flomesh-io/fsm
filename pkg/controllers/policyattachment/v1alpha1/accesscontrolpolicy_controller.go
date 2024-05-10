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

	gwpkg "github.com/flomesh-io/fsm/pkg/gateway/types"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/status"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/accesscontrol"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

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

type accessControlPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
	statusProcessor           *status.PolicyStatusProcessor
}

func (r *accessControlPolicyReconciler) NeedLeaderElection() bool {
	return true
}

// NewAccessControlPolicyReconciler returns a new AccessControlPolicy Reconciler
func NewAccessControlPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	r := &accessControlPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("AccessControlPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}

	r.statusProcessor = &status.PolicyStatusProcessor{
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

func (r *accessControlPolicyReconciler) getAccessControls(policy client.Object, target client.Object) (map[gwpkg.PolicyMatchType][]client.Object, *metav1.Condition) {
	accessControlPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().AccessControlPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, status.ConditionPointer(status.InvalidCondition(policy, fmt.Sprintf("Failed to list AccessControlPolicies: %s", err)))
	}

	policies := make(map[gwpkg.PolicyMatchType][]client.Object)
	referenceGrants := r.fctx.InformerCollection.GetGatewayResourcesFromCache(informers.ReferenceGrantResourceType, false)

	for _, p := range accessControlPolicyList.Items {
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
				len(spec.HTTPAccessControls) > 0:
				policies[gwpkg.PolicyMatchTypeHTTPRoute] = append(policies[gwpkg.PolicyMatchTypeHTTPRoute], &p)
			case gwutils.IsTargetRefToGVK(targetRef, constants.GRPCRouteGVK) &&
				gwutils.IsRefToTarget(referenceGrants, &p, targetRef, target) &&
				len(spec.GRPCAccessControls) > 0:
				policies[gwpkg.PolicyMatchTypeGRPCRoute] = append(policies[gwpkg.PolicyMatchTypeGRPCRoute], &p)
			}
		}
	}

	return policies, nil
}

func (r *accessControlPolicyReconciler) getConflictedHostnamesBasedAccessControlPolicy(route *gwtypes.RouteContext, accessControlPolicy client.Object, hostnamesAccessControls []client.Object) *types.NamespacedName {
	currentPolicy := accessControlPolicy.(*gwpav1alpha1.AccessControlPolicy)

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

			validListeners := gwutils.GetValidListenersForGateway(gateway)

			allowedListeners, _ := gwutils.GetAllowedListeners(r.fctx.InformerCollection.GetListers().Namespace, gateway, parent.ParentRef, route, validListeners)
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, route.Hostnames)
				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}
				for _, hostname := range hostnames {
					for _, hr := range hostnamesAccessControls {
						hr := hr.(*gwpav1alpha1.AccessControlPolicy)

						r1 := accesscontrol.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, *hr)
						if r1 == nil {
							continue
						}

						r2 := accesscontrol.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, *currentPolicy)
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

				r1 := accesscontrol.GetAccessControlConfigIfHTTPRouteMatchesPolicy(m, *routePolicy)
				if r1 == nil {
					continue
				}

				r2 := accesscontrol.GetAccessControlConfigIfHTTPRouteMatchesPolicy(m, *currentPolicy)
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

				r1 := accesscontrol.GetAccessControlConfigIfGRPCRouteMatchesPolicy(m, *routePolicy)
				if r1 == nil {
					continue
				}

				r2 := accesscontrol.GetAccessControlConfigIfGRPCRouteMatchesPolicy(m, *currentPolicy)
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
				r1 := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(listener.Port, *accessControl)
				if r1 == nil {
					continue
				}

				r2 := accesscontrol.GetAccessControlConfigIfPortMatchesPolicy(listener.Port, *currentPolicy)
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.AccessControlPolicy{}).
		Watches(
			&gwv1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.referenceGrantToPolicyAttachment),
		).
		Complete(r)
}

func (r *accessControlPolicyReconciler) referenceGrantToPolicyAttachment(_ context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	requests := make([]reconcile.Request, 0)
	policies := r.fctx.InformerCollection.GetGatewayResourcesFromCache(informers.AccessControlPoliciesResourceType, false)

	for _, p := range policies {
		policy := p.(*gwpav1alpha1.AccessControlPolicy)

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
