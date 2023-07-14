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

package servicelb

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/sidecar/driver"
	"github.com/flomesh-io/fsm/pkg/utils"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/net"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sort"
	"strconv"
	"strings"
)

// ServiceReconciler reconciles a Service object
type serviceReconciler struct {
	recorder record.EventRecorder
	fctx     *driver.ControllerContext
	cfg      configurator.Configurator
	client.Client
}

func NewServiceReconciler(ctx *driver.ControllerContext) controllers.Reconciler {
	return &serviceReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("ServiceLB"),
		fctx:     ctx,
		Client:   ctx.Manager.GetClient(),
		cfg:      ctx.Configurator,
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *serviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Service instance
	svc := &corev1.Service{}
	if err := r.Get(
		ctx,
		req.NamespacedName,
		svc,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("Service resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Service, %v", err)
		return ctrl.Result{}, err
	}

	if err := r.deployDaemonSet(ctx, svc, r.cfg); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateService(ctx, svc, r.cfg); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *serviceReconciler) deployDaemonSet(ctx context.Context, svc *corev1.Service, mc configurator.Configurator) error {
	klog.V(5).Infof("Going to deploy DaemonSet ...")

	if !mc.IsServiceLBEnabled() || svc.DeletionTimestamp != nil || svc.Spec.Type != corev1.ServiceTypeLoadBalancer || svc.Spec.ClusterIP == "" || svc.Spec.ClusterIP == "None" {
		klog.V(5).Infof("Deleting DaemonSet ...")
		return r.deleteDaemonSet(ctx, svc)
	}

	ds, err := r.newDaemonSet(ctx, svc, mc)
	if err != nil {
		return err
	}

	if ds != nil {
		klog.V(5).Infof("Setting controller reference, Owner Service[%s/%s], DaemonSet[%s/%s] ...", svc.Namespace, svc.Namespace, ds.Namespace, ds.Name)
		if err := ctrl.SetControllerReference(svc, ds, r.fctx.Scheme); err != nil {
			return err
		}

		klog.V(5).Infof("Creating/updating DaemonSet[%s/%s] ...", ds.Namespace, ds.Name)
		result, err := utils.CreateOrUpdate(ctx, r.Client, ds)
		if err != nil {
			return err
		}

		switch result {
		case controllerutil.OperationResultCreated, controllerutil.OperationResultUpdated:
			defer r.recorder.Eventf(svc, corev1.EventTypeNormal, "AppliedDaemonSet", "Applied LoadBalancer DaemonSet %s/%s", ds.Namespace, ds.Name)
		}
	}

	return nil
}

func (r *serviceReconciler) deleteDaemonSet(ctx context.Context, svc *corev1.Service) error {
	name := generateName(svc)
	ds := &appv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: svc.Namespace,
			Name:      name,
		},
	}

	if err := r.Delete(ctx, ds); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	defer r.recorder.Eventf(svc, corev1.EventTypeNormal, "DeletedDaemonSet", "Deleted LoadBalancer DaemonSet %s/%s", svc.Namespace, name)
	return nil
}

func (r *serviceReconciler) newDaemonSet(ctx context.Context, svc *corev1.Service, mc configurator.Configurator) (*appv1.DaemonSet, error) {
	klog.V(5).Infof("Creating a new DaemonSet template ...")

	name := generateName(svc)
	intOne := intstr.FromInt(1)

	ds := &appv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: svc.Namespace,
			Labels: map[string]string{
				nodeSelectorLabel: "false",
				svcNameLabel:      svc.Name,
				svcNamespaceLabel: svc.Namespace,
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		Spec: appv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":             name,
						svcNameLabel:      svc.Name,
						svcNamespaceLabel: svc.Namespace,
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: pointer.Bool(false),
				},
			},
			UpdateStrategy: appv1.DaemonSetUpdateStrategy{
				Type: appv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appv1.RollingUpdateDaemonSet{
					MaxUnavailable: &intOne,
				},
			},
		},
	}

	for _, port := range svc.Spec.Ports {
		portName := fmt.Sprintf("lb-%s-%d", strings.ToLower(string(port.Protocol)), port.Port)
		container := corev1.Container{
			Name:            portName,
			Image:           mc.ServiceLbImage(),
			ImagePullPolicy: utils.ImagePullPolicyByTag(mc.ServiceLbImage()),
			Ports: []corev1.ContainerPort{
				{
					Name:          portName,
					ContainerPort: port.Port,
					HostPort:      port.Port,
					Protocol:      port.Protocol,
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "SRC_PORT",
					Value: strconv.Itoa(int(port.Port)),
				},
				{
					Name:  "DEST_PROTO",
					Value: string(port.Protocol),
				},
				{
					Name:  "DEST_PORT",
					Value: strconv.Itoa(int(port.Port)),
				},
				{
					Name:  "DEST_IPS",
					Value: strings.Join(svc.Spec.ClusterIPs, " "),
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"NET_ADMIN",
					},
				},
			},
		}

		ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, container)
	}

	ds.Spec.Template.Spec.Tolerations = append(ds.Spec.Template.Spec.Tolerations, []corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/master",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
		{
			Key:      "node-role.kubernetes.io/control-plane",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
		{
			Key:      "CriticalAddonsOnly",
			Operator: "Exists",
		},
	}...)

	nodesWithLabel := &corev1.NodeList{}
	if err := r.List(
		ctx,
		nodesWithLabel,
		client.InNamespace(corev1.NamespaceAll),
		client.MatchingLabels{
			daemonsetNodeLabel: "false",
		},
	); err != nil {
		return nil, err
	}

	if len(nodesWithLabel.Items) > 0 {
		ds.Spec.Template.Spec.NodeSelector = map[string]string{
			daemonsetNodeLabel: "true",
		}
		if svc.Labels[daemonsetNodePoolLabel] != "" {
			ds.Spec.Template.Spec.NodeSelector[daemonsetNodePoolLabel] = svc.Labels[daemonsetNodePoolLabel]
		}
		ds.Labels[nodeSelectorLabel] = "true"
	}

	return ds, nil
}

func (r *serviceReconciler) updateService(ctx context.Context, svc *corev1.Service, mc configurator.Configurator) error {
	if !mc.IsServiceLBEnabled() || svc.DeletionTimestamp != nil || svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return r.removeFinalizer(ctx, svc)
	}

	pods := &corev1.PodList{}
	if err := r.List(
		ctx,
		pods,
		client.InNamespace(svc.Namespace),
		client.MatchingLabels{
			svcNameLabel:      svc.Name,
			svcNamespaceLabel: svc.Namespace,
		},
	); err != nil {
		return err
	}

	existingIPs := serviceIPs(svc)
	expectedIPs, err := r.podIPs(ctx, pods.Items, svc)
	if err != nil {
		return err
	}

	sort.Strings(expectedIPs)
	sort.Strings(existingIPs)

	if utils.StringsEqual(expectedIPs, existingIPs) {
		return nil
	}

	svc = svc.DeepCopy()
	if err = r.addFinalizer(ctx, svc); err != nil {
		return err
	}

	svc.Status.LoadBalancer.Ingress = nil
	for _, ip := range expectedIPs {
		svc.Status.LoadBalancer.Ingress = append(svc.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})
	}

	defer r.recorder.Eventf(svc, corev1.EventTypeNormal, "UpdatedIngressIP", "LoadBalancer Ingress IP addresses updated: %s", strings.Join(expectedIPs, ", "))

	return r.Status().Update(ctx, svc)
}

// serviceIPs returns the list of ingress IP addresses from the Service
func serviceIPs(svc *corev1.Service) []string {
	var ips []string

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		}
	}

	return ips
}

func (r *serviceReconciler) podIPs(ctx context.Context, pods []corev1.Pod, svc *corev1.Service) ([]string, error) {
	extIPs := map[string]bool{}
	intIPs := map[string]bool{}

	for _, pod := range pods {
		if pod.Spec.NodeName == "" || pod.Status.PodIP == "" {
			continue
		}

		if !isPodStatusConditionTrue(pod.Status.Conditions, corev1.PodReady) {
			continue
		}

		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, node); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return nil, err
		}

		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeExternalIP {
				extIPs[addr.Address] = true
			} else if addr.Type == corev1.NodeInternalIP {
				intIPs[addr.Address] = true
			}
		}
	}

	keys := func(addrs map[string]bool) (ips []string) {
		for k := range addrs {
			ips = append(ips, k)
		}
		return ips
	}

	var ips []string
	if len(extIPs) > 0 {
		ips = keys(extIPs)
	} else {
		ips = keys(intIPs)
	}

	ips, err := filterByIPFamily(ips, svc)
	if err != nil {
		return nil, err
	}

	//if len(ips) > 0 && h.rootless {
	//    return []string{"127.0.0.1"}, nil
	//}

	return ips, nil
}

func filterByIPFamily(ips []string, svc *corev1.Service) ([]string, error) {
	var ipFamilyPolicy corev1.IPFamilyPolicyType
	var ipv4Addresses []string

	for _, ip := range ips {
		if net.IsIPv4String(ip) {
			ipv4Addresses = append(ipv4Addresses, ip)
		}
	}

	if svc.Spec.IPFamilyPolicy != nil {
		ipFamilyPolicy = *svc.Spec.IPFamilyPolicy
	}

	switch ipFamilyPolicy {
	case corev1.IPFamilyPolicySingleStack:
		if svc.Spec.IPFamilies[0] == corev1.IPv4Protocol {
			return ipv4Addresses, nil
		}
	}

	return nil, fmt.Errorf("unhandled ipFamilyPolicy")
}

func (r *serviceReconciler) addFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		svc.Finalizers = append(svc.Finalizers, finalizerName)
		return r.Update(ctx, svc)
	}

	return nil
}

func (r *serviceReconciler) removeFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		return nil
	}

	for k, v := range svc.Finalizers {
		if v != finalizerName {
			continue
		}
		svc.Finalizers = append(svc.Finalizers[:k], svc.Finalizers[k+1:]...)
	}

	return r.Update(ctx, svc)
}

func (r *serviceReconciler) hasFinalizer(ctx context.Context, svc *corev1.Service) bool {
	for _, finalizer := range svc.Finalizers {
		if finalizer == finalizerName {
			return true
		}
	}

	return false
}

func generateName(svc *corev1.Service) string {
	return fmt.Sprintf("svclb-%s-%s", svc.Name, svc.UID[:8])
}

func isPodStatusConditionTrue(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) bool {
	return isPodStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionTrue)
}

func isPodStatusConditionPresentAndEqual(conditions []corev1.PodCondition, conditionType corev1.PodConditionType, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *serviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Endpoints{}).
		Owns(&appv1.DaemonSet{}).
		Complete(r)
}
