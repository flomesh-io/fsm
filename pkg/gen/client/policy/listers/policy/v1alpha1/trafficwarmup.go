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
	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// TrafficWarmupLister helps list TrafficWarmups.
// All objects returned here must be treated as read-only.
type TrafficWarmupLister interface {
	// List lists all TrafficWarmups in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*policyv1alpha1.TrafficWarmup, err error)
	// TrafficWarmups returns an object that can list and get TrafficWarmups.
	TrafficWarmups(namespace string) TrafficWarmupNamespaceLister
	TrafficWarmupListerExpansion
}

// trafficWarmupLister implements the TrafficWarmupLister interface.
type trafficWarmupLister struct {
	listers.ResourceIndexer[*policyv1alpha1.TrafficWarmup]
}

// NewTrafficWarmupLister returns a new TrafficWarmupLister.
func NewTrafficWarmupLister(indexer cache.Indexer) TrafficWarmupLister {
	return &trafficWarmupLister{listers.New[*policyv1alpha1.TrafficWarmup](indexer, policyv1alpha1.Resource("trafficwarmup"))}
}

// TrafficWarmups returns an object that can list and get TrafficWarmups.
func (s *trafficWarmupLister) TrafficWarmups(namespace string) TrafficWarmupNamespaceLister {
	return trafficWarmupNamespaceLister{listers.NewNamespaced[*policyv1alpha1.TrafficWarmup](s.ResourceIndexer, namespace)}
}

// TrafficWarmupNamespaceLister helps list and get TrafficWarmups.
// All objects returned here must be treated as read-only.
type TrafficWarmupNamespaceLister interface {
	// List lists all TrafficWarmups in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*policyv1alpha1.TrafficWarmup, err error)
	// Get retrieves the TrafficWarmup from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*policyv1alpha1.TrafficWarmup, error)
	TrafficWarmupNamespaceListerExpansion
}

// trafficWarmupNamespaceLister implements the TrafficWarmupNamespaceLister
// interface.
type trafficWarmupNamespaceLister struct {
	listers.ResourceIndexer[*policyv1alpha1.TrafficWarmup]
}
