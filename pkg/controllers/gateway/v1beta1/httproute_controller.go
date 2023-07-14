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
	"github.com/flomesh-io/fsm-classic/controllers"
	fctx "github.com/flomesh-io/fsm-classic/pkg/context"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/status"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type httpRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.FsmContext
	statusProcessor *status.RouteStatusProcessor
}

func NewHTTPRouteReconciler(ctx *fctx.FsmContext) controllers.Reconciler {
	return &httpRouteReconciler{
		recorder:        ctx.Manager.GetEventRecorderFor("HTTPRoute"),
		fctx:            ctx,
		statusProcessor: &status.RouteStatusProcessor{Fctx: ctx},
	}
}

func (r *httpRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	httpRoute := &gwv1beta1.HTTPRoute{}
	err := r.fctx.Get(ctx, req.NamespacedName, httpRoute)
	if errors.IsNotFound(err) {
		r.fctx.EventHandler.OnDelete(&gwv1beta1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if httpRoute.DeletionTimestamp != nil {
		r.fctx.EventHandler.OnDelete(httpRoute)
		return ctrl.Result{}, nil
	}

	routeStatus, err := r.statusProcessor.ProcessRouteStatus(ctx, httpRoute)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(routeStatus) > 0 {
		httpRoute.Status.Parents = routeStatus
		if err := r.fctx.Status().Update(ctx, httpRoute); err != nil {
			return ctrl.Result{}, err
		}
	}

	r.fctx.EventHandler.OnAdd(httpRoute)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *httpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.HTTPRoute{}).
		Complete(r)
}
