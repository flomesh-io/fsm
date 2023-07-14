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

package flb

import (
	"context"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func IsFlbEnabled(svc *corev1.Service, api *kube.K8sAPI) bool {
	if svc == nil {
		return false
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	// if service doesn't have flb.flomesh.io/enabled annotation
	if svc.Annotations == nil || svc.Annotations[constants.FlbEnabledAnnotation] == "" {
		// check ns annotation
		ns, err := api.Client.CoreV1().
			Namespaces().
			Get(context.TODO(), svc.Namespace, metav1.GetOptions{})

		if err != nil {
			klog.Errorf("Failed to get namespace %q: %s", svc.Namespace, err)
			return false
		}

		if ns.Annotations == nil || ns.Annotations[constants.FlbEnabledAnnotation] == "" {
			return false
		}

		klog.V(5).Infof("Found annotation %q on Namespace %q", constants.FlbEnabledAnnotation, ns.Name)
		return utils.ParseEnabled(ns.Annotations[constants.FlbEnabledAnnotation])
	}

	// parse svc annotation
	klog.V(5).Infof("Found annotation %q on Service %s/%s", constants.FlbEnabledAnnotation, svc.Namespace, svc.Name)
	return utils.ParseEnabled(svc.Annotations[constants.FlbEnabledAnnotation])
}
