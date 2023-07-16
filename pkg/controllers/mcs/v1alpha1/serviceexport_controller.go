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
	"fmt"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"time"
)

// serviceExportReconciler reconciles a ServiceExport object
type serviceExportReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func NewServiceExportReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &serviceExportReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ServiceExport"),
		fctx:     ctx,
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the ServiceExport closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceExport object against the actual ServiceExport state, and then
// perform operations to make the ServiceExport state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *serviceExportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()
	//if !mc.IsControlPlane && mc.IsManaged {
	//    return ctrl.Result{}, nil
	//}

	export := &mcsv1alpha1.ServiceExport{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		export,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("[ServiceExport] ServiceExport resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get ServiceExport, %#v", err)
		return ctrl.Result{}, err
	}

	if export.Status.Conditions == nil {
		export.Status.Conditions = make([]metav1.Condition, 0)
	}

	svc := &corev1.Service{}
	if err := r.fctx.Get(ctx, req.NamespacedName, svc); err != nil {
		// the service doesn't exist
		if errors.IsNotFound(err) {
			return r.nonexistService(ctx, req, export)
		}

		return r.failedGetService(ctx, req, export, err)
	}

	// Found service

	// service is being deleted
	if svc.DeletionTimestamp != nil {
		return r.deletedService(ctx, req, export)
	}

	// ExternalName service cannot be exported
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		return r.unsupportedServiceType(ctx, req, export)
	}

	mc := r.fctx.Config
	if mc.IsIngressEnabled() {
		// Find and compare path from ingress
		ingList := &networkingv1.IngressList{}
		if err := r.fctx.List(ctx, ingList, client.InNamespace(corev1.NamespaceAll)); err != nil {
			return r.failedListIngresses(ctx, export, err)
		}
		for _, er := range export.Spec.Rules {
			for _, ing := range ingList.Items {
				// should not check against itself
				if metav1.IsControlledBy(&ing, export) {
					continue
				}
				for _, rule := range ing.Spec.Rules {
					for _, path := range rule.HTTP.Paths {
						if path.Path == er.Path && string(*path.PathType) == string(*er.PathType) {
							return r.pathConflicts(ctx, export, path, ing)
						}
					}
				}
			}
		}

		// create Ingress for the ServiceExport
		ing := &networkingv1.Ingress{}
		if err := r.fctx.Get(
			ctx,
			client.ObjectKey{
				Namespace: export.Namespace,
				Name:      fmt.Sprintf("svcexp-ing-%s", export.Name),
			},
			ing,
		); err != nil {
			if errors.IsNotFound(err) {
				// create new Ingress
				ing = newIngress(export)
				if err := ctrl.SetControllerReference(export, ing, r.fctx.Scheme); err != nil {
					return ctrl.Result{}, err
				}
				if err := r.fctx.Create(ctx, ing); err != nil {
					return ctrl.Result{}, err
				}

				return r.successExport(ctx, req, export)
			}

			return ctrl.Result{}, err
		}

		if export.Annotations == nil {
			export.Annotations = make(map[string]string)
		}
		oldHash := export.Annotations[constants.MultiClustersServiceExportHash]
		hash := utils.SimpleHash(export.Spec)

		// Changed, update ingress
		if oldHash != hash {
			// update export hash
			export.Annotations[constants.MultiClustersServiceExportHash] = hash
			if err := r.fctx.Update(ctx, export); err != nil {
				return ctrl.Result{}, err
			}

			// update ingress
			ing.Spec.Rules = []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: ingressPaths(export),
						},
					},
				},
			}

			if ing.Annotations == nil {
				ing.Annotations = ingressAnnotations(export)
			} else {
				for key, value := range ingressAnnotations(export) {
					ing.Annotations[key] = value
				}
			}

			if err := r.fctx.Update(ctx, ing); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return r.successExport(ctx, req, export)
}

func (r *serviceExportReconciler) nonexistService(ctx context.Context, req ctrl.Request, export *mcsv1alpha1.ServiceExport) (ctrl.Result, error) {
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("Service %s not found", req.NamespacedName),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *serviceExportReconciler) failedGetService(ctx context.Context, req ctrl.Request, export *mcsv1alpha1.ServiceExport, err error) (ctrl.Result, error) {
	// unknown errors
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("Get Service %s error: %s", req.NamespacedName, err),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	// stop processing
	return ctrl.Result{}, nil
}

func (r *serviceExportReconciler) deletedService(ctx context.Context, req ctrl.Request, export *mcsv1alpha1.ServiceExport) (ctrl.Result, error) {
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("Service %s is being deleted.", req.NamespacedName),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	// stop processing
	return ctrl.Result{}, nil
}

func (r *serviceExportReconciler) unsupportedServiceType(ctx context.Context, req ctrl.Request, export *mcsv1alpha1.ServiceExport) (ctrl.Result, error) {
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("Type of Service %s is %s, cannot be exported.", req.NamespacedName, corev1.ServiceTypeExternalName),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	// stop processing
	return ctrl.Result{}, nil
}

func (r *serviceExportReconciler) failedListIngresses(ctx context.Context, export *mcsv1alpha1.ServiceExport, err error) (ctrl.Result, error) {
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("Get Ingress List error: %s", err),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	// not processed successfully, requeue and retry it later
	return ctrl.Result{}, err
}

func (r *serviceExportReconciler) pathConflicts(ctx context.Context, export *mcsv1alpha1.ServiceExport, path networkingv1.HTTPIngressPath, ing networkingv1.Ingress) (ctrl.Result, error) {
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Failed",
		Message:            fmt.Sprintf("The path %q has been defined in Ingress %s/%s", path.Path, ing.Namespace, ing.Name),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	// stop processing, as the export failed due to path conflict
	return ctrl.Result{}, nil
}

func newIngress(export *mcsv1alpha1.ServiceExport) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   export.Namespace,
			Name:        fmt.Sprintf("svcexp-ing-%s", export.Name),
			Annotations: ingressAnnotations(export),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: pointer.String("pipy"),
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: ingressPaths(export),
						},
					},
				},
			},
		},
	}
}

func ingressAnnotations(export *mcsv1alpha1.ServiceExport) map[string]string {
	annos := make(map[string]string)

	if export.Spec.PathRewrite != nil {
		klog.V(5).Infof("PathRewrite=%#v", export.Spec.PathRewrite)
		if export.Spec.PathRewrite.From != "" && export.Spec.PathRewrite.To != "" {
			annos[constants.PipyIngressAnnotationRewriteFrom] = export.Spec.PathRewrite.From
			annos[constants.PipyIngressAnnotationRewriteTo] = export.Spec.PathRewrite.To
		}
	}

	if export.Spec.SessionSticky {
		annos[constants.PipyIngressAnnotationSessionSticky] = "true"
	}

	balancer := string(export.Spec.LoadBalancer)
	if balancer != "" {
		annos[constants.PipyIngressAnnotationLoadBalancer] = balancer
	}

	return annos
}

func ingressPaths(export *mcsv1alpha1.ServiceExport) []networkingv1.HTTPIngressPath {
	paths := make([]networkingv1.HTTPIngressPath, 0)
	for _, rule := range export.Spec.Rules {
		paths = append(paths, networkingv1.HTTPIngressPath{
			Path:     rule.Path,
			PathType: rule.PathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: export.Name,
					Port: networkingv1.ServiceBackendPort{
						Number: rule.PortNumber,
					},
				},
			},
		})
	}
	return paths
}

func (r *serviceExportReconciler) successExport(ctx context.Context, req ctrl.Request, export *mcsv1alpha1.ServiceExport) (ctrl.Result, error) {
	// service is exported successfully
	metautil.SetStatusCondition(&export.Status.Conditions, metav1.Condition{
		Type:               string(mcsv1alpha1.ServiceExportValid),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: export.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Success",
		Message:            fmt.Sprintf("Service %s is exported successfully.", req.NamespacedName),
	})

	if err := r.fctx.Status().Update(ctx, export); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *serviceExportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mc := r.fctx.Config

	if mc.IsIngressEnabled() {
		return ctrl.NewControllerManagedBy(mgr).
			For(&mcsv1alpha1.ServiceExport{}).
			Owns(&networkingv1.Ingress{}).
			Complete(r)
	}

	if mc.IsGatewayApiEnabled() {
		return ctrl.NewControllerManagedBy(mgr).
			For(&mcsv1alpha1.ServiceExport{}).
			Owns(&gwv1beta1.HTTPRoute{}).
			Complete(r)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&mcsv1alpha1.ServiceExport{}).
		Complete(r)
}
