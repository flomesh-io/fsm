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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
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
	listers.ResourceIndexer[*v1alpha1.AccessCert]
}

// NewAccessCertLister returns a new AccessCertLister.
func NewAccessCertLister(indexer cache.Indexer) AccessCertLister {
	return &accessCertLister{listers.New[*v1alpha1.AccessCert](indexer, v1alpha1.Resource("accesscert"))}
}

// AccessCerts returns an object that can list and get AccessCerts.
func (s *accessCertLister) AccessCerts(namespace string) AccessCertNamespaceLister {
	return accessCertNamespaceLister{listers.NewNamespaced[*v1alpha1.AccessCert](s.ResourceIndexer, namespace)}
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
	listers.ResourceIndexer[*v1alpha1.AccessCert]
}
