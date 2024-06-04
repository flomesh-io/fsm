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

	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/gateway/status/route"

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
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type tcpRouteReconciler struct {
	recorder        record.EventRecorder
	fctx            *fctx.ControllerContext
	statusProcessor *route.RouteStatusProcessor
}

func (r *tcpRouteReconciler) NeedLeaderElection() bool {
	return true
}

// NewTCPRouteReconciler returns a new TCPRoute Reconciler
func NewTCPRouteReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &tcpRouteReconciler{
		recorder:        ctx.Manager.GetEventRecorderFor("TCPRoute"),
		fctx:            ctx,
		statusProcessor: route.NewRouteStatusProcessor(ctx.Manager.GetCache()),
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

	//routeStatus, err := r.statusProcessor.Process(ctx, tcpRoute, nil)
	//if err != nil {
	//	return ctrl.Result{}, err
	//}
	//
	//if len(routeStatus) > 0 {
	//	r.fctx.StatusUpdater.Send(status.Update{
	//		Resource:       &gwv1alpha2.TCPRoute{},
	//		NamespacedName: client.ObjectKeyFromObject(tcpRoute),
	//		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
	//			tr, ok := obj.(*gwv1alpha2.TCPRoute)
	//			if !ok {
	//				log.Error().Msgf("Unexpected object type %T", obj)
	//			}
	//			trCopy := tr.DeepCopy()
	//			trCopy.Status.Parents = routeStatus
	//
	//			return trCopy
	//		}),
	//	})
	//}

	rsu := route.NewRouteStatusUpdate(
		tcpRoute,
		&tcpRoute.ObjectMeta,
		&tcpRoute.TypeMeta,
		nil,
		gwutils.ToSlicePtr(tcpRoute.Status.Parents),
	)
	if err := r.statusProcessor.Process(ctx, rsu, tcpRoute.Spec.ParentRefs); err != nil {
		return ctrl.Result{}, err
	}

	r.fctx.StatusUpdater.Send(status.Update{
		Resource:       &gwv1.HTTPRoute{},
		NamespacedName: client.ObjectKeyFromObject(tcpRoute),
		Mutator:        rsu,
	})

	r.fctx.GatewayEventHandler.OnAdd(tcpRoute, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *tcpRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1alpha2.TCPRoute{}).
		Complete(r); err != nil {
		return err
	}

	return addTCPRouteIndexers(context.Background(), mgr)
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
