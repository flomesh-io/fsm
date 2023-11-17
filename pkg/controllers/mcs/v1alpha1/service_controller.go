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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

// serviceReconciler reconciles a Service object
type serviceReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *serviceReconciler) NeedLeaderElection() bool {
	return true
}

// NewServiceReconciler returns a new Service.Reconciler
func NewServiceReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &serviceReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Service"),
		fctx:     ctx,
	}
}

func (r *serviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	svc := &corev1.Service{}
	if err := r.fctx.Get(ctx, req.NamespacedName, svc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// service is being deleted
	if svc.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	importName := serviceImportOwner(svc.OwnerReferences)
	// If ServiceImport name is empty, stands for it's not MCS or has not been linked to ServiceImport yet
	if importName == "" {
		return ctrl.Result{}, nil
	}

	svcImport := &mcsv1alpha1.ServiceImport{}
	if err := r.fctx.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: importName}, svcImport); err != nil {
		return ctrl.Result{}, err
	}

	if len(svcImport.Spec.IPs) > 0 {
		return ctrl.Result{}, nil
	}

	svcImport.Spec.IPs = []string{svc.Spec.ClusterIP}
	if err := r.fctx.Update(ctx, svcImport); err != nil {
		return ctrl.Result{}, err
	}
	log.Info().Msgf("Updated ServiceImport %s/%s, ClusterIP: %s", req.Namespace, importName, svc.Spec.ClusterIP)

	return ctrl.Result{}, nil
}

func serviceImportOwner(refs []metav1.OwnerReference) string {
	for _, ref := range refs {
		if ref.APIVersion == mcsv1alpha1.SchemeGroupVersion.String() && ref.Kind == serviceImportKind {
			return ref.Name
		}
	}
	return ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *serviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
