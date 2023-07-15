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
	svcimpv1alpha1 "github.com/flomesh-io/fsm-classic/apis/serviceimport/v1alpha1"
	svcimpv1alpha1informers "github.com/flomesh-io/fsm-classic/pkg/generated/informers/externalversions/serviceimport/v1alpha1"
	svcimpv1alpha1lister "github.com/flomesh-io/fsm-classic/pkg/generated/listers/serviceimport/v1alpha1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

type ServiceImportHandler interface {
	OnServiceImportAdd(serviceImport *svcimpv1alpha1.ServiceImport)
	OnServiceImportUpdate(oldServiceImport, serviceImport *svcimpv1alpha1.ServiceImport)
	OnServiceImportDelete(serviceImport *svcimpv1alpha1.ServiceImport)
	OnServiceImportSynced()
}

type ServiceImportController struct {
	Informer     cache.SharedIndexInformer
	Store        ServiceImportStore
	HasSynced    cache.InformerSynced
	Lister       svcimpv1alpha1lister.ServiceImportLister
	eventHandler ServiceImportHandler
}

type ServiceImportStore struct {
	cache.Store
}

func (l *ServiceImportStore) ByKey(key string) (*svcimpv1alpha1.ServiceImport, error) {
	s, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("no object matching key %q in local store", key)
	}
	return s.(*svcimpv1alpha1.ServiceImport), nil
}

func NewServiceImportControllerWithEventHandler(serviceImportInformer svcimpv1alpha1informers.ServiceImportInformer, resyncPeriod time.Duration, handler ServiceImportHandler) *ServiceImportController {
	informer := serviceImportInformer.Informer()

	result := &ServiceImportController{
		HasSynced: informer.HasSynced,
		Informer:  informer,
		Lister:    serviceImportInformer.Lister(),
		Store: ServiceImportStore{
			Store: informer.GetStore(),
		},
	}

	informer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddServiceImport,
			UpdateFunc: result.handleUpdateServiceImport,
			DeleteFunc: result.handleDeleteServiceImport,
		},
		resyncPeriod,
	)

	if handler != nil {
		result.eventHandler = handler
	}

	return result
}

func (c *ServiceImportController) Run(stopCh <-chan struct{}) {
	klog.InfoS("Starting ServiceImport config controller")

	if !cache.WaitForNamedCacheSync("ServiceImport config", stopCh, c.HasSynced) {
		return
	}

	if c.eventHandler != nil {
		klog.V(3).Info("Calling handler.OnServiceImportSynced()")
		c.eventHandler.OnServiceImportSynced()
	}
}

func (c *ServiceImportController) handleAddServiceImport(obj interface{}) {
	serviceImport, ok := obj.(*svcimpv1alpha1.ServiceImport)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnServiceImportAdd")
		c.eventHandler.OnServiceImportAdd(serviceImport)
	}
}

func (c *ServiceImportController) handleUpdateServiceImport(oldObj, newObj interface{}) {
	oldServiceImport, ok := oldObj.(*svcimpv1alpha1.ServiceImport)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	serviceImport, ok := newObj.(*svcimpv1alpha1.ServiceImport)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnServiceImportUpdate")
		c.eventHandler.OnServiceImportUpdate(oldServiceImport, serviceImport)
	}
}

func (c *ServiceImportController) handleDeleteServiceImport(obj interface{}) {
	serviceImport, ok := obj.(*svcimpv1alpha1.ServiceImport)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if serviceImport, ok = tombstone.Obj.(*svcimpv1alpha1.ServiceImport); !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnServiceImportDelete")
		c.eventHandler.OnServiceImportDelete(serviceImport)
	}
}
