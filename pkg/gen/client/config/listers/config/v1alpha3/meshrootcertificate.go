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

package v1alpha3

import (
	v1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// MeshRootCertificateLister helps list MeshRootCertificates.
// All objects returned here must be treated as read-only.
type MeshRootCertificateLister interface {
	// List lists all MeshRootCertificates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha3.MeshRootCertificate, err error)
	// MeshRootCertificates returns an object that can list and get MeshRootCertificates.
	MeshRootCertificates(namespace string) MeshRootCertificateNamespaceLister
	MeshRootCertificateListerExpansion
}

// meshRootCertificateLister implements the MeshRootCertificateLister interface.
type meshRootCertificateLister struct {
	indexer cache.Indexer
}

// NewMeshRootCertificateLister returns a new MeshRootCertificateLister.
func NewMeshRootCertificateLister(indexer cache.Indexer) MeshRootCertificateLister {
	return &meshRootCertificateLister{indexer: indexer}
}

// List lists all MeshRootCertificates in the indexer.
func (s *meshRootCertificateLister) List(selector labels.Selector) (ret []*v1alpha3.MeshRootCertificate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha3.MeshRootCertificate))
	})
	return ret, err
}

// MeshRootCertificates returns an object that can list and get MeshRootCertificates.
func (s *meshRootCertificateLister) MeshRootCertificates(namespace string) MeshRootCertificateNamespaceLister {
	return meshRootCertificateNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MeshRootCertificateNamespaceLister helps list and get MeshRootCertificates.
// All objects returned here must be treated as read-only.
type MeshRootCertificateNamespaceLister interface {
	// List lists all MeshRootCertificates in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha3.MeshRootCertificate, err error)
	// Get retrieves the MeshRootCertificate from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha3.MeshRootCertificate, error)
	MeshRootCertificateNamespaceListerExpansion
}

// meshRootCertificateNamespaceLister implements the MeshRootCertificateNamespaceLister
// interface.
type meshRootCertificateNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MeshRootCertificates in the indexer for a given namespace.
func (s meshRootCertificateNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.MeshRootCertificate, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha3.MeshRootCertificate))
	})
	return ret, err
}

// Get retrieves the MeshRootCertificate from the indexer for a given namespace and name.
func (s meshRootCertificateNamespaceLister) Get(name string) (*v1alpha3.MeshRootCertificate, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha3.Resource("meshrootcertificate"), name)
	}
	return obj.(*v1alpha3.MeshRootCertificate), nil
}
