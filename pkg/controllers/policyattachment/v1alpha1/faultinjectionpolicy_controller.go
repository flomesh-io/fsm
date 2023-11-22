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

	"github.com/flomesh-io/fsm/pkg/gateway/policy/utils/faultinjection"

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

type faultInjectionPolicyReconciler struct {
	recorder                  record.EventRecorder
	fctx                      *fctx.ControllerContext
	gatewayAPIClient          gwclient.Interface
	policyAttachmentAPIClient policyAttachmentApiClientset.Interface
}

// NewFaultInjectionPolicyReconciler returns a new FaultInjectionPolicy Reconciler
func NewFaultInjectionPolicyReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &faultInjectionPolicyReconciler{
		recorder:                  ctx.Manager.GetEventRecorderFor("FaultInjectionPolicy"),
		fctx:                      ctx,
		gatewayAPIClient:          gwclient.NewForConfigOrDie(ctx.KubeConfig),
		policyAttachmentAPIClient: policyAttachmentApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reads that state of the cluster for a FaultInjectionPolicy object and makes changes based on the state read
func (r *faultInjectionPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &gwpav1alpha1.FaultInjectionPolicy{}
	err := r.fctx.Get(ctx, req.NamespacedName, policy)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwpav1alpha1.FaultInjectionPolicy{
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

func (r *faultInjectionPolicyReconciler) getStatusCondition(ctx context.Context, policy *gwpav1alpha1.FaultInjectionPolicy) metav1.Condition {
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
		faultInjectionPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().FaultInjectionPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list FaultInjectionPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedHostnamesOrRouteBasedFaultInjectionPolicy(httpRoute, policy, faultInjectionPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with FaultInjectionPolicy: %s", conflict),
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
		faultInjectionPolicyList, err := r.policyAttachmentAPIClient.GatewayV1alpha1().FaultInjectionPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonInvalid),
				Message:            fmt.Sprintf("Failed to list FaultInjectionPolicies: %s", err),
			}
		}
		if conflict := r.getConflictedHostnamesOrRouteBasedFaultInjectionPolicy(grpcRoute, policy, faultInjectionPolicyList); conflict != nil {
			return metav1.Condition{
				Type:               string(gwv1alpha2.PolicyConditionAccepted),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: policy.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1alpha2.PolicyReasonConflicted),
				Message:            fmt.Sprintf("Conflict with FaultInjectionPolicy: %s", conflict),
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

func (r *faultInjectionPolicyReconciler) getConflictedHostnamesOrRouteBasedFaultInjectionPolicy(route client.Object, faultInjectionPolicy *gwpav1alpha1.FaultInjectionPolicy, allFaultInjectionPolicies *gwpav1alpha1.FaultInjectionPolicyList) *types.NamespacedName {
	hostnamesFaultInjections := make([]gwpav1alpha1.FaultInjectionPolicy, 0)
	routeFaultInjections := make([]gwpav1alpha1.FaultInjectionPolicy, 0)
	for _, p := range allFaultInjectionPolicies.Items {
		if gwutils.IsAcceptedPolicyAttachment(p.Status.Conditions) &&
			gwutils.IsRefToTarget(p.Spec.TargetRef, route) {
			if len(p.Spec.Hostnames) > 0 {
				hostnamesFaultInjections = append(hostnamesFaultInjections, p)
			}
			if len(p.Spec.HTTPFaultInjections) > 0 || len(p.Spec.GRPCFaultInjections) > 0 {
				routeFaultInjections = append(routeFaultInjections, p)
			}
		}
	}
	sort.Slice(hostnamesFaultInjections, func(i, j int) bool {
		if hostnamesFaultInjections[i].CreationTimestamp.Time.Equal(hostnamesFaultInjections[j].CreationTimestamp.Time) {
			return client.ObjectKeyFromObject(&hostnamesFaultInjections[i]).String() < client.ObjectKeyFromObject(&hostnamesFaultInjections[j]).String()
		}

		return hostnamesFaultInjections[i].CreationTimestamp.Time.Before(hostnamesFaultInjections[j].CreationTimestamp.Time)
	})
	sort.Slice(routeFaultInjections, func(i, j int) bool {
		if routeFaultInjections[i].CreationTimestamp.Time.Equal(routeFaultInjections[j].CreationTimestamp.Time) {
			return client.ObjectKeyFromObject(&routeFaultInjections[i]).String() < client.ObjectKeyFromObject(&routeFaultInjections[j]).String()
		}

		return routeFaultInjections[i].CreationTimestamp.Time.Before(routeFaultInjections[j].CreationTimestamp.Time)
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
		if conflict := r.getConflictedHostnamesBasedFaultInjectionPolicy(info, faultInjectionPolicy, hostnamesFaultInjections); conflict != nil {
			return conflict
		}
		if conflict := r.getConflictedRouteBasedFaultInjectionPolicy(route, faultInjectionPolicy, routeFaultInjections); conflict != nil {
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
		if conflict := r.getConflictedHostnamesBasedFaultInjectionPolicy(info, faultInjectionPolicy, hostnamesFaultInjections); conflict != nil {
			return conflict
		}
		if conflict := r.getConflictedRouteBasedFaultInjectionPolicy(route, faultInjectionPolicy, routeFaultInjections); conflict != nil {
			return conflict
		}
	}

	return nil
}

func (r *faultInjectionPolicyReconciler) getConflictedHostnamesBasedFaultInjectionPolicy(route routeInfo, faultInjectionPolicy *gwpav1alpha1.FaultInjectionPolicy, hostnamesFaultInjections []gwpav1alpha1.FaultInjectionPolicy) *types.NamespacedName {
	if len(faultInjectionPolicy.Spec.Hostnames) == 0 {
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
					for _, hr := range hostnamesFaultInjections {
						r1 := faultinjection.GetFaultInjectionConfigIfRouteHostnameMatchesPolicy(hostname, hr)
						if r1 == nil {
							continue
						}

						r2 := faultinjection.GetFaultInjectionConfigIfRouteHostnameMatchesPolicy(hostname, *faultInjectionPolicy)
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

func (r *faultInjectionPolicyReconciler) getConflictedRouteBasedFaultInjectionPolicy(route client.Object, faultInjectionPolicy *gwpav1alpha1.FaultInjectionPolicy, routeFaultInjections []gwpav1alpha1.FaultInjectionPolicy) *types.NamespacedName {
	if len(faultInjectionPolicy.Spec.HTTPFaultInjections) == 0 &&
		len(faultInjectionPolicy.Spec.GRPCFaultInjections) == 0 {
		return nil
	}

	switch route := route.(type) {
	case *gwv1beta1.HTTPRoute:
		for _, rule := range route.Spec.Rules {
			for _, m := range rule.Matches {
				for _, faultInjection := range routeFaultInjections {
					if len(faultInjection.Spec.HTTPFaultInjections) == 0 {
						continue
					}

					r1 := faultinjection.GetFaultInjectionConfigIfHTTPRouteMatchesPolicy(m, faultInjection)
					if r1 == nil {
						continue
					}

					r2 := faultinjection.GetFaultInjectionConfigIfHTTPRouteMatchesPolicy(m, *faultInjectionPolicy)
					if r2 == nil {
						continue
					}

					if reflect.DeepEqual(r1, r2) {
						continue
					}

					return &types.NamespacedName{
						Name:      faultInjection.Name,
						Namespace: faultInjection.Namespace,
					}
				}
			}
		}
	case *gwv1alpha2.GRPCRoute:
		for _, rule := range route.Spec.Rules {
			for _, m := range rule.Matches {
				for _, rr := range routeFaultInjections {
					if len(rr.Spec.GRPCFaultInjections) == 0 {
						continue
					}

					r1 := faultinjection.GetFaultInjectionConfigIfGRPCRouteMatchesPolicy(m, rr)
					if r1 == nil {
						continue
					}

					r2 := faultinjection.GetFaultInjectionConfigIfGRPCRouteMatchesPolicy(m, *faultInjectionPolicy)
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

// SetupWithManager sets up the controller with the Manager.
func (r *faultInjectionPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.FaultInjectionPolicy{}).
		Complete(r)
}
