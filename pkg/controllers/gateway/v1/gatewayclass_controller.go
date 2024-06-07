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
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

type gatewayClassReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *gatewayClassReconciler) NeedLeaderElection() bool {
	return true
}

// NewGatewayClassReconciler returns a new reconciler for GatewayClass
func NewGatewayClassReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &gatewayClassReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("GatewayClass"),
		fctx:     ctx,
	}
}

// Reconcile reconciles a GatewayClass object
func (r *gatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gatewayClass := &gwv1.GatewayClass{}
	if err := r.fctx.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		gatewayClass,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("GatewayClass resource not found. Ignoring since object must be deleted")
			r.fctx.GatewayEventHandler.OnDelete(&gwv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: req.Namespace,
					Name:      req.Name,
				}},
			)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get GatewayClass, %v", err)
		return ctrl.Result{}, err
	}

	if gatewayClass.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(gatewayClass)
		return ctrl.Result{}, nil
	}

	// ignore if the GatewayClass is not managed by the FSM GatewayController
	if gatewayClass.Spec.ControllerName != constants.GatewayController {
		return ctrl.Result{}, nil
	}

	r.fctx.StatusUpdater.Send(status.Update{
		Resource:       &gwv1.GatewayClass{},
		NamespacedName: client.ObjectKeyFromObject(gatewayClass),
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			class, ok := obj.(*gwv1.GatewayClass)
			if !ok {
				log.Error().Msgf("Unexpected object type %T", obj)
			}
			classCopy := class.DeepCopy()
			r.setAccepted(classCopy)

			return classCopy
		}),
	})

	return ctrl.Result{}, nil
}

func (r *gatewayClassReconciler) setAccepted(gatewayClass *gwv1.GatewayClass) {
	defer r.recorder.Eventf(gatewayClass, corev1.EventTypeNormal, "Accepted", "GatewayClass is accepted")

	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gwv1.GatewayClassConditionStatusAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayClassReasonAccepted),
		Message:            fmt.Sprintf("GatewayClass %q is accepted.", gatewayClass.Name),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	gwclsPrct := predicate.NewPredicateFuncs(func(object client.Object) bool {
		gatewayClass, ok := object.(*gwv1.GatewayClass)
		if !ok {
			log.Error().Msgf("unexpected object type: %T", object)
			return false
		}

		return gatewayClass.Spec.ControllerName == constants.GatewayController
	})

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.GatewayClass{}, builder.WithPredicates(gwclsPrct)).
		Complete(r); err != nil {
		return err
	}

	return addGatewayClassIndexers(context.Background(), mgr)
}

func addGatewayClassIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.GatewayClass{}, constants.ControllerGatewayClassIndex, func(obj client.Object) []string {
		cls := obj.(*gwv1.GatewayClass)
		return []string{string(cls.Spec.ControllerName)}
	}); err != nil {
		return err
	}

	return nil
}
