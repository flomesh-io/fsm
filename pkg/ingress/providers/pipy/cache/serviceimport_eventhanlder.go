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
	"github.com/flomesh-io/fsm-classic/apis/serviceimport/v1alpha1"
	"k8s.io/klog/v2"
)

func (c *Cache) OnServiceImportAdd(serviceImport *v1alpha1.ServiceImport) {
	c.OnServiceImportUpdate(nil, serviceImport)
}

func (c *Cache) OnServiceImportUpdate(oldServiceImport, serviceImport *v1alpha1.ServiceImport) {
	if c.serviceImportChanges.Update(oldServiceImport, serviceImport) && c.isInitialized() {
		klog.V(5).Infof("Detects ServiceImport change, syncing...")
		c.Sync()
	}
}

func (c *Cache) OnServiceImportDelete(serviceImport *v1alpha1.ServiceImport) {
	c.OnServiceImportUpdate(serviceImport, nil)
}

func (c *Cache) OnServiceImportSynced() {
	c.mu.Lock()
	c.serviceImportSynced = true
	c.setInitialized(c.servicesSynced && c.endpointsSynced && c.ingressesSynced && c.ingressClassesSynced)
	c.mu.Unlock()

	c.syncRoutes()
}
