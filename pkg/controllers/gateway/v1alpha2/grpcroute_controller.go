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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/gateway/status"
)

type grpcRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *status.RouteStatusProcessor
}

// NewGRPCRouteReconciler returns a new GRPCRoute.Reconciler
func NewGRPCRouteReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &grpcRouteReconciler{
		recorder:        ctx.Manager.GetEventRecorderFor("GRPCRoute"),
		fctx:            ctx,
		statusProcessor: &status.RouteStatusProcessor{Informers: ctx.InformerCollection},
	}
}

// Reconcile reads that state of the cluster for a GRPCRoute object and makes changes based on the state read
func (r *grpcRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	grpcRoute := &gwv1alpha2.GRPCRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, grpcRoute)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwv1alpha2.GRPCRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if grpcRoute.DeletionTimestamp != nil {
		r.fctx.EventHandler.OnDelete(grpcRoute)
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

	r.fctx.EventHandler.OnAdd(grpcRoute)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *grpcRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha2.GRPCRoute{}).
		Complete(r)
}
