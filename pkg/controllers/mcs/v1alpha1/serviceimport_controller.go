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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

// serviceImportReconciler reconciles a ServiceImport object
type serviceImportReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

// NewServiceImportReconciler returns a new ServiceImport.Reconciler
func NewServiceImportReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &serviceImportReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ServiceImport"),
		fctx:     ctx,
	}
}

func (r *serviceImportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	svcImport := &mcsv1alpha1.ServiceImport{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		svcImport,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("[ServiceImport] ServiceImport resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get ServiceImport, %v", err)
		return ctrl.Result{}, err
	}

	if shouldIgnoreImport(svcImport) {
		return ctrl.Result{}, nil
	}

	// Ensure the existence of the derived service
	if svcImport.Annotations[constants.MultiClusterDerivedServiceAnnotation] == "" {
		if svcImport.Annotations == nil {
			svcImport.Annotations = make(map[string]string)
		}

		svcImport.Annotations[constants.MultiClusterDerivedServiceAnnotation] = req.Name
		if err := r.fctx.Update(ctx, svcImport); err != nil {
			return ctrl.Result{}, err
		}
		log.Info().Msgf("Added annotation %s=%s", constants.MultiClusterDerivedServiceAnnotation, req.Name)

		return ctrl.Result{}, nil
	}

	svc, err := r.upsertDerivedService(ctx, svcImport)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(svcImport.Spec.IPs) == 0 {
		return ctrl.Result{}, nil
	}

	// update LoadBalancer status with provided ClusterSetIPs
	ingress := make([]corev1.LoadBalancerIngress, 0)
	for _, ip := range svcImport.Spec.IPs {
		ingress = append(ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})
	}

	svc.Status = corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: ingress,
		},
	}

	if err := r.fctx.Status().Update(ctx, svc); err != nil {
		return ctrl.Result{}, err
	}

	// TODO: create/update/delete EndpointSlice and add required annotations/labels
	//for _, p := range svcImport.Spec.Ports {
	//    for _, ep := range p.Endpoints {
	//        ep.Target.
	//    }
	//}

	return ctrl.Result{}, nil
}

func (r *serviceImportReconciler) upsertDerivedService(ctx context.Context, svcImport *mcsv1alpha1.ServiceImport) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: svcImport.Namespace,
			Name:      svcImport.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       svcImport.Name,
					Kind:       serviceImportKind,
					APIVersion: mcsv1alpha1.SchemeGroupVersion.String(),
					UID:        svcImport.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeClusterIP,
			Ports: servicePorts(svcImport),
		},
	}

	// just create to avoid concurrent write
	if err := r.fctx.Create(ctx, svc); err != nil {
		if errors.IsAlreadyExists(err) {
			if err = r.fctx.Get(
				ctx,
				types.NamespacedName{Namespace: svcImport.Namespace, Name: svcImport.Name},
				svc,
			); err != nil {
				return nil, err
			}

			if isAlreadyOwnerOfService(svcImport, svc.OwnerReferences) {
				return svc, nil
			}

			if err = controllerutil.SetOwnerReference(svcImport, svc, r.fctx.Scheme); err != nil {
				return nil, err
			}

			if err = r.fctx.Update(ctx, svc); err != nil {
				return nil, err
			}

			return svc, nil
		}

		return nil, err
	}

	log.Info().Msgf("Created service %s/%s", svc.Namespace, svc.Name)

	return svc, nil
}

func servicePorts(svcImport *mcsv1alpha1.ServiceImport) []corev1.ServicePort {
	ports := make([]corev1.ServicePort, len(svcImport.Spec.Ports))
	for i, p := range svcImport.Spec.Ports {
		ports[i] = corev1.ServicePort{
			Name:        p.Name,
			Protocol:    p.Protocol,
			Port:        p.Port,
			AppProtocol: p.AppProtocol,
		}
	}
	return ports
}

func shouldIgnoreImport(svcImport *mcsv1alpha1.ServiceImport) bool {
	if svcImport.DeletionTimestamp != nil {
		return true
	}

	if svcImport.Spec.Type != mcsv1alpha1.ClusterSetIP {
		return true
	}

	return false
}

func isAlreadyOwnerOfService(svcImport *mcsv1alpha1.ServiceImport, svcOwnerRefs []metav1.OwnerReference) bool {
	for _, ref := range svcOwnerRefs {
		if ref.APIVersion == mcsv1alpha1.SchemeGroupVersion.String() && ref.Kind == svcImport.Kind {
			return ref.Name == svcImport.Name
		}
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *serviceImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcsv1alpha1.ServiceImport{}).
		Complete(r)
}
