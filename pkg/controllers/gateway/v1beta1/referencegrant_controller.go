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

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/flomesh-io/fsm/pkg/logger"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/constants"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

var (
	log = logger.NewPretty("gatewayapi-controller/v1beta1")
)

type referenceGrantReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *referenceGrantReconciler) NeedLeaderElection() bool {
	return true
}

// NewReferenceGrantReconciler returns a new ReferenceGrant Reconciler
func NewReferenceGrantReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &referenceGrantReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ReferenceGrant"),
		fctx:     ctx,
	}
}

// Reconcile reads that state of the cluster for a ReferenceGrant object and makes changes based on the state read
func (r *referenceGrantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	referenceGrant := &gwv1beta1.ReferenceGrant{}
	err := r.fctx.Get(ctx, req.NamespacedName, referenceGrant)
	if errors.IsNotFound(err) {
		r.fctx.GatewayEventHandler.OnDelete(&gwv1beta1.ReferenceGrant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			}})
		return reconcile.Result{}, nil
	}

	if referenceGrant.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(referenceGrant)
		return ctrl.Result{}, nil
	}

	// As ReferenceGrant has no status, we don't need to update it

	r.fctx.GatewayEventHandler.OnAdd(referenceGrant, false)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *referenceGrantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1beta1.ReferenceGrant{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.secretToRefGrants)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.configMapToRefGrants)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.serviceToRefGrants)).
		Complete(r); err != nil {
		return err
	}

	return addReferenceGrantIndexers(context.Background(), mgr)
}

func (r *referenceGrantReconciler) secretToRefGrants(ctx context.Context, object client.Object) []reconcile.Request {
	secret := object.(*corev1.Secret)

	list := &gwv1beta1.ReferenceGrantList{}
	err := r.fctx.Manager.GetCache().List(ctx, list, client.InNamespace(secret.Namespace))
	if err != nil {
		return nil
	}

	var refGrants []reconcile.Request
	for _, refGrant := range list.Items {
		for _, target := range refGrant.Spec.To {
			if target.Group != corev1.GroupName {
				continue
			}

			if target.Kind != constants.KubernetesSecretKind {
				continue
			}

			if target.Name == nil || string(*target.Name) == secret.Name {
				refGrants = append(refGrants, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: refGrant.Namespace,
						Name:      refGrant.Name,
					},
				})

				break
			}
		}
	}

	return refGrants
}

func (r *referenceGrantReconciler) configMapToRefGrants(ctx context.Context, object client.Object) []reconcile.Request {
	cm := object.(*corev1.ConfigMap)

	list := &gwv1beta1.ReferenceGrantList{}
	err := r.fctx.Manager.GetCache().List(ctx, list, client.InNamespace(cm.Namespace))
	if err != nil {
		return nil
	}

	var refGrants []reconcile.Request
	for _, refGrant := range list.Items {
		for _, target := range refGrant.Spec.To {
			if target.Group != corev1.GroupName {
				continue
			}

			if target.Kind != constants.KubernetesConfigMapKind {
				continue
			}

			if target.Name == nil || string(*target.Name) == cm.Name {
				refGrants = append(refGrants, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: refGrant.Namespace,
						Name:      refGrant.Name,
					},
				})

				break
			}
		}
	}

	return refGrants
}

func (r *referenceGrantReconciler) serviceToRefGrants(ctx context.Context, object client.Object) []reconcile.Request {
	service := object.(*corev1.Service)

	list := &gwv1beta1.ReferenceGrantList{}
	err := r.fctx.Manager.GetCache().List(ctx, list, client.InNamespace(service.Namespace))
	if err != nil {
		return nil
	}

	var refGrants []reconcile.Request
	for _, refGrant := range list.Items {
		for _, target := range refGrant.Spec.To {
			if target.Group != corev1.GroupName {
				continue
			}

			if target.Kind != constants.KubernetesServiceKind {
				continue
			}

			if target.Name == nil || string(*target.Name) == service.Name {
				refGrants = append(refGrants, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: refGrant.Namespace,
						Name:      refGrant.Name,
					},
				})

				break
			}
		}
	}

	return refGrants
}

func addReferenceGrantIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1beta1.ReferenceGrant{}, constants.TargetKindRefGrantIndex, func(obj client.Object) []string {
		refGrant := obj.(*gwv1beta1.ReferenceGrant)

		var referredResources []string
		for _, target := range refGrant.Spec.To {
			referredResources = append(referredResources, string(target.Kind))
		}

		return referredResources
	}); err != nil {
		return err
	}

	return nil
}
