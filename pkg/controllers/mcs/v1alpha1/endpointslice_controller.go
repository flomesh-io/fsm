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

package v1alpha1

import (
	"context"
	_ "embed"
	"github.com/flomesh-io/fsm/controllers"
	"github.com/flomesh-io/fsm/pkg/commons"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// endpointSliceReconciler reconciles an EndpointSlice object
type endpointSliceReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.FsmContext
}

func NewEndpointSliceReconciler(ctx *fctx.FsmContext) controllers.Reconciler {
	return &endpointSliceReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("EndpointSlice"),
		fctx:     ctx,
	}
}

func (r *endpointSliceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	epSlice := &discoveryv1.EndpointSlice{}
	if err := r.fctx.Get(ctx, req.NamespacedName, epSlice); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if shouldIgnoreEndpointSlice(epSlice) {
		return ctrl.Result{}, nil
	}

	// Ensure the EndpointSlice is labelled to match the ServiceImport's derived
	// Service.
	serviceName := derivedName(types.NamespacedName{Namespace: epSlice.Namespace, Name: epSlice.Labels[commons.MultiClusterLabelServiceName]})
	if epSlice.Labels[discoveryv1.LabelServiceName] == serviceName {
		return ctrl.Result{}, nil
	}

	epSlice.Labels[discoveryv1.LabelServiceName] = serviceName
	epSlice.Labels[commons.MultiClusterLabelServiceName] = serviceName
	if err := r.fctx.Update(ctx, epSlice); err != nil {
		return ctrl.Result{}, err
	}

	klog.Info("added label", discoveryv1.LabelServiceName, serviceName)

	return ctrl.Result{}, nil
}

func shouldIgnoreEndpointSlice(epSlice *discoveryv1.EndpointSlice) bool {
	if epSlice.DeletionTimestamp != nil {
		return true
	}

	if epSlice.Labels[commons.MultiClusterLabelServiceName] == "" {
		return true
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *endpointSliceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&discoveryv1.EndpointSlice{}).
		Complete(r)
}
