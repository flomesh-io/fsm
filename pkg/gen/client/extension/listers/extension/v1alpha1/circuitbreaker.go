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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CircuitBreakerLister helps list CircuitBreakers.
// All objects returned here must be treated as read-only.
type CircuitBreakerLister interface {
	// List lists all CircuitBreakers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.CircuitBreaker, err error)
	// CircuitBreakers returns an object that can list and get CircuitBreakers.
	CircuitBreakers(namespace string) CircuitBreakerNamespaceLister
	CircuitBreakerListerExpansion
}

// circuitBreakerLister implements the CircuitBreakerLister interface.
type circuitBreakerLister struct {
	indexer cache.Indexer
}

// NewCircuitBreakerLister returns a new CircuitBreakerLister.
func NewCircuitBreakerLister(indexer cache.Indexer) CircuitBreakerLister {
	return &circuitBreakerLister{indexer: indexer}
}

// List lists all CircuitBreakers in the indexer.
func (s *circuitBreakerLister) List(selector labels.Selector) (ret []*v1alpha1.CircuitBreaker, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CircuitBreaker))
	})
	return ret, err
}

// CircuitBreakers returns an object that can list and get CircuitBreakers.
func (s *circuitBreakerLister) CircuitBreakers(namespace string) CircuitBreakerNamespaceLister {
	return circuitBreakerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// CircuitBreakerNamespaceLister helps list and get CircuitBreakers.
// All objects returned here must be treated as read-only.
type CircuitBreakerNamespaceLister interface {
	// List lists all CircuitBreakers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.CircuitBreaker, err error)
	// Get retrieves the CircuitBreaker from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.CircuitBreaker, error)
	CircuitBreakerNamespaceListerExpansion
}

// circuitBreakerNamespaceLister implements the CircuitBreakerNamespaceLister
// interface.
type circuitBreakerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all CircuitBreakers in the indexer for a given namespace.
func (s circuitBreakerNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.CircuitBreaker, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CircuitBreaker))
	})
	return ret, err
}

// Get retrieves the CircuitBreaker from the indexer for a given namespace and name.
func (s circuitBreakerNamespaceLister) Get(name string) (*v1alpha1.CircuitBreaker, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("circuitbreaker"), name)
	}
	return obj.(*v1alpha1.CircuitBreaker), nil
}
