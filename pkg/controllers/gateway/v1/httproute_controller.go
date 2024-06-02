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
	"github.com/flomesh-io/fsm/pkg/gateway/routestatus"
	"github.com/flomesh-io/fsm/pkg/gateway/status"

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
	statusProcessor *routestatus.RouteStatusProcessor
}

func (r *httpRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewHTTPRouteReconciler returns a new HTTPRoute Reconciler
func NewHTTPRouteReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &httpRouteReconciler{
		recorder:        ctx.Manager.GetEventRecorderFor("HTTPRoute"),
		fctx:            ctx,
		statusProcessor: &routestatus.RouteStatusProcessor{Ctx: ctx},
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

	routeStatus, err := r.statusProcessor.ProcessRouteStatus(ctx, httpRoute)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(routeStatus) > 0 {
		r.fctx.StatusUpdater.Send(status.Update{
			Resource:       &gwv1.HTTPRoute{},
			NamespacedName: client.ObjectKeyFromObject(httpRoute),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				hr, ok := obj.(*gwv1.HTTPRoute)
				if !ok {
					log.Error().Msgf("Unexpected object type %T", obj)
				}
				hrCopy := hr.DeepCopy()
				hrCopy.Status.Parents = routeStatus

				return hrCopy
			}),
		})
	}

	r.fctx.GatewayEventHandler.OnAdd(httpRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *httpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.HTTPRoute{}).
		Complete(r); err != nil {
		return err
	}

	return addHTTPRouteIndexers(context.Background(), mgr)
}

func addHTTPRouteIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.HTTPRoute{}, constants.GatewayHTTPRouteIndex, gatewayHTTPRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.HTTPRoute{}, constants.BackendHTTPRouteIndex, backendHTTPRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func gatewayHTTPRouteIndexFunc(obj client.Object) []string {
	httproute := obj.(*gwv1.HTTPRoute)
	var gateways []string
	for _, parent := range httproute.Spec.ParentRefs {
		if parent.Kind == nil || string(*parent.Kind) == constants.GatewayAPIGatewayKind {
			gateways = append(gateways,
				types.NamespacedName{
					Namespace: gwutils.NamespaceDerefOr(parent.Namespace, httproute.Namespace),
					Name:      string(parent.Name),
				}.String(),
			)
		}
	}

	return gateways
}

func backendHTTPRouteIndexFunc(obj client.Object) []string {
	httproute := obj.(*gwv1.HTTPRoute)
	var backendRefs []string
	for _, rule := range httproute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				backendRefs = append(backendRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(backend.Namespace, httproute.Namespace),
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
								Namespace: gwutils.NamespaceDerefOr(mirror.Namespace, httproute.Namespace),
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
							Namespace: gwutils.NamespaceDerefOr(mirror.Namespace, httproute.Namespace),
							Name:      string(mirror.Name),
						}.String(),
					)
				}
			}
		}
	}

	return backendRefs
}
