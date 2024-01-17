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

// Package cluster contains webhook logic for the Cluster resource
package cluster

import (
	"fmt"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"

	flomeshadmission "github.com/flomesh-io/fsm/pkg/admission"
	clusterv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

type register struct {
	*webhook.RegisterConfig
}

// NewRegister creates a new cluster webhook register
func NewRegister(cfg *webhook.RegisterConfig) webhook.Register {
	return &register{
		RegisterConfig: cfg,
	}
}

// GetWebhooks returns the webhooks of the cluster resource
func (r *register) GetWebhooks() ([]admissionregv1.MutatingWebhook, []admissionregv1.ValidatingWebhook) {
	rule := flomeshadmission.NewRule(
		[]admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		[]string{"flomesh.io"},
		[]string{"v1alpha1"},
		[]string{"clusters"},
	)

	return []admissionregv1.MutatingWebhook{flomeshadmission.NewMutatingWebhook(
			"mcluster.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.ClusterMutatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}, []admissionregv1.ValidatingWebhook{flomeshadmission.NewValidatingWebhook(
			"vcluster.kb.flomesh.io",
			r.WebhookSvcNs,
			r.WebhookSvcName,
			constants.ClusterValidatingWebhookPath,
			r.CaBundle,
			nil,
			nil,
			admissionregv1.Fail,
			[]admissionregv1.RuleWithOperations{rule},
		)}
}

// GetHandlers returns the webhook handlers of the cluster resource
func (r *register) GetHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		constants.ClusterMutatingWebhookPath:   webhook.DefaultingWebhookFor(r.Scheme, newDefaulter(r.KubeClient, r.Config)),
		constants.ClusterValidatingWebhookPath: webhook.ValidatingWebhookFor(r.Scheme, newValidator(r.KubeClient)),
	}
}

type defaulter struct {
	kubeClient kubernetes.Interface
	cfg        configurator.Configurator
}

func newDefaulter(kubeClient kubernetes.Interface, cfg configurator.Configurator) *defaulter {
	return &defaulter{
		kubeClient: kubeClient,
		cfg:        cfg,
	}
}

func (w *defaulter) RuntimeObject() runtime.Object {
	return &clusterv1alpha1.Cluster{}
}

func (w *defaulter) SetDefaults(obj interface{}) {
	c, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return
	}

	log.Debug().Msgf("Default Webhook, name=%s", c.Name)
	log.Debug().Msgf("Before setting default values, spec=%v", c.Spec)

	//meshConfig := w.configStore.MeshConfig.GetConfig()
	//
	//if meshConfig == nil {
	//	return
	//}

	// for InCluster connector, it's name is always 'local'
	//if c.Spec.IsInCluster {
	//	c.Name = "local"
	//}
	//if c.Labels == nil {
	//	c.Labels = make(map[string]string)
	//}
	//
	//if c.Spec.IsInCluster {
	//	c.Labels[constants.MultiClustersConnectorMode] = "local"
	//} else {
	//	c.Labels[constants.MultiClustersConnectorMode] = "remote"
	//}

	log.Debug().Msgf("After setting default values, spec=%v", c.Spec)
}

type validator struct {
	kubeClient kubernetes.Interface
}

// RuntimeObject returns the runtime object of the validator
func (w *validator) RuntimeObject() runtime.Object {
	return &clusterv1alpha1.Cluster{}
}

// ValidateCreate validates the creation of the cluster resource
func (w *validator) ValidateCreate(obj interface{}) error {
	//cluster, ok := obj.(*clusterv1alpha1.Cluster)
	//if !ok {
	//	return nil
	//}

	//if cluster.Spec.IsInCluster {
	//	// There can be ONLY ONE Cluster of InCluster mode
	//	clusterList, err := w.k8sAPI.FlomeshClient.
	//		ClusterV1alpha1().
	//		Clusters().
	//		List(context.TODO(), metav1.ListOptions{})
	//	if err != nil {
	//		log.Error().Msgf("Failed to list Clusters, %v", err)
	//		return err
	//	}
	//
	//	numOfInCluster := 0
	//	for _, c := range clusterList.Items {
	//		if c.Spec.IsInCluster {
	//			numOfInCluster++
	//		}
	//	}
	//	if numOfInCluster >= 1 {
	//		errMsg := fmt.Sprintf("there're %d InCluster resources, should ONLY have exact ONE", numOfInCluster)
	//		log.Error().Msgf(errMsg)
	//		return errors.New(errMsg)
	//	}
	//}

	return doValidation(obj)
}

// ValidateUpdate validates the update of the cluster resource
func (w *validator) ValidateUpdate(_, obj interface{}) error {
	//oldCluster, ok := oldObj.(*clusterv1alpha1.Cluster)
	//if !ok {
	//	return nil
	//}
	//
	//cluster, ok := obj.(*clusterv1alpha1.Cluster)
	//if !ok {
	//	return nil
	//}

	//if oldCluster.Spec.IsInCluster != cluster.Spec.IsInCluster {
	//	return errors.New("cannot update an immutable field: spec.IsInCluster")
	//}

	return doValidation(obj)
}

// ValidateDelete validates the deletion of the cluster resource
func (w *validator) ValidateDelete(_ interface{}) error {
	return nil
}

func newValidator(kubeClient kubernetes.Interface) *validator {
	return &validator{
		kubeClient: kubeClient,
	}
}

func doValidation(obj interface{}) error {
	c, ok := obj.(*clusterv1alpha1.Cluster)
	if !ok {
		return nil
	}

	//if c.Labels == nil || c.Labels[constants.MultiClustersConnectorMode] == "" {
	//	return fmt.Errorf("missing required label 'multicluster.flomesh.io/connector-mode'")
	//}

	//connectorMode := c.Labels[constants.MultiClustersConnectorMode]
	//switch connectorMode {
	//case "local", "remote":
	//	log.Debug().Msgf("multicluster.flomesh.io/connector-mode=%s", connectorMode)
	//default:
	//	return fmt.Errorf("invalid value %q for label multicluster.flomesh.io/connector-mode, must be either 'local' or 'remote'", connectorMode)
	//}

	//if c.Spec.IsInCluster {
	//	if connectorMode == "remote" {
	//		return fmt.Errorf("label and spec doesn't match: multicluster.flomesh.io/connector-mode=remote, spec.IsInCluster=true")
	//	}
	//
	//	return nil
	//} else {
	//	if connectorMode == "local" {
	//		return fmt.Errorf("label and spec doesn't match: multicluster.flomesh.io/connector-mode=local, spec.IsInCluster=false")
	//	}

	host := c.Spec.GatewayHost
	if host == "" {
		return errors.New("GatewayHost is required in OutCluster mode")
	}

	if c.Spec.Kubeconfig == "" {
		return fmt.Errorf("kubeconfig must be set in OutCluster mode")
	}

	//if c.Name == "local" {
	//	return errors.New("Cluster Name 'local' is reserved for InCluster Mode ONLY, please change the cluster name")
	//}

	isDNSName := false
	if ipErrs := validation.IsValidIPv4Address(field.NewPath(""), host); len(ipErrs) > 0 {
		// Not IPv4 address
		log.Warn().Msgf("%q is NOT a valid IPv4 address: %v", host, ipErrs)
		if dnsErrs := validation.IsDNS1123Subdomain(host); len(dnsErrs) > 0 {
			// Not valid DNS domain name
			return fmt.Errorf("invalid DNS name %q: %v", host, dnsErrs)
		}

		// is DNS name
		isDNSName = true
	}

	var gwIPv4 net.IP
	if isDNSName {
		ipAddr, err := net.ResolveIPAddr("ip4", host)
		if err != nil {
			return fmt.Errorf("%q cannot be resolved to IP", host)
		}
		log.Debug().Msgf("%q is resolved to IP: %s", host, ipAddr.IP)
		gwIPv4 = ipAddr.IP.To4()
	} else {
		gwIPv4 = net.ParseIP(host).To4()
	}

	if gwIPv4 == nil {
		return fmt.Errorf("%q cannot be resolved to a IPv4 address", host)
	}

	if gwIPv4 != nil && (gwIPv4.IsLoopback() || gwIPv4.IsUnspecified()) {
		return fmt.Errorf("gateway Host %s is resolved to Loopback IP or Unspecified", host)
	}

	port := int(c.Spec.GatewayPort)
	if errs := validation.IsValidPortNum(port); len(errs) > 0 {
		return fmt.Errorf("invalid port number %d: %v", c.Spec.GatewayPort, errs)
	}

	return nil
}
