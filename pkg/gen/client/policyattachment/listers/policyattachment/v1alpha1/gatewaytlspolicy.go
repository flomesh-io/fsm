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

// GatewayTLSPolicyLister helps list GatewayTLSPolicies.
// All objects returned here must be treated as read-only.
type GatewayTLSPolicyLister interface {
	// List lists all GatewayTLSPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.GatewayTLSPolicy, err error)
	// GatewayTLSPolicies returns an object that can list and get GatewayTLSPolicies.
	GatewayTLSPolicies(namespace string) GatewayTLSPolicyNamespaceLister
	GatewayTLSPolicyListerExpansion
}

// gatewayTLSPolicyLister implements the GatewayTLSPolicyLister interface.
type gatewayTLSPolicyLister struct {
	indexer cache.Indexer
}

// NewGatewayTLSPolicyLister returns a new GatewayTLSPolicyLister.
func NewGatewayTLSPolicyLister(indexer cache.Indexer) GatewayTLSPolicyLister {
	return &gatewayTLSPolicyLister{indexer: indexer}
}

// List lists all GatewayTLSPolicies in the indexer.
func (s *gatewayTLSPolicyLister) List(selector labels.Selector) (ret []*v1alpha1.GatewayTLSPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.GatewayTLSPolicy))
	})
	return ret, err
}

// GatewayTLSPolicies returns an object that can list and get GatewayTLSPolicies.
func (s *gatewayTLSPolicyLister) GatewayTLSPolicies(namespace string) GatewayTLSPolicyNamespaceLister {
	return gatewayTLSPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// GatewayTLSPolicyNamespaceLister helps list and get GatewayTLSPolicies.
// All objects returned here must be treated as read-only.
type GatewayTLSPolicyNamespaceLister interface {
	// List lists all GatewayTLSPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.GatewayTLSPolicy, err error)
	// Get retrieves the GatewayTLSPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.GatewayTLSPolicy, error)
	GatewayTLSPolicyNamespaceListerExpansion
}

// gatewayTLSPolicyNamespaceLister implements the GatewayTLSPolicyNamespaceLister
// interface.
type gatewayTLSPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all GatewayTLSPolicies in the indexer for a given namespace.
func (s gatewayTLSPolicyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.GatewayTLSPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.GatewayTLSPolicy))
	})
	return ret, err
}

// Get retrieves the GatewayTLSPolicy from the indexer for a given namespace and name.
func (s gatewayTLSPolicyNamespaceLister) Get(name string) (*v1alpha1.GatewayTLSPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("gatewaytlspolicy"), name)
	}
	return obj.(*v1alpha1.GatewayTLSPolicy), nil
}
