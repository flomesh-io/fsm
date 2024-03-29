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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AccessCertLister helps list AccessCerts.
// All objects returned here must be treated as read-only.
type AccessCertLister interface {
	// List lists all AccessCerts in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AccessCert, err error)
	// AccessCerts returns an object that can list and get AccessCerts.
	AccessCerts(namespace string) AccessCertNamespaceLister
	AccessCertListerExpansion
}

// accessCertLister implements the AccessCertLister interface.
type accessCertLister struct {
	indexer cache.Indexer
}

// NewAccessCertLister returns a new AccessCertLister.
func NewAccessCertLister(indexer cache.Indexer) AccessCertLister {
	return &accessCertLister{indexer: indexer}
}

// List lists all AccessCerts in the indexer.
func (s *accessCertLister) List(selector labels.Selector) (ret []*v1alpha1.AccessCert, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AccessCert))
	})
	return ret, err
}

// AccessCerts returns an object that can list and get AccessCerts.
func (s *accessCertLister) AccessCerts(namespace string) AccessCertNamespaceLister {
	return accessCertNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// AccessCertNamespaceLister helps list and get AccessCerts.
// All objects returned here must be treated as read-only.
type AccessCertNamespaceLister interface {
	// List lists all AccessCerts in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AccessCert, err error)
	// Get retrieves the AccessCert from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.AccessCert, error)
	AccessCertNamespaceListerExpansion
}

// accessCertNamespaceLister implements the AccessCertNamespaceLister
// interface.
type accessCertNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all AccessCerts in the indexer for a given namespace.
func (s accessCertNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.AccessCert, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AccessCert))
	})
	return ret, err
}

// Get retrieves the AccessCert from the indexer for a given namespace and name.
func (s accessCertNamespaceLister) Get(name string) (*v1alpha1.AccessCert, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("accesscert"), name)
	}
	return obj.(*v1alpha1.AccessCert), nil
}
