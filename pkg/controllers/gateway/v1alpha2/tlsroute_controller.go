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

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status/routes"

	"k8s.io/apimachinery/pkg/util/sets"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	whblder "github.com/flomesh-io/fsm/pkg/webhook/builder"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type tlsRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *routes.RouteStatusProcessor
	webhook         whtypes.Register
}

func (r *tlsRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewTLSRouteReconciler returns a new TLSRoute.Reconciler
func NewTLSRouteReconciler(ctx *fctx.ControllerContext, webhook whtypes.Register) controllers.Reconciler {
	recorder := ctx.Manager.GetEventRecorderFor("TLSRoute")
	return &tlsRouteReconciler{
		recorder:        recorder,
		fctx:            ctx,
		statusProcessor: routes.NewRouteStatusProcessor(ctx.Manager.GetCache(), recorder, ctx.StatusUpdater),
		webhook:         webhook,
	}
}

// Reconcile reconciles a TLSRoute object
func (r *tlsRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	tlsRoute := &gwv1alpha2.TLSRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, tlsRoute)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1alpha2.TLSRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if tlsRoute.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(tlsRoute)
		return ctrl.Result{}, nil
	}

	rsu := routes.NewRouteStatusUpdate(
		tlsRoute,
		tlsRoute.GroupVersionKind(),
		tlsRoute.Spec.Hostnames,
		gwutils.ToSlicePtr(tlsRoute.Status.Parents),
	)
	if err := r.statusProcessor.Process(ctx, rsu, tlsRoute.Spec.ParentRefs); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.GatewayEventHandler.OnAdd(tlsRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *tlsRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := whblder.WebhookManagedBy(mgr).
		For(&gwv1alpha2.TLSRoute{}).
		WithDefaulter(r.webhook).
		WithValidator(r.webhook).
		RecoverPanic().
		Complete(); err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha2.TLSRoute{}).
		Watches(&gwv1.Gateway{}, handler.EnqueueRequestsFromMapFunc(r.gatewayToTLSRoutes)).
		Complete(r); err != nil {
		return err
	}

	return addTLSRouteIndexers(context.Background(), mgr)
}

func (r *tlsRouteReconciler) gatewayToTLSRoutes(ctx context.Context, object client.Object) []reconcile.Request {
	gateway, ok := object.(*gwv1.Gateway)
	if !ok {
		log.Error().Msgf("Unexpected type %T", object)
		return nil
	}

	var requests []reconcile.Request

	list := &gwv1alpha2.TLSRouteList{}
	if err := r.fctx.Manager.GetCache().List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayTLSRouteIndex, client.ObjectKeyFromObject(gateway).String()),
	}); err != nil {
		log.Error().Msgf("Failed to list TLSRoutes: %v", err)
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

func addTLSRouteIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.TLSRoute{}, constants.GatewayTLSRouteIndex, func(obj client.Object) []string {
		tlsRoute := obj.(*gwv1alpha2.TLSRoute)
		var gateways []string
		for _, parent := range tlsRoute.Spec.ParentRefs {
			if string(*parent.Kind) == constants.GatewayAPIGatewayKind {
				gateways = append(gateways,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(parent.Namespace, tlsRoute.Namespace),
						Name:      string(parent.Name),
					}.String(),
				)
			}
		}
		return gateways
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.TLSRoute{}, constants.BackendTLSRouteIndex, backendTLSRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1alpha2.TLSRoute{}, constants.CrossNamespaceBackendNamespaceTLSRouteIndex, crossNamespaceBackendNamespaceTLSRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func backendTLSRouteIndexFunc(obj client.Object) []string {
	tlsRoute := obj.(*gwv1alpha2.TLSRoute)
	var backendRefs []string
	for _, rule := range tlsRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				backendRefs = append(backendRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(backend.Namespace, tlsRoute.Namespace),
						Name:      string(backend.Name),
					}.String(),
				)
			}
		}
	}

	return backendRefs
}

func crossNamespaceBackendNamespaceTLSRouteIndexFunc(obj client.Object) []string {
	tlsRoute := obj.(*gwv1alpha2.TLSRoute)
	namespaces := sets.New[string]()
	for _, rule := range tlsRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				if backend.Namespace != nil && string(*backend.Namespace) != tlsRoute.Namespace {
					namespaces.Insert(string(*backend.Namespace))
				}
			}
		}
	}

	return namespaces.UnsortedList()
}
