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

package cache

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	fsminformers "github.com/flomesh-io/fsm-classic/pkg/generated/informers/externalversions"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	mcscfg "github.com/flomesh-io/fsm-classic/pkg/mcs/config"
	conn "github.com/flomesh-io/fsm-classic/pkg/mcs/context"
	"github.com/flomesh-io/fsm-classic/pkg/mcs/controller"
	mcsevent "github.com/flomesh-io/fsm-classic/pkg/mcs/event"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/async"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Cache struct {
	connectorConfig *mcscfg.ConnectorConfig
	k8sAPI          *kube.K8sAPI
	recorder        events.EventRecorder
	clusterCfg      *config.Store
	broker          *mcsevent.Broker

	mu sync.Mutex

	serviceExportSynced bool
	initialized         int32
	syncRunner          *async.BoundedFrequencyRunner

	controllers *controller.Controllers
	broadcaster events.EventBroadcaster
}

func NewCache(ctx context.Context, api *kube.K8sAPI, clusterCfg *config.Store, broker *mcsevent.Broker, resyncPeriod time.Duration) *Cache {
	connectorCtx := ctx.(*conn.ConnectorContext)
	key := connectorCtx.ClusterKey
	formattedKey := strings.ReplaceAll(key, "/", "-")
	klog.Infof("Creating cache for Cluster [%s] ...", key)

	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: api.Client.EventsV1()})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, fmt.Sprintf("fsm-cluster-connector-remote-%s", formattedKey))

	c := &Cache{
		connectorConfig: connectorCtx.ConnectorConfig,
		k8sAPI:          api,
		recorder:        recorder,
		clusterCfg:      clusterCfg,
		broadcaster:     eventBroadcaster,
		broker:          broker,
	}

	fsmInformerFactory := fsminformers.NewSharedInformerFactoryWithOptions(api.FlomeshClient, resyncPeriod)
	serviceExportController := controller.NewServiceExportControllerWithEventHandler(
		fsmInformerFactory.Serviceexport().V1alpha1().ServiceExports(),
		resyncPeriod,
		c,
	)

	c.controllers = &controller.Controllers{
		ServiceExport: serviceExportController,
	}

	minSyncPeriod := 3 * time.Second
	syncPeriod := 30 * time.Second
	burstSyncs := 2
	runnerName := fmt.Sprintf("sync-runner-%s", formattedKey)
	c.syncRunner = async.NewBoundedFrequencyRunner(runnerName, c.syncManagedCluster, minSyncPeriod, syncPeriod, burstSyncs)

	return c
}

func (c *Cache) setInitialized(value bool) {
	var initialized int32
	if value {
		initialized = 1
	}
	atomic.StoreInt32(&c.initialized, initialized)
}

func (c *Cache) syncManagedCluster() {
	// Nothing to do for the time-being

	//c.mu.Lock()
	//defer c.mu.Unlock()
	klog.Infof("[%s] Syncing resources of managed clusters ...", c.connectorConfig.Key())
}

func (c *Cache) Sync() {
	c.syncRunner.Run()
}

func (c *Cache) SyncLoop(stopCh <-chan struct{}) {
	c.syncRunner.Loop(stopCh)
}

func (c *Cache) GetBroadcaster() events.EventBroadcaster {
	return c.broadcaster
}

func (c *Cache) GetControllers() *controller.Controllers {
	return c.controllers
}

func (c *Cache) GetRecorder() events.EventRecorder {
	return c.recorder
}
