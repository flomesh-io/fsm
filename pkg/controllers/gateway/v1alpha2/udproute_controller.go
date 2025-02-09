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

package v1alpha2

import (
	"context"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

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
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type udpRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *routes.RouteStatusProcessor
	webhook         whtypes.Register
}

func (r *udpRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewUDPRouteReconciler returns a new UDPRoute Reconciler
func NewUDPRouteReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	recorder := ctx.Manager.GetEventRecorderFor("UDPRoute")
	return &udpRouteReconciler{
		recorder:        recorder,
		fctx:            ctx,
		statusProcessor: routes.NewRouteStatusProcessor(ctx.Manager.GetCache(), recorder, ctx.StatusUpdater),
		webhook:         webhook,
	}
}

// Reconcile reconciles a UDPRoute object
func (r *udpRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	udpRoute := &gwv1alpha2.UDPRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, udpRoute)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1alpha2.UDPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if udpRoute.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(udpRoute)
		return ctrl.Result{}, nil
	}

	rsu := routes.NewRouteStatusUpdate(
		udpRoute,
		udpRoute.GroupVersionKind(),
		nil,
		gwutils.ToSlicePtr(udpRoute.Status.Parents),
	)
	if err := r.statusProcessor.Process(ctx, rsu, udpRoute.Spec.ParentRefs); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.GatewayEventHandler.OnAdd(udpRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *udpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwv1alpha2.UDPRoute{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha2.UDPRoute{}).
		Watches(&gwv1.Gateway{}, handler.EnqueueRequestsFromMapFunc(r.gatewayToUDPRoutes)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.serviceToUDPRoutes)).
		Watches(&gwv1beta1.ReferenceGrant{}, handler.EnqueueRequestsFromMapFunc(r.referenceGrantToUDPRoutes)).
		Watches(&gwpav1alpha2.RouteRuleFilterPolicy{}, handler.EnqueueRequestsFromMapFunc(r.routeRuleFilterPolicyToUDPRoutes)).
		Complete(r); err != nil {
		return err
	}

	return addUDPRouteIndexers(context.Background(), mgr)
}

func (r *udpRouteReconciler) gatewayToUDPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	gateway, ok := object.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	list := &gwv1alpha2.UDPRouteList{}
	if err := r.fctx.Manager.GetCache().List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayUDPRouteIndex, client.ObjectKeyFromObject(gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
		return nil
	}

	for _, route := range list.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: route.Namespace,
				Name:      route.Name,
			},
		})
	}

	return requests
}

func (r *udpRouteReconciler) serviceToUDPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	service, ok := object.(*corev1.Service)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	list := &gwv1alpha2.UDPRouteList{}
	if err := r.fctx.Manager.GetCache().List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendUDPRouteIndex, client.ObjectKeyFromObject(service).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
		return nil
	}

	for _, route := range list.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: route.Namespace,
				Name:      route.Name,
			},
		})
	}

	return requests
}

func (r *udpRouteReconciler) referenceGrantToUDPRoutes(ctx context.Context, obj client.Object) []reconcile.Request {
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
		if from.Group == gwv1.GroupName && from.Kind == constants.GatewayAPIUDPRouteKind {
			fromNamespaces.Insert(string(from.Namespace))
		}
	}

	// Not for UDPRoute
	if fromNamespaces.Len() == 0 {
		return nil
	}

	list := &gwv1alpha2.UDPRouteList{}
	if err := r.fctx.Manager.GetCache().List(ctx, list, &client.ListOptions{
		// This index implies that the UDPRoute has a backend of type Service in the same namespace as the ReferenceGrant
		FieldSelector: fields.OneTermEqualSelector(constants.CrossNamespaceBackendNamespaceUDPRouteIndex, refGrant.Namespace),
	}); err != nil {
		log.Error().Msgf("Failed to list UDPRoutes: %v", err)
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

func (r *udpRouteReconciler) routeRuleFilterPolicyToUDPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	policy, ok := object.(*gwpav1alpha2.RouteRuleFilterPolicy)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	for _, targetRef := range policy.Spec.TargetRefs {
		if targetRef.Kind != constants.GatewayAPIUDPRouteKind {
			continue
		}

		udpRoute := &gwv1alpha2.UDPRoute{}
		key := types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      string(targetRef.Name),
		}
		if err := r.fctx.Manager.GetCache().Get(ctx, key, udpRoute); err != nil {
			log.Error().Msgf("Failed to get UDPRoute: %v", key.String())
			continue
		}

		for _, rule := range udpRoute.Spec.Rules {
			if rule.Name == nil {
				continue
			}

			if targetRef.Rule == *rule.Name {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: udpRoute.Namespace,
						Name:      udpRoute.Name,
					},
				})

				break
			}
		}
	}

	return requests
}

func addUDPRouteIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.UDPRoute{}, constants.GatewayUDPRouteIndex, func(obj client.Object) []string {
		udpRoute := obj.(*gwv1alpha2.UDPRoute)
		var gateways []string
		for _, parent := range udpRoute.Spec.ParentRefs {
			if string(*parent.Kind) == constants.GatewayAPIGatewayKind {
				gateways = append(gateways,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(parent.Namespace, udpRoute.Namespace),
						Name:      string(parent.Name),
					}.String(),
				)
			}
		}
		return gateways
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.UDPRoute{}, constants.BackendUDPRouteIndex, backendUDPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.UDPRoute{}, constants.CrossNamespaceBackendNamespaceUDPRouteIndex, crossNamespaceBackendNamespaceUDPRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func backendUDPRouteIndexFunc(obj client.Object) []string {
	udpRoute := obj.(*gwv1alpha2.UDPRoute)
	var backendRefs []string
	for _, rule := range udpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				backendRefs = append(backendRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(backend.Namespace, udpRoute.Namespace),
						Name:      string(backend.Name),
					}.String(),
				)
			}
		}
	}

	return backendRefs
}

func crossNamespaceBackendNamespaceUDPRouteIndexFunc(obj client.Object) []string {
	udpRoute := obj.(*gwv1alpha2.UDPRoute)
	namespaces := sets.New[string]()
	for _, rule := range udpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				if backend.Namespace != nil && string(*backend.Namespace) != udpRoute.Namespace {
					namespaces.Insert(string(*backend.Namespace))
				}
			}
		}
	}

	return namespaces.UnsortedList()
}
