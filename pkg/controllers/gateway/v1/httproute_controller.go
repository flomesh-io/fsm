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

package v1

import (
	"context"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/apimachinery/pkg/util/sets"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"k8s.io/utils/ptr"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type httpRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *routes.RouteStatusProcessor
	webhook         whtypes.Register
}

func (r *httpRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewHTTPRouteReconciler returns a new HTTPRoute Reconciler
func NewHTTPRouteReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	return &httpRouteReconciler{
		recorder:        ctx.Manager.GetEventRecorderFor("HTTPRoute"),
		fctx:            ctx,
		statusProcessor: routes.NewRouteStatusProcessor(ctx.Manager.GetCache(), ctx.StatusUpdater),
		webhook:         webhook,
	}
}

// Reconcile reads that state of the cluster for a HTTPRoute object and makes changes based on the state read
func (r *httpRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	httpRoute := &gwv1.HTTPRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, httpRoute)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if httpRoute.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(httpRoute)
		return ctrl.Result{}, nil
	}

	rsu := routes.NewRouteStatusUpdate(
		httpRoute,
		&httpRoute.ObjectMeta,
		&httpRoute.TypeMeta,
		httpRoute.Spec.Hostnames,
		gwutils.ToSlicePtr(httpRoute.Status.Parents),
	)
	if err := r.statusProcessor.Process(ctx, rsu, httpRoute.Spec.ParentRefs); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.GatewayEventHandler.OnAdd(httpRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *httpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwv1.HTTPRoute{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.HTTPRoute{}).
		Watches(&gwv1alpha3.BackendTLSPolicy{}, handler.EnqueueRequestsFromMapFunc(r.backendTLSToHTTPRoutes)).
		Watches(&gwpav1alpha2.BackendLBPolicy{}, handler.EnqueueRequestsFromMapFunc(r.backendLBToHTTPRoutes)).
		Watches(&gwpav1alpha2.HealthCheckPolicy{}, handler.EnqueueRequestsFromMapFunc(r.healthCheckToHTTPRoutes)).
		Watches(&gwpav1alpha2.RetryPolicy{}, handler.EnqueueRequestsFromMapFunc(r.retryToHTTPRoutes)).
		Watches(&gwv1beta1.ReferenceGrant{}, handler.EnqueueRequestsFromMapFunc(r.referenceGrantToHTTPRoutes)).
		Complete(r); err != nil {
		return err
	}

	return addHTTPRouteIndexers(context.Background(), mgr)
}

func (r *httpRouteReconciler) backendTLSToHTTPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	policy, ok := object.(*gwv1alpha3.BackendTLSPolicy)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	targetRefs := make([]gwv1alpha2.NamespacedPolicyTargetReference, len(policy.Spec.TargetRefs))
	for i, ref := range policy.Spec.TargetRefs {
		targetRefs[i] = gwv1alpha2.NamespacedPolicyTargetReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Name:      ref.Name,
			Namespace: ptr.To(gwv1.Namespace(policy.Namespace)),
		}
	}

	return r.policyToHTTPRoutes(ctx, policy, targetRefs)
}

func (r *httpRouteReconciler) backendLBToHTTPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	policy, ok := object.(*gwpav1alpha2.BackendLBPolicy)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	targetRefs := make([]gwv1alpha2.NamespacedPolicyTargetReference, len(policy.Spec.TargetRefs))
	for i, ref := range policy.Spec.TargetRefs {
		targetRefs[i] = gwv1alpha2.NamespacedPolicyTargetReference{
			Group:     ref.Group,
			Kind:      ref.Kind,
			Name:      ref.Name,
			Namespace: ptr.To(gwv1.Namespace(policy.Namespace)),
		}
	}

	return r.policyToHTTPRoutes(ctx, policy, targetRefs)
}

func (r *httpRouteReconciler) healthCheckToHTTPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	policy, ok := object.(*gwpav1alpha2.HealthCheckPolicy)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	return r.policyToHTTPRoutes(ctx, policy, policy.Spec.TargetRefs)
}

func (r *httpRouteReconciler) retryToHTTPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	policy, ok := object.(*gwpav1alpha2.HealthCheckPolicy)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	return r.policyToHTTPRoutes(ctx, policy, policy.Spec.TargetRefs)
}

func (r *httpRouteReconciler) policyToHTTPRoutes(ctx context.Context, policy client.Object, targetRefs []gwv1alpha2.NamespacedPolicyTargetReference) []reconcile.Request {
	var requests []reconcile.Request

	for _, targetRef := range targetRefs {
		if targetRef.Group != corev1.GroupName {
			continue
		}

		if targetRef.Kind != constants.KubernetesServiceKind {
			continue
		}

		httpRouteList := &gwv1.HTTPRouteList{}
		if err := r.fctx.Manager.GetCache().List(ctx, httpRouteList, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(constants.BackendHTTPRouteIndex, types.NamespacedName{
				Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()),
				Name:      string(targetRef.Name),
			}.String()),
		}); err != nil {
			log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
			continue
		}

		for _, httpRoute := range httpRouteList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: httpRoute.Namespace,
					Name:      httpRoute.Name,
				},
			})
		}
	}

	return requests
}

func (r *httpRouteReconciler) referenceGrantToHTTPRoutes(ctx context.Context, obj client.Object) []reconcile.Request {
	refGrant, ok := obj.(*gwv1beta1.ReferenceGrant)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	isConcerned := false
	for _, target := range refGrant.Spec.To {
		if target.Kind == constants.KubernetesServiceKind {
			isConcerned = true
		}
	}

	// Not target for Service
	if !isConcerned {
		return nil
	}

	fromNamespaces := sets.New[string]()
	for _, from := range refGrant.Spec.From {
		if from.Group == gwv1.GroupName && from.Kind == constants.GatewayAPIHTTPRouteKind {
			fromNamespaces.Insert(string(from.Namespace))
		}
	}

	// Not for HTTPRoute
	if fromNamespaces.Len() == 0 {
		return nil
	}

	list := &gwv1.HTTPRouteList{}
	if err := r.fctx.Manager.GetCache().List(ctx, list, &client.ListOptions{
		// This index implies that the HTTPRoute has a backend of type Service in the same namespace as the ReferenceGrant
		FieldSelector: fields.OneTermEqualSelector(constants.CrossNamespaceBackendNamespaceHTTPRouteIndex, refGrant.Namespace),
	}); err != nil {
		log.Error().Msgf("Failed to list HTTPRoutes: %v", err)
		return nil
	}

	if len(list.Items) == 0 {
		return nil
	}

	requests := make([]reconcile.Request, 0)
	for _, ns := range fromNamespaces.UnsortedList() {
		for _, h := range list.Items {
			// not controlled by this ReferenceGrant
			if h.Namespace != ns {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: h.Namespace,
					Name:      h.Name,
				},
			})
		}
	}

	return requests
}

func addHTTPRouteIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.HTTPRoute{}, constants.GatewayHTTPRouteIndex, gatewayHTTPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.HTTPRoute{}, constants.ExtensionFilterHTTPRouteIndex, extensionFilterHTTPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.HTTPRoute{}, constants.BackendHTTPRouteIndex, backendHTTPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.HTTPRoute{}, constants.CrossNamespaceBackendNamespaceHTTPRouteIndex, crossNamespaceBackendNamespaceHTTPRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func gatewayHTTPRouteIndexFunc(obj client.Object) []string {
	httpRoute := obj.(*gwv1.HTTPRoute)
	var gateways []string
	for _, parent := range httpRoute.Spec.ParentRefs {
		if parent.Kind == nil || string(*parent.Kind) == constants.GatewayAPIGatewayKind {
			gateways = append(gateways,
				types.NamespacedName{
					Namespace: gwutils.NamespaceDerefOr(parent.Namespace, httpRoute.Namespace),
					Name:      string(parent.Name),
				}.String(),
			)
		}
	}

	return gateways
}

func extensionFilterHTTPRouteIndexFunc(obj client.Object) []string {
	httpRoute := obj.(*gwv1.HTTPRoute)

	var filters []string
	for _, rule := range httpRoute.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gwv1.HTTPRouteFilterExtensionRef {
				continue
			}
			if filter.ExtensionRef.Kind == constants.GatewayAPIExtensionFilterKind && filter.ExtensionRef.Group == extv1alpha1.GroupName {
				filters = append(filters,
					types.NamespacedName{
						Namespace: httpRoute.Namespace,
						Name:      string(filter.ExtensionRef.Name),
					}.String(),
				)
			}
		}

		for _, bk := range rule.BackendRefs {
			for _, filter := range bk.Filters {
				if filter.Type != gwv1.HTTPRouteFilterExtensionRef {
					continue
				}
				if filter.ExtensionRef.Kind == constants.GatewayAPIExtensionFilterKind && filter.ExtensionRef.Group == extv1alpha1.GroupName {
					filters = append(filters,
						types.NamespacedName{
							Namespace: httpRoute.Namespace,
							Name:      string(filter.ExtensionRef.Name),
						}.String(),
					)
				}
			}
		}
	}

	return filters
}

func backendHTTPRouteIndexFunc(obj client.Object) []string {
	httpRoute := obj.(*gwv1.HTTPRoute)
	var backendRefs []string
	for _, rule := range httpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				backendRefs = append(backendRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(backend.Namespace, httpRoute.Namespace),
						Name:      string(backend.Name),
					}.String(),
				)
			}

			for _, filter := range backend.Filters {
				if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
					if filter.RequestMirror.BackendRef.Kind == nil || string(*filter.RequestMirror.BackendRef.Kind) == constants.KubernetesServiceKind {
						mirror := filter.RequestMirror.BackendRef
						backendRefs = append(backendRefs,
							types.NamespacedName{
								Namespace: gwutils.NamespaceDerefOr(mirror.Namespace, httpRoute.Namespace),
								Name:      string(mirror.Name),
							}.String(),
						)
					}
				}
			}
		}

		for _, filter := range rule.Filters {
			if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
				if filter.RequestMirror.BackendRef.Kind == nil || string(*filter.RequestMirror.BackendRef.Kind) == constants.KubernetesServiceKind {
					mirror := filter.RequestMirror.BackendRef
					backendRefs = append(backendRefs,
						types.NamespacedName{
							Namespace: gwutils.NamespaceDerefOr(mirror.Namespace, httpRoute.Namespace),
							Name:      string(mirror.Name),
						}.String(),
					)
				}
			}
		}
	}

	return backendRefs
}

func crossNamespaceBackendNamespaceHTTPRouteIndexFunc(obj client.Object) []string {
	httpRoute := obj.(*gwv1.HTTPRoute)
	namespaces := sets.New[string]()
	for _, rule := range httpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				if backend.Namespace != nil && string(*backend.Namespace) != httpRoute.Namespace {
					namespaces.Insert(string(*backend.Namespace))
				}
			}

			for _, filter := range backend.Filters {
				if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
					if filter.RequestMirror.BackendRef.Kind == nil || string(*filter.RequestMirror.BackendRef.Kind) == constants.KubernetesServiceKind {
						mirror := filter.RequestMirror.BackendRef
						if mirror.Namespace != nil && string(*mirror.Namespace) != httpRoute.Namespace {
							namespaces.Insert(string(*mirror.Namespace))
						}
					}
				}
			}
		}

		for _, filter := range rule.Filters {
			if filter.Type == gwv1.HTTPRouteFilterRequestMirror {
				if filter.RequestMirror.BackendRef.Kind == nil || string(*filter.RequestMirror.BackendRef.Kind) == constants.KubernetesServiceKind {
					mirror := filter.RequestMirror.BackendRef
					if mirror.Namespace != nil && string(*mirror.Namespace) != httpRoute.Namespace {
						namespaces.Insert(string(*mirror.Namespace))
					}
				}
			}
		}
	}

	return namespaces.UnsortedList()
}
