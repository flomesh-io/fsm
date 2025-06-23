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

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type machineConnectorReconciler struct {
	connectorReconciler
}

// NewMachineConnectorReconciler returns a new reconciler for machine connector resources
func NewMachineConnectorReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &machineConnectorReconciler{
		connectorReconciler: connectorReconciler{
			recorder:           ctx.Manager.GetEventRecorderFor("machine-connector"),
			fctx:               ctx,
			connectorAPIClient: connectorClientset.NewForConfigOrDie(ctx.KubeConfig),
		},
	}
}

// Reconcile reconciles a Gateway resource
func (r *machineConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	connector := &ctv1.MachineConnector{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		connector,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.removeDeployment(string(ctv1.MachineDiscoveryService), req.Namespace, req.Name)
			log.Info().Msgf("MachineConnector resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get MachineConnector, %v", err)
		return ctrl.Result{}, err
	}

	if connector.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if !r.hasDeployment(string(ctv1.MachineDiscoveryService), req.Namespace, req.Name) {
		mc := r.fctx.Configurator
		result, err := r.deployConnector(connector, mc)
		if err != nil || result.RequeueAfter > 0 || result.Requeue {
			return result, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *machineConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ctv1.MachineConnector{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			_, ok := obj.(*ctv1.MachineConnector)
			if !ok {
				log.Error().Msgf("unexpected object type %T", obj)
			}
			return ok
		}))).
		Complete(r)
}
