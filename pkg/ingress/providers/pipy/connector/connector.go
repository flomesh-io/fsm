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
	"fmt"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

func (c *Connector) Run(stopCh <-chan struct{}) error {
	errCh := make(chan error)

	if c.cache.GetBroadcaster() != nil && c.kubeClient != nil {
		klog.V(3).Infof("Starting broadcaster ......")
		c.cache.GetBroadcaster().StartRecordingToSink(stopCh)
	}

	// register event handlers
	klog.V(3).Infof("Registering event handlers ......")
	controllers := c.cache.GetControllers()

	go controllers.Service.Run(stopCh)
	go controllers.Endpoints.Run(stopCh)
	go controllers.IngressClassv1.Run(stopCh)
	go controllers.Ingressv1.Run(stopCh)
	go controllers.ServiceImport.Run(stopCh)
	go controllers.Secret.Run(stopCh)

	// start the informers manually
	klog.V(3).Infof("Starting informers(svc, ep & ingress class) ......")
	go controllers.Service.Informer.Run(stopCh)
	go controllers.Endpoints.Informer.Run(stopCh)
	go controllers.Secret.Informer.Run(stopCh)
	go controllers.IngressClassv1.Informer.Run(stopCh)

	klog.V(3).Infof("Waiting for caches to be synced ......")
	// Ingress depends on service & enpoints, they must be synced first
	if !k8scache.WaitForCacheSync(stopCh,
		controllers.Endpoints.HasSynced,
		controllers.Service.HasSynced,
		controllers.Secret.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for services, endpoints & secrets caches to sync"))
	}

	// Ingress also depends on IngressClass, but it'c not needed to have relation with svc & ep
	if !k8scache.WaitForCacheSync(stopCh, controllers.IngressClassv1.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for ingress class cache to sync"))
	}

	// start the ServiceImport Informer
	klog.V(3).Infof("Starting ServiceImport informer ......")
	go controllers.ServiceImport.Informer.Run(stopCh)
	if !k8scache.WaitForCacheSync(stopCh, controllers.ServiceImport.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for ServiceExport to sync"))
	}

	// Sleep for a while, so that there's enough time for processing
	klog.V(5).Infof("Sleep for a while ......")
	time.Sleep(1 * time.Second)

	// start the Ingress Informer
	klog.V(3).Infof("Starting ingress informer ......")
	go controllers.Ingressv1.Informer.Run(stopCh)
	if !k8scache.WaitForCacheSync(stopCh, controllers.Ingressv1.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for ingress caches to sync"))
	}

	// start the cache runner
	go c.cache.SyncLoop(stopCh)

	return <-errCh
}
