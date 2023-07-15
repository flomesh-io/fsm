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

type ServiceHandler interface {
	OnServiceAdd(service *corev1.Service)
	OnServiceUpdate(oldService, service *corev1.Service)
	OnServiceDelete(service *corev1.Service)
	OnServiceSynced()
}

type ServiceController struct {
	Informer     cache.SharedIndexInformer
	Store        ServiceStore
	HasSynced    cache.InformerSynced
	Lister       v1.ServiceLister
	eventHandler ServiceHandler
}

type ServiceStore struct {
	cache.Store
}

func (l *ServiceStore) ByKey(key string) (*corev1.Service, error) {
	s, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("no object matching key %q in local store", key)
	}
	return s.(*corev1.Service), nil
}

func NewServiceControllerWithEventHandler(serviceInformer coreinformers.ServiceInformer, resyncPeriod time.Duration, handler ServiceHandler) *ServiceController {
	informer := serviceInformer.Informer()

	result := &ServiceController{
		HasSynced: informer.HasSynced,
		Informer:  informer,
		Lister:    serviceInformer.Lister(),
		Store: ServiceStore{
			Store: informer.GetStore(),
		},
	}

	informer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddService,
			UpdateFunc: result.handleUpdateService,
			DeleteFunc: result.handleDeleteService,
		},
		resyncPeriod,
	)

	if handler != nil {
		result.eventHandler = handler
	}

	return result
}

func (c *ServiceController) Run(stopCh <-chan struct{}) {
	klog.InfoS("Starting service config controller")

	if !cache.WaitForNamedCacheSync("service config", stopCh, c.HasSynced) {
		return
	}

	if c.eventHandler != nil {
		klog.V(3).Info("Calling handler.OnServiceSynced()")
		c.eventHandler.OnServiceSynced()
	}
}

func (c *ServiceController) handleAddService(obj interface{}) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnServiceAdd")
		c.eventHandler.OnServiceAdd(service)
	}
}

func (c *ServiceController) handleUpdateService(oldObj, newObj interface{}) {
	oldService, ok := oldObj.(*corev1.Service)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	service, ok := newObj.(*corev1.Service)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnServiceUpdate")
		c.eventHandler.OnServiceUpdate(oldService, service)
	}
}

func (c *ServiceController) handleDeleteService(obj interface{}) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if service, ok = tombstone.Obj.(*corev1.Service); !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnServiceDelete")
		c.eventHandler.OnServiceDelete(service)
	}
}
