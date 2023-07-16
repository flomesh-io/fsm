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

package connector

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/mcs/cache"
	conn "github.com/flomesh-io/fsm/pkg/mcs/context"
	mcsevent "github.com/flomesh-io/fsm/pkg/mcs/event"
	"github.com/flomesh-io/fsm/pkg/version"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"time"
)

type Connector struct {
	context    context.Context
	k8sAPI     *kube.K8sAPI
	cache      *cache.Cache
	clusterCfg *config.Store
	broker     *mcsevent.Broker
}

func NewConnector(ctx context.Context, broker *mcsevent.Broker, resyncPeriod time.Duration) (*Connector, error) {
	connectorCtx := ctx.(*conn.ConnectorContext)

	k8sAPI, err := kube.NewAPIForConfig(connectorCtx.KubeConfig, 30*time.Second)
	if err != nil {
		return nil, err
	}

	if !version.IsSupportedK8sVersion(k8sAPI) {
		err := fmt.Errorf("kubernetes server version %s is not supported, requires at least %s",
			version.ServerVersion.String(), version.MinK8sVersion.String())
		klog.Error(err)

		return nil, err
	}

	clusterCfg := config.NewStore(k8sAPI)
	mc := clusterCfg.MeshConfig.GetConfig()

	// checks if fsm is installed in the cluster, this's a MUST otherwise it doesn't work
	_, err = k8sAPI.Client.AppsV1().
		Deployments(mc.GetMeshNamespace()).
		Get(context.TODO(), commons.ManagerDeploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Error("FSM Control Plane is not installed or not in a proper state, please check it.")
			return nil, err
		}

		klog.Errorf("Get FSM manager component %s/%s error: %s", mc.GetMeshNamespace(), commons.ManagerDeploymentName, err)
		return nil, err
	}

	connectorCache := cache.NewCache(connectorCtx, k8sAPI, clusterCfg, broker, resyncPeriod)

	return &Connector{
		context:    connectorCtx,
		k8sAPI:     k8sAPI,
		cache:      connectorCache,
		clusterCfg: clusterCfg,
		broker:     broker,
	}, nil
}
