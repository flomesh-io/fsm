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

package controller

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

type EndpointsHandler interface {
	OnEndpointsAdd(endpoints *corev1.Endpoints)
	OnEndpointsUpdate(oldEndpoints, endpoints *corev1.Endpoints)
	OnEndpointsDelete(endpoints *corev1.Endpoints)
	OnEndpointsSynced()
}

type EndpointsController struct {
	Informer     cache.SharedIndexInformer
	Store        EndpointStore
	HasSynced    cache.InformerSynced
	Lister       v1.EndpointsLister
	eventHandler EndpointsHandler
}

type EndpointStore struct {
	cache.Store
}

func (l *EndpointStore) ByKey(key string) (*corev1.Endpoints, error) {
	s, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("no object matching key %q in local store", key)
	}
	return s.(*corev1.Endpoints), nil
}

func NewEndpointsControllerWithEventHandler(endpointsInformer coreinformers.EndpointsInformer, resyncPeriod time.Duration, handler EndpointsHandler) *EndpointsController {
	informer := endpointsInformer.Informer()

	result := &EndpointsController{
		HasSynced: informer.HasSynced,
		Informer:  informer,
		Lister:    endpointsInformer.Lister(),
		Store: EndpointStore{
			Store: informer.GetStore(),
		},
	}

	informer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddEndpoints,
			UpdateFunc: result.handleUpdateEndpoints,
			DeleteFunc: result.handleDeleteEndpoints,
		},
		resyncPeriod,
	)

	if handler != nil {
		result.eventHandler = handler
	}

	return result
}

func (c *EndpointsController) Run(stopCh <-chan struct{}) {
	klog.InfoS("Starting endpoints config controller")

	if !cache.WaitForNamedCacheSync("endpoints config", stopCh, c.HasSynced) {
		return
	}

	if c.eventHandler != nil {
		klog.V(3).Info("Calling handler.OnEndpointsSynced()")
		c.eventHandler.OnEndpointsSynced()
	}
}

func (c *EndpointsController) handleAddEndpoints(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnEndpointsAdd")
		c.eventHandler.OnEndpointsAdd(endpoints)
	}
}

func (c *EndpointsController) handleUpdateEndpoints(oldObj, newObj interface{}) {
	oldEndpoints, ok := oldObj.(*corev1.Endpoints)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	endpoints, ok := newObj.(*corev1.Endpoints)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnEndpointsUpdate")
		c.eventHandler.OnEndpointsUpdate(oldEndpoints, endpoints)
	}
}

func (c *EndpointsController) handleDeleteEndpoints(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if endpoints, ok = tombstone.Obj.(*corev1.Endpoints); !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnEndpointsDelete")
		c.eventHandler.OnEndpointsDelete(endpoints)
	}
}
