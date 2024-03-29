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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// UpstreamTLSPolicyLister helps list UpstreamTLSPolicies.
// All objects returned here must be treated as read-only.
type UpstreamTLSPolicyLister interface {
	// List lists all UpstreamTLSPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.UpstreamTLSPolicy, err error)
	// UpstreamTLSPolicies returns an object that can list and get UpstreamTLSPolicies.
	UpstreamTLSPolicies(namespace string) UpstreamTLSPolicyNamespaceLister
	UpstreamTLSPolicyListerExpansion
}

// upstreamTLSPolicyLister implements the UpstreamTLSPolicyLister interface.
type upstreamTLSPolicyLister struct {
	indexer cache.Indexer
}

// NewUpstreamTLSPolicyLister returns a new UpstreamTLSPolicyLister.
func NewUpstreamTLSPolicyLister(indexer cache.Indexer) UpstreamTLSPolicyLister {
	return &upstreamTLSPolicyLister{indexer: indexer}
}

// List lists all UpstreamTLSPolicies in the indexer.
func (s *upstreamTLSPolicyLister) List(selector labels.Selector) (ret []*v1alpha1.UpstreamTLSPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.UpstreamTLSPolicy))
	})
	return ret, err
}

// UpstreamTLSPolicies returns an object that can list and get UpstreamTLSPolicies.
func (s *upstreamTLSPolicyLister) UpstreamTLSPolicies(namespace string) UpstreamTLSPolicyNamespaceLister {
	return upstreamTLSPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// UpstreamTLSPolicyNamespaceLister helps list and get UpstreamTLSPolicies.
// All objects returned here must be treated as read-only.
type UpstreamTLSPolicyNamespaceLister interface {
	// List lists all UpstreamTLSPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.UpstreamTLSPolicy, err error)
	// Get retrieves the UpstreamTLSPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.UpstreamTLSPolicy, error)
	UpstreamTLSPolicyNamespaceListerExpansion
}

// upstreamTLSPolicyNamespaceLister implements the UpstreamTLSPolicyNamespaceLister
// interface.
type upstreamTLSPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all UpstreamTLSPolicies in the indexer for a given namespace.
func (s upstreamTLSPolicyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.UpstreamTLSPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.UpstreamTLSPolicy))
	})
	return ret, err
}

// Get retrieves the UpstreamTLSPolicy from the indexer for a given namespace and name.
func (s upstreamTLSPolicyNamespaceLister) Get(name string) (*v1alpha1.UpstreamTLSPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("upstreamtlspolicy"), name)
	}
	return obj.(*v1alpha1.UpstreamTLSPolicy), nil
}
