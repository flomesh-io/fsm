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
	"sort"
	"time"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

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
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

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

func (r *gatewayClassReconciler) NeedLeaderElection() bool {
	return true
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

	r.fctx.StatusUpdater.Send(status.Update{
		Resource:       gatewayClass,
		NamespacedName: client.ObjectKeyFromObject(gatewayClass),
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			class, ok := obj.(*gwv1.GatewayClass)
			if !ok {
				log.Error().Msgf("unsupported object type %T", obj)
			}
			classCopy := class.DeepCopy()
			r.setAcceptedStatus(classCopy)

			return classCopy
		}),
	})

	gatewayClassList, err := r.gatewayAPIClient.GatewayV1().
		GatewayClasses().
		List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error().Msgf("failed list gatewayclasses: %s", err)
		return ctrl.Result{}, err
	}

	// If there's multiple GatewayClasses whose ControllerName is flomesh.io/gateway-controller, the oldest is set to active and the rest are set to inactive
	r.updateActiveStatus(gatewayClassList)

	// As status of all GatewayClasses have been updated, just send the event
	r.fctx.GatewayEventHandler.OnAdd(&gwv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      req.Name,
		}},
		false,
	)

	return ctrl.Result{}, nil
}

func (r *gatewayClassReconciler) setAcceptedStatus(gatewayClass *gwv1.GatewayClass) {
	if gatewayClass.Spec.ControllerName == constants.GatewayController {
		r.setAccepted(gatewayClass)
	}
}

func (r *gatewayClassReconciler) updateActiveStatus(list *gwv1.GatewayClassList) {
	acceptedClasses := make([]*gwv1.GatewayClass, 0)
	for _, class := range list.Items {
		class := class // fix lint GO-LOOP-REF
		if class.Spec.ControllerName == constants.GatewayController && utils.IsAcceptedGatewayClass(&class) {
			acceptedClasses = append(acceptedClasses, &class)
		}
	}

	sort.Slice(acceptedClasses, func(i, j int) bool {
		if acceptedClasses[i].CreationTimestamp.Time.Equal(acceptedClasses[j].CreationTimestamp.Time) {
			return acceptedClasses[i].Name < acceptedClasses[j].Name
		}

		return acceptedClasses[i].CreationTimestamp.Time.Before(acceptedClasses[j].CreationTimestamp.Time)
	})

	//statusChangedClasses := make([]*gwv1.GatewayClass, 0)
	for i, class := range acceptedClasses {
		// ONLY the oldest GatewayClass is active
		if i == 0 {
			if !utils.IsActiveGatewayClass(class) {
				//r.setActive(acceptedClasses[i])
				//statusChangedClasses = append(statusChangedClasses, acceptedClasses[i])
				r.fctx.StatusUpdater.Send(status.Update{
					Resource:       class,
					NamespacedName: client.ObjectKeyFromObject(class),
					Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
						clazz, ok := obj.(*gwv1.GatewayClass)
						if !ok {
							log.Error().Msgf("unsupported object type %T", obj)
						}
						classCopy := clazz.DeepCopy()
						r.setActive(classCopy)

						return classCopy
					}),
				})
			}
			continue
		}

		if utils.IsActiveGatewayClass(class) {
			//r.setInactive(acceptedClasses[i])
			//statusChangedClasses = append(statusChangedClasses, acceptedClasses[i])
			r.fctx.StatusUpdater.Send(status.Update{
				Resource:       class,
				NamespacedName: client.ObjectKeyFromObject(class),
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					clazz, ok := obj.(*gwv1.GatewayClass)
					if !ok {
						log.Error().Msgf("unsupported object type %T", obj)
					}
					classCopy := clazz.DeepCopy()
					r.setInactive(classCopy)

					return classCopy
				}),
			})
		}
	}

	//return statusChangedClasses
}

func (r *gatewayClassReconciler) setAccepted(gatewayClass *gwv1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gwv1.GatewayClassConditionStatusAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayClassReasonAccepted),
		Message:            fmt.Sprintf("GatewayClass %q is accepted.", gatewayClass.Name),
	})
	defer r.recorder.Eventf(gatewayClass, corev1.EventTypeNormal, "Accepted", "GatewayClass is accepted")
}

func (r *gatewayClassReconciler) setActive(gatewayClass *gwv1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gateway.GatewayClassConditionStatusActive),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gateway.GatewayClassReasonActive),
		Message:            fmt.Sprintf("GatewayClass %q is set to active.", gatewayClass.Name),
	})
	defer r.recorder.Eventf(gatewayClass, corev1.EventTypeNormal, "Active", "GatewayClass is set to active")
}
func (r *gatewayClassReconciler) setInactive(gatewayClass *gwv1.GatewayClass) {
	metautil.SetStatusCondition(&gatewayClass.Status.Conditions, metav1.Condition{
		Type:               string(gateway.GatewayClassConditionStatusActive),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gatewayClass.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gateway.GatewayClassReasonInactive),
		Message:            fmt.Sprintf("GatewayClass %q is inactive as there's already an active GatewayClass.", gatewayClass.Name),
	})
	defer r.recorder.Eventf(gatewayClass, corev1.EventTypeNormal, "Inactive", "GatewayClass is set to inactive")
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.GatewayClass{}, builder.WithPredicates(gwclsPrct)).
		Complete(r)
}
