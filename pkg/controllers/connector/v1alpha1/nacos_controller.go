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

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrl "sigs.k8s.io/controller-runtime"

	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type nacosConnectorReconciler struct {
	connectorReconciler
}

// NewNacosConnectorReconciler returns a new reconciler for nacos connector resources
func NewNacosConnectorReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &nacosConnectorReconciler{
		connectorReconciler: connectorReconciler{
			recorder:           ctx.Manager.GetEventRecorderFor("nacos-connector"),
			fctx:               ctx,
			connectorAPIClient: connectorClientset.NewForConfigOrDie(ctx.KubeConfig),
		},
	}
}

// Reconcile reconciles a Gateway resource
func (r *nacosConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	connector := &connectorv1alpha1.NacosConnector{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		connector,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("NacosConnector resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get NacosConnector, %v", err)
		return ctrl.Result{}, err
	}

	if connector.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	mc := r.fctx.Config
	result, err := r.deployConnector(connector, mc)
	if err != nil || result.RequeueAfter > 0 || result.Requeue {
		return result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *nacosConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&connectorv1alpha1.NacosConnector{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			_, ok := obj.(*connectorv1alpha1.NacosConnector)
			if !ok {
				log.Error().Msgf("unexpected object type %T", obj)
			}
			return ok
		}))).
		Complete(r)
}
