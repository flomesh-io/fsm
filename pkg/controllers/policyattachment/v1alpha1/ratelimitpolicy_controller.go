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

package v1beta1

import (
	"context"
	"fmt"
	"time"

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

// TODO: handle conflict
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
	case constants.GatewayKind:
		gateway := &gwv1beta1.Gateway{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy), Name: string(policy.Spec.TargetRef.Name)}, gateway); err != nil {
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
		//polices, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		//if err != nil {
		//	return metav1.Condition{
		//		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		//		Status:             metav1.ConditionFalse,
		//		ObservedGeneration: policy.Generation,
		//		LastTransitionTime: metav1.Time{Time: time.Now()},
		//		Reason:             string(gwv1alpha2.PolicyReasonInvalid),
		//		Message:            fmt.Sprintf("Failed to list RateLimitPolicies: %s", err),
		//	}
		//}
	case constants.HTTPRouteKind:
		httpRoute := &gwv1beta1.HTTPRoute{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy), Name: string(policy.Spec.TargetRef.Name)}, httpRoute); err != nil {
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
		//polices, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		//if err != nil {
		//	return metav1.Condition{
		//		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		//		Status:             metav1.ConditionFalse,
		//		ObservedGeneration: policy.Generation,
		//		LastTransitionTime: metav1.Time{Time: time.Now()},
		//		Reason:             string(gwv1alpha2.PolicyReasonInvalid),
		//		Message:            fmt.Sprintf("Failed to list RateLimitPolicies: %s", err),
		//	}
		//}
		//for _, p := range polices.Items {
		//	if gwutils.IsAcceptedRateLimitPolicy(&p) {
		//
		//	}
		//}
	case constants.GRPCRouteKind:
		grpcRoute := &gwv1alpha2.GRPCRoute{}
		if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: getTargetNamespace(policy), Name: string(policy.Spec.TargetRef.Name)}, grpcRoute); err != nil {
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
		//polices, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		//if err != nil {
		//	return metav1.Condition{
		//		Type:               string(gwv1alpha2.PolicyConditionAccepted),
		//		Status:             metav1.ConditionFalse,
		//		ObservedGeneration: policy.Generation,
		//		LastTransitionTime: metav1.Time{Time: time.Now()},
		//		Reason:             string(gwv1alpha2.PolicyReasonInvalid),
		//		Message:            fmt.Sprintf("Failed to list RateLimitPolicies: %s", err),
		//	}
		//}
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

func getTargetNamespace(policy *gwpav1alpha1.RateLimitPolicy) string {
	if policy.Spec.TargetRef.Namespace == nil {
		return policy.Namespace
	}

	return string(*policy.Spec.TargetRef.Namespace)
}

// SetupWithManager sets up the controller with the Manager.
func (r *rateLimitPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwpav1alpha1.RateLimitPolicy{}).
		Complete(r)
}

// SetupWithManager sets up the controller with the Manager.
//func (r *rateLimitPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
//	return ctrl.NewControllerManagedBy(mgr).
//		For(&gwpav1alpha1.RateLimitPolicy{}).
//		Watches(
//			&source.Kind{Type: &gwv1beta1.HTTPRoute{}},
//			handler.EnqueueRequestsFromMapFunc(r.httpRoutesToRateLimitPolicies),
//			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
//				route, ok := obj.(*gwv1beta1.HTTPRoute)
//				if !ok {
//					log.Error().Msgf("unexpected object type %T", obj)
//					return false
//				}
//
//				return gwutils.IsActiveRoute(route.Status.Parents)
//			})),
//		).
//		Watches(
//			&source.Kind{Type: &gwv1alpha2.GRPCRoute{}},
//			handler.EnqueueRequestsFromMapFunc(r.grpcRoutesToRateLimitPolicies),
//			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
//				route, ok := obj.(*gwv1alpha2.GRPCRoute)
//				if !ok {
//					log.Error().Msgf("unexpected object type %T", obj)
//					return false
//				}
//
//				return gwutils.IsActiveRoute(route.Status.Parents)
//			})),
//		).
//		Watches(
//			&source.Kind{Type: &gwv1beta1.Gateway{}},
//			handler.EnqueueRequestsFromMapFunc(r.gatewayToRateLimitPolicies),
//			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
//				gateway, ok := obj.(*gwv1beta1.Gateway)
//				if !ok {
//					log.Error().Msgf("unexpected object type: %T", obj)
//					return false
//				}
//
//				gatewayClass, err := r.gatewayAPIClient.
//					GatewayV1beta1().
//					GatewayClasses().
//					Get(context.TODO(), string(gateway.Spec.GatewayClassName), metav1.GetOptions{})
//				if err != nil {
//					log.Error().Msgf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
//					return false
//				}
//
//				if gatewayClass.Spec.ControllerName != constants.GatewayController {
//					log.Warn().Msgf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, constants.GatewayController)
//					return false
//				}
//
//				return true
//			})),
//		).
//		Complete(r)
//}
//
//func (r *rateLimitPolicyReconciler) httpRoutesToRateLimitPolicies(obj client.Object) []reconcile.Request {
//	route, ok := obj.(*gwv1beta1.HTTPRoute)
//	if !ok {
//		log.Error().Msgf("unexpected object type %T", obj)
//		return nil
//	}
//
//	policies, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
//	if err != nil {
//		log.Error().Msgf("failed to list RateLimitPolicies: %s", err)
//		return nil
//	}
//
//	httpPolicies := make([]*gwpav1alpha1.RateLimitPolicy, 0)
//	for _, policy := range policies.Items {
//		if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && policy.Spec.TargetRef.Kind == "HTTPRoute" {
//			if len(policy.Spec.Match.Hostnames) == 0 && policy.Spec.Match.Route == nil {
//				continue
//			}
//
//			if policy.Spec.RateLimit.L7RateLimit == nil {
//				continue
//			}
//
//			httpPolicies = append(httpPolicies, &policy)
//		}
//	}
//
//	for _, ref := range route.Spec.ParentRefs {
//
//	}
//
//	gateway := r.gatewayAPIClient.GatewayV1beta1().Gateways(route.Namespace)
//
//	routeHostnames := make([]gwv1beta1.Hostname, 0)
//	for _, policy := range httpPolicies {
//		if len(policy.Spec.Match.Hostnames) > 0 {
//
//		}
//
//		if policy.Spec.Match.Route != nil {
//
//		}
//	}
//
//	return nil
//}
//
//func (r *rateLimitPolicyReconciler) grpcRoutesToRateLimitPolicies(obj client.Object) []reconcile.Request {
//	route, ok := obj.(*gwv1alpha2.GRPCRoute)
//	if !ok {
//		log.Error().Msgf("unexpected object type %T", obj)
//		return nil
//	}
//
//	policies, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
//	if err != nil {
//		log.Error().Msgf("failed to list RateLimitPolicies: %s", err)
//		return nil
//	}
//
//	grpcPolicies := make([]*gwpav1alpha1.RateLimitPolicy, 0)
//	for _, policy := range policies.Items {
//		if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && policy.Spec.TargetRef.Kind == "GRPCRoute" {
//			if len(policy.Spec.Match.Hostnames) == 0 && policy.Spec.Match.Route == nil {
//				continue
//			}
//
//			if policy.Spec.RateLimit.L7RateLimit == nil {
//				continue
//			}
//
//			grpcPolicies = append(grpcPolicies, &policy)
//		}
//	}
//
//	return nil
//}
//
//func (r *rateLimitPolicyReconciler) gatewayToRateLimitPolicies(obj client.Object) []reconcile.Request {
//	gateway, ok := obj.(*gwv1beta1.Gateway)
//	if !ok {
//		log.Error().Msgf("unexpected object type %T", obj)
//		return nil
//	}
//
//	if gwutils.IsActiveGateway(gateway) {
//		policies, err := r.policyAttachmentAPIClient.GatewayV1alpha1().RateLimitPolicies(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
//		if err != nil {
//			log.Error().Msgf("failed to list RateLimitPolicies: %s", err)
//			return nil
//		}
//
//		portBasedPolicies := make([]*gwpav1alpha1.RateLimitPolicy, 0)
//		for _, policy := range policies.Items {
//			if policy.Spec.TargetRef.Group == "gateway.networking.k8s.io" && policy.Spec.TargetRef.Kind == "Gateway" {
//				if policy.Spec.Match.Port == nil {
//					continue
//				}
//
//				if policy.Spec.RateLimit.L4RateLimit == nil {
//					continue
//				}
//
//				portBasedPolicies = append(portBasedPolicies, &policy)
//			}
//		}
//
//		listeners := map[gwv1beta1.SectionName]gwv1beta1.PortNumber{}
//		for _, listener := range gateway.Spec.Listeners {
//			listeners[listener.Name] = listener.Port
//		}
//
//		validListenerPorts := make(map[gwv1beta1.PortNumber]struct{})
//		for _, listenerStatus := range gateway.Status.Listeners {
//			if gwutils.IsListenerAccepted(listenerStatus) && gwutils.IsListenerProgrammed(listenerStatus) {
//				validListenerPorts[listeners[listenerStatus.Name]] = struct{}{}
//			}
//		}
//
//		var reconciles []reconcile.Request
//		for _, policy := range portBasedPolicies {
//			if _, ok := validListenerPorts[*policy.Spec.Match.Port]; ok {
//				reconciles = append(reconciles, reconcile.Request{
//					NamespacedName: types.NamespacedName{
//						Namespace: policy.Namespace,
//						Name:      policy.Name,
//					},
//				})
//			}
//		}
//
//		return reconciles
//	}
//
//	return nil
//}
