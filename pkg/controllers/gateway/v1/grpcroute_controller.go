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

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type grpcRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *routestatus.RouteStatusProcessor
}

func (r *grpcRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewGRPCRouteReconciler returns a new GRPCRoute.Reconciler
func NewGRPCRouteReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &grpcRouteReconciler{
		recorder:        ctx.Manager.GetEventRecorderFor("GRPCRoute"),
		fctx:            ctx,
		statusProcessor: &routestatus.RouteStatusProcessor{Ctx: ctx},
	}
}

// Reconcile reads that state of the cluster for a GRPCRoute object and makes changes based on the state read
func (r *grpcRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	grpcRoute := &gwv1.GRPCRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, grpcRoute)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1.GRPCRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if grpcRoute.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(grpcRoute)
		return ctrl.Result{}, nil
	}

	routeStatus, err := r.statusProcessor.ProcessRouteStatus(ctx, grpcRoute)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(routeStatus) > 0 {
		grpcRoute.Status.Parents = routeStatus
		if err := r.fctx.Status().Update(ctx, grpcRoute); err != nil {
			return ctrl.Result{}, err
		}
	}

	r.fctx.GatewayEventHandler.OnAdd(grpcRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *grpcRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.GRPCRoute{}).
		Complete(r); err != nil {
		return err
	}

	return addGRPCRouteIndexers(context.Background(), mgr)
}

func addGRPCRouteIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.GRPCRoute{}, constants.GatewayGRPCRouteIndex, gatewayGRPCRouteIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.GRPCRoute{}, constants.BackendGRPCRouteIndex, backendGRPCRouteIndexFunc); err != nil {
		return err
	}

	return nil
}

func gatewayGRPCRouteIndexFunc(obj client.Object) []string {
	grpcroute := obj.(*gwv1.GRPCRoute)
	var gateways []string
	for _, parent := range grpcroute.Spec.ParentRefs {
		if parent.Kind == nil || string(*parent.Kind) == constants.GatewayAPIGatewayKind {
			// If an explicit Gateway namespace is not provided, use the GRPCRoute namespace to
			// lookup the provided Gateway Name.
			gateways = append(gateways,
				types.NamespacedName{
					Namespace: gwutils.Namespace(parent.Namespace, grpcroute.Namespace),
					Name:      string(parent.Name),
				}.String(),
			)
		}
	}
	return gateways
}

func backendGRPCRouteIndexFunc(obj client.Object) []string {
	grpcroute := obj.(*gwv1.GRPCRoute)
	var backendRefs []string
	for _, rule := range grpcroute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind == nil || string(*backend.Kind) == constants.KubernetesServiceKind {
				backendRefs = append(backendRefs,
					types.NamespacedName{
						Namespace: gwutils.Namespace(backend.Namespace, grpcroute.Namespace),
						Name:      string(backend.Name),
					}.String(),
				)
			}

			for _, filter := range backend.Filters {
				if filter.Type == gwv1.GRPCRouteFilterRequestMirror {
					if filter.RequestMirror.BackendRef.Kind == nil || string(*filter.RequestMirror.BackendRef.Kind) == constants.KubernetesServiceKind {
						mirror := filter.RequestMirror.BackendRef
						backendRefs = append(backendRefs,
							types.NamespacedName{
								Namespace: gwutils.Namespace(mirror.Namespace, grpcroute.Namespace),
								Name:      string(mirror.Name),
							}.String(),
						)
					}
				}
			}
		}

		for _, filter := range rule.Filters {
			if filter.Type == gwv1.GRPCRouteFilterRequestMirror {
				if filter.RequestMirror.BackendRef.Kind == nil || string(*filter.RequestMirror.BackendRef.Kind) == constants.KubernetesServiceKind {
					mirror := filter.RequestMirror.BackendRef
					backendRefs = append(backendRefs,
						types.NamespacedName{
							Namespace: gwutils.Namespace(mirror.Namespace, grpcroute.Namespace),
							Name:      string(mirror.Name),
						}.String(),
					)
				}
			}
		}
	}

	return backendRefs
}
