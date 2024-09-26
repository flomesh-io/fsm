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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
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
	listers.ResourceIndexer[*v1alpha1.CircuitBreaker]
}

// NewCircuitBreakerLister returns a new CircuitBreakerLister.
func NewCircuitBreakerLister(indexer cache.Indexer) CircuitBreakerLister {
	return &circuitBreakerLister{listers.New[*v1alpha1.CircuitBreaker](indexer, v1alpha1.Resource("circuitbreaker"))}
}

// CircuitBreakers returns an object that can list and get CircuitBreakers.
func (s *circuitBreakerLister) CircuitBreakers(namespace string) CircuitBreakerNamespaceLister {
	return circuitBreakerNamespaceLister{listers.NewNamespaced[*v1alpha1.CircuitBreaker](s.ResourceIndexer, namespace)}
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
	listers.ResourceIndexer[*v1alpha1.CircuitBreaker]
}
