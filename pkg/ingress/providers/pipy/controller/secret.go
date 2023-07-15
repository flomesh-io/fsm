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

type SecretHandler interface {
	OnSecretAdd(secret *corev1.Secret)
	OnSecretUpdate(oldSecret, secret *corev1.Secret)
	OnSecretDelete(secret *corev1.Secret)
	OnSecretSynced()
}

type SecretController struct {
	Informer     cache.SharedIndexInformer
	Store        SecretStore
	HasSynced    cache.InformerSynced
	Lister       v1.SecretLister
	eventHandler SecretHandler
}

type SecretStore struct {
	cache.Store
}

func (l *SecretStore) ByKey(key string) (*corev1.Secret, error) {
	s, exists, err := l.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("no object matching key %q in local store", key)
	}
	return s.(*corev1.Secret), nil
}

func NewSecretControllerWithEventHandler(secretInformer coreinformers.SecretInformer, resyncPeriod time.Duration, handler SecretHandler) *SecretController {
	informer := secretInformer.Informer()

	result := &SecretController{
		HasSynced: informer.HasSynced,
		Informer:  informer,
		Lister:    secretInformer.Lister(),
		Store: SecretStore{
			Store: informer.GetStore(),
		},
	}

	informer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddSecret,
			UpdateFunc: result.handleUpdateSecret,
			DeleteFunc: result.handleDeleteSecret,
		},
		resyncPeriod,
	)

	if handler != nil {
		result.eventHandler = handler
	}

	return result
}

func (c *SecretController) Run(stopCh <-chan struct{}) {
	klog.InfoS("Starting secret config controller")

	if !cache.WaitForNamedCacheSync("secret config", stopCh, c.HasSynced) {
		return
	}

	if c.eventHandler != nil {
		klog.V(3).Info("Calling handler.OnSecretSynced()")
		c.eventHandler.OnSecretSynced()
	}
}

func (c *SecretController) handleAddSecret(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnSecretAdd")
		c.eventHandler.OnSecretAdd(secret)
	}
}

func (c *SecretController) handleUpdateSecret(oldObj, newObj interface{}) {
	oldSecret, ok := oldObj.(*corev1.Secret)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	secret, ok := newObj.(*corev1.Secret)
	if !ok {
		runtime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}

	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnSecretUpdate")
		c.eventHandler.OnSecretUpdate(oldSecret, secret)
	}
}

func (c *SecretController) handleDeleteSecret(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if secret, ok = tombstone.Obj.(*corev1.Secret); !ok {
			runtime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	if c.eventHandler != nil {
		klog.V(4).Info("Calling handler.OnSecretDelete")
		c.eventHandler.OnSecretDelete(secret)
	}
}
