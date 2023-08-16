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
	"fmt"
	"sort"
	"time"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/apis/gateway"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/gateway/utils"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type gatewayClassReconciler struct {
	recorder         record.EventRecorder
	fctx             *fctx.ControllerContext
	gatewayAPIClient gwclient.Interface
}

// NewGatewayClassReconciler returns a new reconciler for GatewayClass
func NewGatewayClassReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &gatewayClassReconciler{
		recorder:         ctx.Manager.GetEventRecorderFor("GatewayClass"),
		fctx:             ctx,
		gatewayAPIClient: gatewayApiClientset.NewForConfigOrDie(ctx.KubeConfig),
	}
}

// Reconcile reconciles a GatewayClass object
func (r *gatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gatewayClass := &gwv1beta1.GatewayClass{}
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
			r.fctx.EventHandler.OnDelete(&gwv1beta1.GatewayClass{
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
		r.fctx.EventHandler.OnDelete(gatewayClass)
		return ctrl.Result{}, nil
	}

	// Accept all GatewayClasses those ControllerName is flomesh.io/gateway-controller
	r.setAcceptedStatus(gatewayClass)
	result, err := r.updateStatus(ctx, gatewayClass, gwv1beta1.GatewayClassConditionStatusAccepted)
	if err != nil {
		return result, err
	}

	gatewayClassList, err := r.gatewayAPIClient.GatewayV1beta1().
		GatewayClasses().
		List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error().Msgf("failed list gatewayclasses: %s", err)
		return ctrl.Result{}, err
	}

	// If there's multiple GatewayClasses, the oldest is set to active and the rest are set to inactive
	for _, class := range r.setActiveStatus(gatewayClassList) {
		result, err := r.updateStatus(ctx, class, gateway.GatewayClassConditionStatusActive)
		if err != nil {
			return result, err
		}
	}

	// As status of all GatewayClasses have been updated, just send the event
	r.fctx.EventHandler.OnAdd(&gwv1beta1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      req.Name,
		}},
	)

	return ctrl.Result{}, nil
}

func (r *gatewayClassReconciler) updateStatus(ctx context.Context, class *gwv1beta1.GatewayClass, status gwv1beta1.GatewayClassConditionType) (ctrl.Result, error) {
	if err := r.fctx.Status().Update(ctx, class); err != nil {
		//defer r.recorder.Eventf(class, corev1.EventTypeWarning, "UpdateStatus", "Failed to update status of GatewayClass: %s", err)
		return ctrl.Result{}, err
	}

	switch status {
	case gwv1beta1.GatewayClassConditionStatusAccepted:
		if utils.IsAcceptedGatewayClass(class) {
			defer r.recorder.Eventf(class, corev1.EventTypeNormal, "Accepted", "GatewayClass is accepted")
		} else {
			defer r.recorder.Eventf(class, corev1.EventTypeNormal, "Rejected", "GatewayClass is rejected")
		}
	case gateway.GatewayClassConditionStatusActive:
		if utils.IsActiveGatewayClass(class) {
			defer r.recorder.Eventf(class, corev1.EventTypeNormal, "Active", "GatewayClass is set to active")
		} else {
			defer r.recorder.Eventf(class, corev1.EventTypeNormal, "Inactive", "GatewayClass is set to inactive")
		}
	}

	return ctrl.Result{}, nil
}

func (r *gatewayClassReconciler) setAcceptedStatus(gatewayClass *gwv1beta1.GatewayClass) {
	if gatewayClass.Spec.ControllerName == constants.GatewayController {
		r.setAccepted(gatewayClass)
	} else {
		r.setRejected(gatewayClass)
	}
}

func (r *gatewayClassReconciler) setActiveStatus(list *gwv1beta1.GatewayClassList) []*gwv1beta1.GatewayClass {
	acceptedClasses := make([]*gwv1beta1.GatewayClass, 0)
	for _, class := range list.Items {
		class := class // fix lint GO-LOOP-REF
		if utils.IsAcceptedGatewayClass(&class) {
			acceptedClasses = append(acceptedClasses, &class)
		}
	}

	sort.Slice(acceptedClasses, func(i, j int) bool {
		if acceptedClasses[i].CreationTimestamp.Time.Equal(acceptedClasses[j].CreationTimestamp.Time) {
			return acceptedClasses[i].Name < acceptedClasses[j].Name
		}

		return acceptedClasses[i].CreationTimestamp.Time.Before(acceptedClasses[j].CreationTimestamp.Time)
	})

	statusChangedClasses := make([]*gwv1beta1.GatewayClass, 0)
	for i, class := range acceptedClasses {
		// ONLY the oldest GatewayClass is active
		if i == 0 {
			if !utils.IsActiveGatewayClass(class) {
				r.setActive(acceptedClasses[i])
				statusChangedClasses = append(statusChangedClasses, acceptedClasses[i])
			}
			continue
		}

		if utils.IsActiveGatewayClass(class) {
			r.setInactive(acceptedClasses[i])
			statusChangedClasses = append(statusChangedClasses, acceptedClasses[i])
		}
	}

	return statusChangedClasses
}

func (r *gatewayClassReconciler) setRejected(gatewayClass *gwv1beta1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gwv1beta1.GatewayClassConditionStatusAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Rejected",
		Message:            fmt.Sprintf("GatewayClass %q is rejected as ControllerName %q is not supported.", gatewayClass.Name, gatewayClass.Spec.ControllerName),
	})
}

func (r *gatewayClassReconciler) setAccepted(gatewayClass *gwv1beta1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gwv1beta1.GatewayClassConditionStatusAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1beta1.GatewayClassReasonAccepted),
		Message:            fmt.Sprintf("GatewayClass %q is accepted.", gatewayClass.Name),
	})
}

func (r *gatewayClassReconciler) setActive(gatewayClass *gwv1beta1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gateway.GatewayClassConditionStatusActive),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gateway.GatewayClassReasonActive),
		Message:            fmt.Sprintf("GatewayClass %q is set to active.", gatewayClass.Name),
	})
}
func (r *gatewayClassReconciler) setInactive(gatewayClass *gwv1beta1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gateway.GatewayClassConditionStatusActive),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gateway.GatewayClassReasonInactive),
		Message:            fmt.Sprintf("GatewayClass %q is inactive as there's already an active GatewayClass.", gatewayClass.Name),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	gwclsPrct := predicate.NewPredicateFuncs(func(object client.Object) bool {
		gatewayClass, ok := object.(*gwv1beta1.GatewayClass)
		if !ok {
			log.Error().Msgf("unexpected object type: %T", object)
			return false
		}

		return gatewayClass.Spec.ControllerName == constants.GatewayController
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.GatewayClass{}, builder.WithPredicates(gwclsPrct)).
		Complete(r)
}
