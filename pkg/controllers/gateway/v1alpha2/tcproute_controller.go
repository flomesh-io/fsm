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

	"github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/apimachinery/pkg/util/sets"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"

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
	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"
)

type tcpRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *routes.RouteStatusProcessor
	webhook         whtypes.Register
}

func (r *tcpRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewTCPRouteReconciler returns a new TCPRoute Reconciler
func NewTCPRouteReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	recorder := ctx.Manager.GetEventRecorderFor("TCPRoute")
	return &tcpRouteReconciler{
		recorder:        recorder,
		fctx:            ctx,
		statusProcessor: routes.NewRouteStatusProcessor(ctx.Manager.GetCache(), recorder, ctx.StatusUpdater),
		webhook:         webhook,
	}
}

// Reconcile reconciles a TCPRoute object
func (r *tcpRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	tcpRoute := &gwv1alpha2.TCPRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, tcpRoute)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1alpha2.TCPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if tcpRoute.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(tcpRoute)
		return ctrl.Result{}, nil
	}

	rsu := routes.NewRouteStatusUpdate(
		tcpRoute,
		tcpRoute.GroupVersionKind(),
		nil,
		gwutils.ToSlicePtr(tcpRoute.Status.Parents),
	)
	if err := r.statusProcessor.Process(ctx, rsu, tcpRoute.Spec.ParentRefs); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.GatewayEventHandler.OnAdd(tcpRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *tcpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwv1alpha2.TCPRoute{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha2.TCPRoute{}).
		Watches(&gwv1.Gateway{}, handler.EnqueueRequestsFromMapFunc(r.gatewayToTCPRoutes)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.serviceToTCPRoutes)).
		Watches(&gwv1alpha3.BackendTLSPolicy{}, handler.EnqueueRequestsFromMapFunc(r.backendTLSToTCPRoutes)).
		Watches(&gwv1beta1.ReferenceGrant{}, handler.EnqueueRequestsFromMapFunc(r.referenceGrantToTCPRoutes)).
		Watches(&gwpav1alpha2.RouteRuleFilterPolicy{}, handler.EnqueueRequestsFromMapFunc(r.routeRuleFilterPolicyToTCPRoutes)).
		Complete(r); err != nil {
		return err
	}

	return addTCPRouteIndexers(context.Background(), mgr)
}

func (r *tcpRouteReconciler) gatewayToTCPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	gateway, ok := object.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	list := &gwv1alpha2.TCPRouteList{}
	if err := r.fctx.Manager.GetCache().List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTCPRouteIndex, client.ObjectKeyFromObject(gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
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

func (r *tcpRouteReconciler) serviceToTCPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	service, ok := object.(*corev1.Service)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	list := &gwv1alpha2.TCPRouteList{}
	if err := r.fctx.Manager.GetCache().List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.BackendTCPRouteIndex, client.ObjectKeyFromObject(service).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
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

func (r *tcpRouteReconciler) backendTLSToTCPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
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

	return r.policyToTCPRoutes(ctx, policy, targetRefs)
}

func (r *tcpRouteReconciler) policyToTCPRoutes(ctx context.Context, policy client.Object, targetRefs []gwv1alpha2.NamespacedPolicyTargetReference) []reconcile.Request {
	var requests []reconcile.Request

	for _, targetRef := range targetRefs {
		if targetRef.Group != corev1.GroupName {
			continue
		}

		if targetRef.Kind != constants.KubernetesServiceKind {
			continue
		}

		list := &gwv1alpha2.TCPRouteList{}
		if err := r.fctx.Manager.GetCache().List(ctx, list, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(constants.BackendTCPRouteIndex, types.NamespacedName{
				Namespace: gwutils.NamespaceDerefOr(targetRef.Namespace, policy.GetNamespace()),
				Name:      string(targetRef.Name),
			}.String()),
		}); err != nil {
			log.Error().Msgf("Failed to list TCPRoutes: %v", err)
			continue
		}

		for _, tcpRoute := range list.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tcpRoute.Namespace,
					Name:      tcpRoute.Name,
				},
			})
		}
	}

	return requests
}

func (r *tcpRouteReconciler) referenceGrantToTCPRoutes(ctx context.Context, obj client.Object) []reconcile.Request {
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
		if from.Group == gwv1.GroupName && from.Kind == constants.GatewayAPITCPRouteKind {
			fromNamespaces.Insert(string(from.Namespace))
		}
	}

	// Not for TCPRoute
	if fromNamespaces.Len() == 0 {
		return nil
	}

	list := &gwv1alpha2.TCPRouteList{}
	if err := r.fctx.Manager.GetCache().List(ctx, list, &client.ListOptions{
		// This index implies that the TCPRoute has a backend of type Service in the same namespace as the ReferenceGrant
		FieldSelector: fields.OneTermEqualSelector(constants.CrossNamespaceBackendNamespaceTCPRouteIndex, refGrant.Namespace),
	}); err != nil {
		log.Error().Msgf("Failed to list TCPRoutes: %v", err)
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

func (r *tcpRouteReconciler) routeRuleFilterPolicyToTCPRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	policy, ok := object.(*gwpav1alpha2.RouteRuleFilterPolicy)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	for _, targetRef := range policy.Spec.TargetRefs {
		if targetRef.Kind != constants.GatewayAPITCPRouteKind {
			continue
		}

		tcpRoute := &gwv1alpha2.TCPRoute{}
		key := types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      string(targetRef.Name),
		}
		if err := r.fctx.Manager.GetCache().Get(ctx, key, tcpRoute); err != nil {
			log.Error().Msgf("Failed to get TCPRoute: %v", key.String())
			continue
		}

		for _, rule := range tcpRoute.Spec.Rules {
			if rule.Name == nil {
				continue
			}

			if targetRef.Rule == *rule.Name {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: tcpRoute.Namespace,
						Name:      tcpRoute.Name,
					},
				})

				break
			}
		}
	}

	return requests
}

func addTCPRouteIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.TCPRoute{}, constants.GatewayTCPRouteIndex, func(obj client.Object) []string {
		tcpRoute := obj.(*gwv1alpha2.TCPRoute)
		var gateways []string
		for _, parent := range tcpRoute.Spec.ParentRefs {
			if string(*parent.Kind) == constants.GatewayAPIGatewayKind {
				gateways = append(gateways,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(parent.Namespace, tcpRoute.Namespace),
						Name:      string(parent.Name),
					}.String(),
				)
			}
		}
		return gateways
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.TCPRoute{}, constants.BackendTCPRouteIndex, backendTCPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.TCPRoute{}, constants.CrossNamespaceBackendNamespaceTCPRouteIndex, crossNamespaceBackendNamespaceTCPRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func backendTCPRouteIndexFunc(obj client.Object) []string {
	tcpRoute := obj.(*gwv1alpha2.TCPRoute)
	var backendRefs []string
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				backendRefs = append(backendRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(backend.Namespace, tcpRoute.Namespace),
						Name:      string(backend.Name),
					}.String(),
				)
			}
		}
	}

	return backendRefs
}

func crossNamespaceBackendNamespaceTCPRouteIndexFunc(obj client.Object) []string {
	tcpRoute := obj.(*gwv1alpha2.TCPRoute)
	namespaces := sets.New[string]()
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				if backend.Namespace != nil && string(*backend.Namespace) != tcpRoute.Namespace {
					namespaces.Insert(string(*backend.Namespace))
				}
			}
		}
	}

	return namespaces.UnsortedList()
}
