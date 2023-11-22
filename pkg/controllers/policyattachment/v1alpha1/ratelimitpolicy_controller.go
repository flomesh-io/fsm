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
	"sort"
	"time"

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/ratelimit"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	metautil "k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/apimachinery/pkg/types"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

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
}

// NewRateLimitPolicyReconciler returns a new RateLimitPolicy Reconciler
func NewRateLimitPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &rateLimitPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("RateLimitPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a RateLimitPolicy object and makes changes based on the state read
func (r *rateLimitPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.RateLimitPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.RateLimitPolicy{
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

	metautil.SetStatusCondition(&policy.Status.Conditions, r.getStatusCondition(ctx, policy))
	if err := r.fctx.Status().Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.EventHandler.OnAdd(policy)

	return ctrl.Result{}, nil
}

func (r *rateLimitPolicyReconciler) getStatusCondition(ctx context.Context, policy *gwpav1alpha1.RateLimitPolicy) metav1.Condition {
	if policy.Spec.TargetRef.Group != constants.GatewayAPIGroup {
		return metav1.Condition{
			Type:               string(gwv1alpha2.PolicyConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: policy.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1alpha2.PolicyReasonInvalid),
			Message:            "Invalid target reference group, only gateway.networking.k8s.io is supported",
		}
	}

	switch policy.Spec.TargetRef.Kind {
	case constants.GatewayAPIGatewayKind:
		gateway := &gwv1beta1.Gateway{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy, policy.Spec.TargetRef), Name: string(policy.Spec.TargetRef.Name)}, gateway); err != nil {
			if errors.IsNotFound(err) {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonTargetNotFound),
					Message:            "Invalid target reference, cannot find target Gateway",
				}
			} else {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonInvalid),
					Message:            fmt.Sprintf("Failed to get target Gateway: %s", err),
				}
			}
		}
		rateLimitPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list RateLimitPolicies: %s", err),
			}
		}

		rateLimitPolicies := make([]gwpav1alpha1.RateLimitPolicy, 0)
		for _, p := range rateLimitPolicyList.Items {
			if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
				gwutils.IsRefToTarget(p.Spec.TargetRef, gateway) {
				rateLimitPolicies = append(rateLimitPolicies, p)
			}
		}

		sort.Slice(rateLimitPolicies, func(i, j int) bool {
			if rateLimitPolicies[i].CreationTimestamp.Time.Equal(rateLimitPolicies[j].CreationTimestamp.Time) {
				return client.ObjectKeyFromObject(&rateLimitPolicies[i]).String() < client.ObjectKeyFromObject(&rateLimitPolicies[j]).String()
			}

			return rateLimitPolicies[i].CreationTimestamp.Time.Before(rateLimitPolicies[j].CreationTimestamp.Time)
		})

		if conflict := r.getConflictedPort(gateway, policy, rateLimitPolicies); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with RateLimitPolicy: %s", conflict),
			}
		}

	case constants.GatewayAPIHTTPRouteKind:
		httpRoute := &gwv1beta1.HTTPRoute{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy, policy.Spec.TargetRef), Name: string(policy.Spec.TargetRef.Name)}, httpRoute); err != nil {
			if errors.IsNotFound(err) {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonTargetNotFound),
					Message:            "Invalid target reference, cannot find target HTTPRoute",
				}
			} else {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonInvalid),
					Message:            fmt.Sprintf("Failed to get target HTTPRoute: %s", err),
				}
			}
		}
		rateLimitPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list RateLimitPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedHostnamesOrRouteBasedRateLimitPolicy(httpRoute, policy, rateLimitPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with RateLimitPolicy: %s", conflict),
			}
		}
	case constants.GatewayAPIGRPCRouteKind:
		grpcRoute := &gwv1alpha2.GRPCRoute{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy, policy.Spec.TargetRef), Name: string(policy.Spec.TargetRef.Name)}, grpcRoute); err != nil {
			if errors.IsNotFound(err) {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonTargetNotFound),
					Message:            "Invalid target reference, cannot find target GRPCRoute",
				}
			} else {
				return metav1.Condition{
					Type:               string(gwv1alpha2.PolicyConditionAccepted),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: policy.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1alpha2.PolicyReasonInvalid),
					Message:            fmt.Sprintf("Failed to get target GRPCRoute: %s", err),
				}
			}
		}
		rateLimitPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list RateLimitPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedHostnamesOrRouteBasedRateLimitPolicy(grpcRoute, policy, rateLimitPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with RateLimitPolicy: %s", conflict),
			}
		}
	default:
		return metav1.Condition{
			Type:               string(gwv1alpha2.PolicyConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: policy.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1alpha2.PolicyReasonInvalid),
			Message:            "Invalid target reference kind, only Gateway, HTTPRoute and GRCPRoute are supported",
		}
	}

	return metav1.Condition{
		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: policy.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1alpha2.PolicyReasonAccepted),
		Message:            string(gwv1alpha2.PolicyReasonAccepted),
	}
}

func (r *rateLimitPolicyReconciler) getConflictedHostnamesOrRouteBasedRateLimitPolicy(route client.Object, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy, allRateLimitPolicies *gwpav1alpha1.RateLimitPolicyList) *types.NamespacedName {
	hostnamesRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	routeRateLimits := make([]gwpav1alpha1.RateLimitPolicy, 0)
	for _, p := range allRateLimitPolicies.Items {
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
			gwutils.IsRefToTarget(p.Spec.TargetRef, route) {
			if len(p.Spec.Hostnames) > 0 {
				hostnamesRateLimits = append(hostnamesRateLimits, p)
			}
			if len(p.Spec.HTTPRateLimits) > 0 || len(p.Spec.GRPCRateLimits) > 0 {
				routeRateLimits = append(routeRateLimits, p)
			}
		}
	}
	sort.Slice(hostnamesRateLimits, func(i, j int) bool {
		if hostnamesRateLimits[i].CreationTimestamp.Time.Equal(hostnamesRateLimits[j].CreationTimestamp.Time) {
			return client.ObjectKeyFromObject(&hostnamesRateLimits[i]).String() < client.ObjectKeyFromObject(&hostnamesRateLimits[j]).String()
		}

		return hostnamesRateLimits[i].CreationTimestamp.Time.Before(hostnamesRateLimits[j].CreationTimestamp.Time)
	})
	sort.Slice(routeRateLimits, func(i, j int) bool {
		if routeRateLimits[i].CreationTimestamp.Time.Equal(routeRateLimits[j].CreationTimestamp.Time) {
			return client.ObjectKeyFromObject(&routeRateLimits[i]).String() < client.ObjectKeyFromObject(&routeRateLimits[j]).String()
		}

		return routeRateLimits[i].CreationTimestamp.Time.Before(routeRateLimits[j].CreationTimestamp.Time)
	})

	switch route := route.(type) {
	case *gwv1beta1.HTTPRoute:
		info := routeInfo{
			meta:       route,
			parents:    route.Status.Parents,
			gvk:        route.GroupVersionKind(),
			generation: route.Generation,
			hostnames:  route.Spec.Hostnames,
		}
		if conflict := r.getConflictedHostnamesBasedRateLimitPolicy(info, rateLimitPolicy, hostnamesRateLimits); conflict != nil {
			return conflict
		}
		if conflict := r.getConflictedRouteBasedRateLimitPolicy(route, rateLimitPolicy, routeRateLimits); conflict != nil {
			return conflict
		}

	case *gwv1alpha2.GRPCRoute:
		info := routeInfo{
			meta:       route,
			parents:    route.Status.Parents,
			gvk:        route.GroupVersionKind(),
			generation: route.Generation,
			hostnames:  route.Spec.Hostnames,
		}
		if conflict := r.getConflictedHostnamesBasedRateLimitPolicy(info, rateLimitPolicy, hostnamesRateLimits); conflict != nil {
			return conflict
		}
		if conflict := r.getConflictedRouteBasedRateLimitPolicy(route, rateLimitPolicy, routeRateLimits); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *rateLimitPolicyReconciler) getConflictedHostnamesBasedRateLimitPolicy(route routeInfo, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy, hostnamesRateLimits []gwpav1alpha1.RateLimitPolicy) *types.NamespacedName {
	if len(rateLimitPolicy.Spec.Hostnames) == 0 {
		return nil
	}

	for _, parent := range route.parents {
		if metautil.IsStatusConditionTrue(parent.Conditions, string(gwv1beta1.RouteConditionAccepted)) {
			key := getRouteParentKey(route.meta, parent)

			gateway := &gwv1beta1.Gateway{}
			if err := r.fctx.Get(context.TODO(), key, gateway); err != nil {
				continue
			}

			validListeners := gwutils.GetValidListenersFromGateway(gateway)

			allowedListeners := gwutils.GetAllowedListeners(parent.ParentRef, route.gvk, route.generation, validListeners)
			for _, listener := range allowedListeners {
				hostnames := gwutils.GetValidHostnames(listener.Hostname, route.hostnames)
				if len(hostnames) == 0 {
					// no valid hostnames, should ignore it
					continue
				}
				for _, hostname := range hostnames {
					for _, hr := range hostnamesRateLimits {
						r1 := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, hr)
						if r1 == nil {
							continue
						}

						r2 := ratelimit.GetRateLimitIfRouteHostnameMatchesPolicy(hostname, *rateLimitPolicy)
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

func (r *rateLimitPolicyReconciler) getConflictedRouteBasedRateLimitPolicy(route client.Object, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy, routeRateLimits []gwpav1alpha1.RateLimitPolicy) *types.NamespacedName {
	if len(rateLimitPolicy.Spec.HTTPRateLimits) == 0 &&
		len(rateLimitPolicy.Spec.GRPCRateLimits) == 0 {
		return nil
	}

	switch route := route.(type) {
	case *gwv1beta1.HTTPRoute:
		for _, rule := range route.Spec.Rules {
			for _, m := range rule.Matches {
				for _, rateLimit := range routeRateLimits {
					if len(rateLimit.Spec.HTTPRateLimits) == 0 {
						continue
					}

					r1 := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(m, rateLimit)
					if r1 == nil {
						continue
					}

					r2 := ratelimit.GetRateLimitIfHTTPRouteMatchesPolicy(m, *rateLimitPolicy)
					if r2 == nil {
						continue
					}

					if reflect.DeepEqual(r1, r2) {
						continue
					}

					return &types.NamespacedName{
						Name:      rateLimit.Name,
						Namespace: rateLimit.Namespace,
					}
				}
			}
		}
	case *gwv1alpha2.GRPCRoute:
		for _, rule := range route.Spec.Rules {
			for _, m := range rule.Matches {
				for _, rr := range routeRateLimits {
					if len(rr.Spec.GRPCRateLimits) == 0 {
						continue
					}

					r1 := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(m, rr)
					if r1 == nil {
						continue
					}

					r2 := ratelimit.GetRateLimitIfGRPCRouteMatchesPolicy(m, *rateLimitPolicy)
					if r2 == nil {
						continue
					}

					if reflect.DeepEqual(r1, r2) {
						continue
					}

					return &types.NamespacedName{
						Name:      rr.Name,
						Namespace: rr.Namespace,
					}
				}
			}
		}
	}

	return nil
}

func (r *rateLimitPolicyReconciler) getConflictedPort(gateway *gwv1beta1.Gateway, rateLimitPolicy *gwpav1alpha1.RateLimitPolicy, allRateLimitPolicies []gwpav1alpha1.RateLimitPolicy) *types.NamespacedName {
	if len(rateLimitPolicy.Spec.Ports) == 0 {
		return nil
	}

	validListeners := gwutils.GetValidListenersFromGateway(gateway)
	for _, pr := range allRateLimitPolicies {
		if len(pr.Spec.Ports) > 0 {
			for _, listener := range validListeners {
				r1 := ratelimit.GetRateLimitIfPortMatchesPolicy(listener.Port, pr)
				if r1 == nil {
					continue
				}

				r2 := ratelimit.GetRateLimitIfPortMatchesPolicy(listener.Port, *rateLimitPolicy)
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
		Complete(r)
}
