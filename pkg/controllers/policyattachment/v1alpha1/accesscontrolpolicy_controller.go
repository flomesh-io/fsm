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

type accessControlPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
}

// NewAccessControlPolicyReconciler returns a new AccessControlPolicy Reconciler
func NewAccessControlPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &accessControlPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("AccessControlPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a AccessControlPolicy object and makes changes based on the state read
func (r *accessControlPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.AccessControlPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.AccessControlPolicy{
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

func (r *accessControlPolicyReconciler) getStatusCondition(ctx context.Context, policy *gwpav1alpha1.AccessControlPolicy) metav1.Condition {
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
		accessControlPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().AccessControlPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list AccessControlPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedPort(gateway, policy, accessControlPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with AccessControlPolicy: %s", conflict),
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
		accessControlPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().AccessControlPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list AccessControlPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedHostnamesOrRouteBasedAccessControlPolicy(httpRoute, policy, accessControlPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with AccessControlPolicy: %s", conflict),
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
		accessControlPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().AccessControlPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list AccessControlPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedHostnamesOrRouteBasedAccessControlPolicy(grpcRoute, policy, accessControlPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with AccessControlPolicy: %s", conflict),
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

func (r *accessControlPolicyReconciler) getConflictedHostnamesOrRouteBasedAccessControlPolicy(route client.Object, accessControlPolicy *gwpav1alpha1.AccessControlPolicy, allAccessControlPolicies *gwpav1alpha1.AccessControlPolicyList) *types.NamespacedName {
	hostnamesAccessControls := make([]gwpav1alpha1.AccessControlPolicy, 0)
	routeAccessControls := make([]gwpav1alpha1.AccessControlPolicy, 0)
	for _, p := range allAccessControlPolicies.Items {
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
			gwutils.IsRefToTarget(p.Spec.TargetRef, route) {
			if len(p.Spec.Hostnames) > 0 {
				hostnamesAccessControls = append(hostnamesAccessControls, p)
			}
			if len(p.Spec.HTTPAccessControls) > 0 || len(p.Spec.GRPCAccessControls) > 0 {
				routeAccessControls = append(routeAccessControls, p)
			}
		}
	}
	sort.Slice(hostnamesAccessControls, func(i, j int) bool {
		if hostnamesAccessControls[i].CreationTimestamp.Time.Equal(hostnamesAccessControls[j].CreationTimestamp.Time) {
			return hostnamesAccessControls[i].Name < hostnamesAccessControls[j].Name
		}

		return hostnamesAccessControls[i].CreationTimestamp.Time.Before(hostnamesAccessControls[j].CreationTimestamp.Time)
	})
	sort.Slice(routeAccessControls, func(i, j int) bool {
		if routeAccessControls[i].CreationTimestamp.Time.Equal(routeAccessControls[j].CreationTimestamp.Time) {
			return routeAccessControls[i].Name < routeAccessControls[j].Name
		}

		return routeAccessControls[i].CreationTimestamp.Time.Before(routeAccessControls[j].CreationTimestamp.Time)
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
		if conflict := r.getConflictedHostnamesBasedAccessControlPolicy(info, accessControlPolicy, hostnamesAccessControls); conflict != nil {
			return conflict
		}
		if conflict := r.getConflictedRouteBasedAccessControlPolicy(route, accessControlPolicy, routeAccessControls); conflict != nil {
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
		if conflict := r.getConflictedHostnamesBasedAccessControlPolicy(info, accessControlPolicy, hostnamesAccessControls); conflict != nil {
			return conflict
		}
		if conflict := r.getConflictedRouteBasedAccessControlPolicy(route, accessControlPolicy, routeAccessControls); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *accessControlPolicyReconciler) getConflictedHostnamesBasedAccessControlPolicy(route routeInfo, accessControlPolicy *gwpav1alpha1.AccessControlPolicy, hostnamesAccessControls []gwpav1alpha1.AccessControlPolicy) *types.NamespacedName {
	if len(accessControlPolicy.Spec.Hostnames) == 0 {
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
					for _, hr := range hostnamesAccessControls {
						r1 := gwutils.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, hr)
						if r1 == nil {
							continue
						}

						r2 := gwutils.GetAccessControlConfigIfRouteHostnameMatchesPolicy(hostname, *accessControlPolicy)
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

func (r *accessControlPolicyReconciler) getConflictedRouteBasedAccessControlPolicy(route client.Object, accessControlPolicy *gwpav1alpha1.AccessControlPolicy, routeAccessControls []gwpav1alpha1.AccessControlPolicy) *types.NamespacedName {
	if len(accessControlPolicy.Spec.HTTPAccessControls) == 0 &&
		len(accessControlPolicy.Spec.GRPCAccessControls) == 0 {
		return nil
	}

	switch route := route.(type) {
	case *gwv1beta1.HTTPRoute:
		for _, rule := range route.Spec.Rules {
			for _, m := range rule.Matches {
				for _, accessControl := range routeAccessControls {
					if len(accessControl.Spec.HTTPAccessControls) == 0 {
						continue
					}

					r1 := gwutils.GetAccessControlConfigIfHTTPRouteMatchesPolicy(m, accessControl)
					if r1 == nil {
						continue
					}

					r2 := gwutils.GetAccessControlConfigIfHTTPRouteMatchesPolicy(m, *accessControlPolicy)
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
	case *gwv1alpha2.GRPCRoute:
		for _, rule := range route.Spec.Rules {
			for _, m := range rule.Matches {
				for _, rr := range routeAccessControls {
					if len(rr.Spec.GRPCAccessControls) == 0 {
						continue
					}

					r1 := gwutils.GetAccessControlConfigIfGRPCRouteMatchesPolicy(m, rr)
					if r1 == nil {
						continue
					}

					r2 := gwutils.GetAccessControlConfigIfGRPCRouteMatchesPolicy(m, *accessControlPolicy)
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

func (r *accessControlPolicyReconciler) getConflictedPort(gateway *gwv1beta1.Gateway, accessControlPolicy *gwpav1alpha1.AccessControlPolicy, allAccessControlPolicies *gwpav1alpha1.AccessControlPolicyList) *types.NamespacedName {
	if len(accessControlPolicy.Spec.Ports) == 0 {
		return nil
	}

	validListeners := gwutils.GetValidListenersFromGateway(gateway)
	for _, pr := range allAccessControlPolicies.Items {
		if gwutils.IsAcceptedPolicyAttachment(pr.Status.Conditions) &&
			gwutils.IsRefToTarget(pr.Spec.TargetRef, gateway) &&
			len(pr.Spec.Ports) > 0 {
			for _, listener := range validListeners {
				r1 := gwutils.GetAccessControlConfigIfPortMatchesPolicy(listener.Port, pr)
				if r1 == nil {
					continue
				}

				r2 := gwutils.GetAccessControlConfigIfPortMatchesPolicy(listener.Port, *accessControlPolicy)
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
func (r *accessControlPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.AccessControlPolicy{}).
		Complete(r)
}
