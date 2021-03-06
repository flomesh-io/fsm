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

package cluster

import (
	"context"
	"fmt"
	clusterv1alpha1 "github.com/flomesh-io/fsm/apis/cluster/v1alpha1"
	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/pkg/errors"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	kind      = "Cluster"
	groups    = "flomesh.io"
	resources = "clusters"
	versions  = "v1alpha1"

	mwPath = commons.ClusterMutatingWebhookPath
	mwName = "mcluster.kb.flomesh.io"
	vwPath = commons.ClusterValidatingWebhookPath
	vwName = "vcluster.kb.flomesh.io"
)

func RegisterWebhooks(webhookSvcNs, webhookSvcName string, caBundle []byte) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{groups},
		[]string{versions},
		[]string{resources},
	)

	mutatingWebhook := flomeshadmission.NewMutatingWebhook(
		mwName,
		webhookSvcNs,
		webhookSvcName,
		mwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	validatingWebhook := flomeshadmission.NewValidatingWebhook(
		vwName,
		webhookSvcNs,
		webhookSvcName,
		vwPath,
		caBundle,
		nil,
		[]admissionregv1.RuleWithOperations{rule},
	)

	flomeshadmission.RegisterMutatingWebhook(mwName, mutatingWebhook)
	flomeshadmission.RegisterValidatingWebhook(vwName, validatingWebhook)
}

type ClusterDefaulter struct {
	k8sAPI      *kube.K8sAPI
	configStore *config.Store
}

func NewDefaulter(k8sAPI *kube.K8sAPI, configStore *config.Store) *ClusterDefaulter {
	return &ClusterDefaulter{
		k8sAPI:      k8sAPI,
		configStore: configStore,
	}
}

func (w *ClusterDefaulter) Kind() string {
	return kind
}

func (w *ClusterDefaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return
	}

	klog.V(5).Infof("Default Webhook, name=%s", c.Name)
	klog.V(4).Infof("Before setting default values, spec=%#v", c.Spec)

	meshConfig := w.configStore.MeshConfig.GetConfig()

	if meshConfig == nil {
		return
	}

	if c.Spec.Mode == "" {
		c.Spec.Mode = clusterv1alpha1.InCluster
	}
	// for InCluster connector, it's name is always 'local'
	if c.Spec.Mode == clusterv1alpha1.InCluster {
		c.Name = "local"
		// TODO: checks if need to set r.Spec.ControlPlaneRepoRootUrl
	}

	if c.Spec.Replicas == nil {
		c.Spec.Replicas = defaultReplicas()
	}

	klog.V(4).Infof("After setting default values, spec=%#v", c.Spec)
}

func defaultReplicas() *int32 {
	var r int32 = 1
	return &r
}

type ClusterValidator struct {
	k8sAPI *kube.K8sAPI
}

func (w *ClusterValidator) Kind() string {
	return kind
}

func (w *ClusterValidator) ValidateCreate(obj interface{}) error {
	cluster, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return nil
	}

	if cluster.Spec.Mode == clusterv1alpha1.InCluster {
		// There can be ONLY ONE Cluster of InCluster mode
		clusterList, err := w.k8sAPI.FlomeshClient.
			ClusterV1alpha1().
			Clusters().
			List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Errorf("Failed to list Clusters, %#v", err)
			return err
		}

		numOfInCluster := 0
		for _, c := range clusterList.Items {
			if c.Spec.Mode == clusterv1alpha1.InCluster {
				numOfInCluster++
			}
		}
		if numOfInCluster >= 1 {
			errMsg := fmt.Sprintf("there're %d InCluster resources, should ONLY have exact ONE", numOfInCluster)
			klog.Errorf(errMsg)
			return errors.New(errMsg)
		}
	}

	return doValidation(obj)
}

func (w *ClusterValidator) ValidateUpdate(oldObj, obj interface{}) error {
	oldCluster, ok := oldObj.(*clusterv1alpha1.Cluster)
	if !ok {
		return nil
	}

	cluster, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return nil
	}

	if oldCluster.Spec.Mode != cluster.Spec.Mode {
		return errors.New("cannot update an immutable field: spec.Mode")
	}

	return doValidation(obj)
}

func (w *ClusterValidator) ValidateDelete(obj interface{}) error {
	return nil
}

func NewValidator(k8sAPI *kube.K8sAPI) *ClusterValidator {
	return &ClusterValidator{
		k8sAPI: k8sAPI,
	}
}

func doValidation(obj interface{}) error {
	c, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return nil
	}

	switch c.Spec.Mode {
	case clusterv1alpha1.OutCluster:
		if c.Spec.Gateway == "" {
			return errors.New("Gateway is required in OutCluster mode")
		}

		if c.Spec.Kubeconfig == "" {
			return errors.New("kubeconfig must be set in OutCluster mode")
		}

		if c.Name == "local" {
			return errors.New("Cluster Name 'local' is reserved for InCluster Mode ONLY, please change the cluster name")
		}

		if c.Spec.ControlPlaneRepoRootUrl == "" {
			return errors.New("controlPlaneRepoBaseUrl must be set in OutCluster mode")
		}
	case clusterv1alpha1.InCluster:
		return nil
	}

	return nil
}
