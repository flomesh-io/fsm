/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	multiclusterv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// ServiceImportLister helps list ServiceImports.
// All objects returned here must be treated as read-only.
type ServiceImportLister interface {
	// List lists all ServiceImports in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*multiclusterv1alpha1.ServiceImport, err error)
	// ServiceImports returns an object that can list and get ServiceImports.
	ServiceImports(namespace string) ServiceImportNamespaceLister
	ServiceImportListerExpansion
}

// serviceImportLister implements the ServiceImportLister interface.
type serviceImportLister struct {
	listers.ResourceIndexer[*multiclusterv1alpha1.ServiceImport]
}

// NewServiceImportLister returns a new ServiceImportLister.
func NewServiceImportLister(indexer cache.Indexer) ServiceImportLister {
	return &serviceImportLister{listers.New[*multiclusterv1alpha1.ServiceImport](indexer, multiclusterv1alpha1.Resource("serviceimport"))}
}

// ServiceImports returns an object that can list and get ServiceImports.
func (s *serviceImportLister) ServiceImports(namespace string) ServiceImportNamespaceLister {
	return serviceImportNamespaceLister{listers.NewNamespaced[*multiclusterv1alpha1.ServiceImport](s.ResourceIndexer, namespace)}
}

// ServiceImportNamespaceLister helps list and get ServiceImports.
// All objects returned here must be treated as read-only.
type ServiceImportNamespaceLister interface {
	// List lists all ServiceImports in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*multiclusterv1alpha1.ServiceImport, err error)
	// Get retrieves the ServiceImport from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*multiclusterv1alpha1.ServiceImport, error)
	ServiceImportNamespaceListerExpansion
}

// serviceImportNamespaceLister implements the ServiceImportNamespaceLister
// interface.
type serviceImportNamespaceLister struct {
	listers.ResourceIndexer[*multiclusterv1alpha1.ServiceImport]
}
