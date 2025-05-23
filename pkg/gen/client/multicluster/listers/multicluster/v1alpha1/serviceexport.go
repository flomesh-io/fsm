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

// ServiceExportLister helps list ServiceExports.
// All objects returned here must be treated as read-only.
type ServiceExportLister interface {
	// List lists all ServiceExports in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*multiclusterv1alpha1.ServiceExport, err error)
	// ServiceExports returns an object that can list and get ServiceExports.
	ServiceExports(namespace string) ServiceExportNamespaceLister
	ServiceExportListerExpansion
}

// serviceExportLister implements the ServiceExportLister interface.
type serviceExportLister struct {
	listers.ResourceIndexer[*multiclusterv1alpha1.ServiceExport]
}

// NewServiceExportLister returns a new ServiceExportLister.
func NewServiceExportLister(indexer cache.Indexer) ServiceExportLister {
	return &serviceExportLister{listers.New[*multiclusterv1alpha1.ServiceExport](indexer, multiclusterv1alpha1.Resource("serviceexport"))}
}

// ServiceExports returns an object that can list and get ServiceExports.
func (s *serviceExportLister) ServiceExports(namespace string) ServiceExportNamespaceLister {
	return serviceExportNamespaceLister{listers.NewNamespaced[*multiclusterv1alpha1.ServiceExport](s.ResourceIndexer, namespace)}
}

// ServiceExportNamespaceLister helps list and get ServiceExports.
// All objects returned here must be treated as read-only.
type ServiceExportNamespaceLister interface {
	// List lists all ServiceExports in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*multiclusterv1alpha1.ServiceExport, err error)
	// Get retrieves the ServiceExport from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*multiclusterv1alpha1.ServiceExport, error)
	ServiceExportNamespaceListerExpansion
}

// serviceExportNamespaceLister implements the ServiceExportNamespaceLister
// interface.
type serviceExportNamespaceLister struct {
	listers.ResourceIndexer[*multiclusterv1alpha1.ServiceExport]
}
