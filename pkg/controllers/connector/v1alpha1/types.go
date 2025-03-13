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

// Package v1alpha1 contains controller logic for the Connector API v1alpha1.
package v1alpha1

import (
	_ "embed"
	"fmt"

	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	connectorClientset "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/helm"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector-controller/v1alpha1")
)

var (
	//go:embed chart.tgz
	chartSource []byte
)

type connectorReconciler struct {
	recorder           record.EventRecorder
	fctx               *fctx.ControllerContext
	connectorAPIClient connectorClientset.Interface
}

func (r *connectorReconciler) NeedLeaderElection() bool {
	return true
}

func (r *connectorReconciler) deployConnector(connector ctv1.Connector, mc configurator.Configurator) (ctrl.Result, error) {
	actionConfig := helm.ActionConfig(connector.GetNamespace(), log.Debug().Msgf)

	templateClient := helm.TemplateClient(
		actionConfig,
		r.fctx.MeshName,
		mc.GetFSMNamespace(),
		constants.KubeVersion121,
	)
	if ctrlResult, err := helm.RenderChart(templateClient, connector, chartSource, mc, r.fctx.Client, r.fctx.Scheme, r.resolveValues); err != nil {
		defer r.recorder.Eventf(connector, corev1.EventTypeWarning, "Deploy", "Failed to deploy connector: %s", err)
		return ctrlResult, err
	}
	defer r.recorder.Eventf(connector, corev1.EventTypeNormal, "Deploy", "Deploy connector successfully")

	return ctrl.Result{}, nil
}

func (r *connectorReconciler) resolveValues(object metav1.Object, mc configurator.Configurator) (map[string]interface{}, error) {
	connector, ok := object.(ctv1.Connector)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of *connectorv1alpha1.Connector", object)
	}

	log.Debug().Msgf("[GW] Resolving Values ...")

	finalValues := make(map[string]interface{})

	overrides := []string{
		fmt.Sprintf("fsm.image.registry=%s", mc.GetImageRegistry()),
		fmt.Sprintf("fsm.image.tag=%s", mc.GetImageTag()),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.GetImagePullPolicy()),

		fmt.Sprintf("fsm.meshName=%s", r.fctx.MeshName),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetFSMNamespace()),
		fmt.Sprintf("fsm.fsmServiceAccountName=%s", r.fctx.FsmServiceAccount),
		fmt.Sprintf("fsm.trustDomain=%s", r.fctx.TrustDomain),

		fmt.Sprintf("fsm.controllerLogLevel=%s", mc.GetFSMLogLevel()),

		fmt.Sprintf("fsm.cloudConnector.enable=%t", true),
		fmt.Sprintf("fsm.cloudConnector.connectorProvider=%s", connector.GetProvider()),
		fmt.Sprintf("fsm.cloudConnector.connectorNamespace=%s", connector.GetNamespace()),
		fmt.Sprintf("fsm.cloudConnector.connectorName=%s", connector.GetName()),
		fmt.Sprintf("fsm.cloudConnector.connectorUID=%s", connector.GetUID()),

		fmt.Sprintf("fsm.cloudConnector.replicaCount=%d", replicas(connector, 1)),
		fmt.Sprintf("fsm.cloudConnector.resource.requests.cpu='%s'", requestsCpu(connector, resource.MustParse("0.5")).String()),
		fmt.Sprintf("fsm.cloudConnector.resource.requests.memory=%s", requestsMem(connector, resource.MustParse("128M")).String()),
		fmt.Sprintf("fsm.cloudConnector.resource.limits.cpu='%s'", limitsCpu(connector, resource.MustParse("1")).String()),
		fmt.Sprintf("fsm.cloudConnector.resource.limits.memory=%s", limitsMem(connector, resource.MustParse("1G")).String()),

		fmt.Sprintf("fsm.cloudConnector.leaderElection=%t", leaderElection(connector, true)),
	}

	image := mc.GetMeshConfig().Spec.Image
	if fsmConnectorImageName := image.Name[`fsmConnector`]; len(fsmConnectorImageName) > 0 {
		overrides = append(overrides, fmt.Sprintf("fsm.image.name.fsmConnector=%s", fsmConnectorImageName))
	}
	if fsmCurlImageName := image.Name[`fsmCurl`]; len(fsmCurlImageName) > 0 {
		overrides = append(overrides, fmt.Sprintf("fsm.image.name.fsmCurl=%s", fsmCurlImageName))
	}
	if fsmConnectorImageDigest := image.Digest[`fsmConnector`]; len(fsmConnectorImageDigest) > 0 {
		overrides = append(overrides, fmt.Sprintf("fsm.image.digest.fsmConnector=%s", fsmConnectorImageDigest))
	}
	if fsmCurlImageDigest := image.Digest[`fsmCurl`]; len(fsmCurlImageDigest) > 0 {
		overrides = append(overrides, fmt.Sprintf("fsm.image.digest.fsmCurl=%s", fsmCurlImageDigest))
	}

	if pullSecrets := connector.GetImagePullSecrets(); len(pullSecrets) > 0 {
		for index, pullSecret := range pullSecrets {
			overrides = append(overrides, fmt.Sprintf("fsm.imagePullSecrets[%d].name=%s", index, pullSecret.Name))
		}
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func replicas(connector ctv1.Connector, defVal int32) int32 {
	if connector.GetReplicas() == nil {
		return defVal
	}
	return *connector.GetReplicas()
}

func requestsCpu(connector ctv1.Connector, defVal resource.Quantity) *resource.Quantity {
	if connector.GetResources() == nil {
		return &defVal
	}

	if connector.GetResources().Requests.Cpu() == nil {
		return &defVal
	}

	if connector.GetResources().Requests.Cpu().Value() == 0 {
		return &defVal
	}

	return connector.GetResources().Requests.Cpu()
}

func requestsMem(connector ctv1.Connector, defVal resource.Quantity) *resource.Quantity {
	if connector.GetResources() == nil {
		return &defVal
	}

	if connector.GetResources().Requests.Memory() == nil {
		return &defVal
	}

	if connector.GetResources().Requests.Memory().Value() == 0 {
		return &defVal
	}

	return connector.GetResources().Requests.Memory()
}

func limitsCpu(connector ctv1.Connector, defVal resource.Quantity) *resource.Quantity {
	if connector.GetResources() == nil {
		return &defVal
	}

	if connector.GetResources().Limits.Cpu() == nil {
		return &defVal
	}

	if connector.GetResources().Limits.Cpu().Value() == 0 {
		return &defVal
	}

	return connector.GetResources().Limits.Cpu()
}

func limitsMem(connector ctv1.Connector, defVal resource.Quantity) *resource.Quantity {
	if connector.GetResources() == nil {
		return &defVal
	}

	if connector.GetResources().Limits.Memory() == nil {
		return &defVal
	}

	if connector.GetResources().Limits.Memory().Value() == 0 {
		return &defVal
	}

	return connector.GetResources().Limits.Memory()
}

func leaderElection(connector ctv1.Connector, defVal bool) bool {
	if connector.GetLeaderElection() == nil {
		return defVal
	}
	return *connector.GetLeaderElection()
}
