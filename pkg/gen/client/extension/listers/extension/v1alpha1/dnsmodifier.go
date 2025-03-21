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
	extensionv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// DNSModifierLister helps list DNSModifiers.
// All objects returned here must be treated as read-only.
type DNSModifierLister interface {
	// List lists all DNSModifiers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*extensionv1alpha1.DNSModifier, err error)
	// DNSModifiers returns an object that can list and get DNSModifiers.
	DNSModifiers(namespace string) DNSModifierNamespaceLister
	DNSModifierListerExpansion
}

// dNSModifierLister implements the DNSModifierLister interface.
type dNSModifierLister struct {
	listers.ResourceIndexer[*extensionv1alpha1.DNSModifier]
}

// NewDNSModifierLister returns a new DNSModifierLister.
func NewDNSModifierLister(indexer cache.Indexer) DNSModifierLister {
	return &dNSModifierLister{listers.New[*extensionv1alpha1.DNSModifier](indexer, extensionv1alpha1.Resource("dnsmodifier"))}
}

// DNSModifiers returns an object that can list and get DNSModifiers.
func (s *dNSModifierLister) DNSModifiers(namespace string) DNSModifierNamespaceLister {
	return dNSModifierNamespaceLister{listers.NewNamespaced[*extensionv1alpha1.DNSModifier](s.ResourceIndexer, namespace)}
}

// DNSModifierNamespaceLister helps list and get DNSModifiers.
// All objects returned here must be treated as read-only.
type DNSModifierNamespaceLister interface {
	// List lists all DNSModifiers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*extensionv1alpha1.DNSModifier, err error)
	// Get retrieves the DNSModifier from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*extensionv1alpha1.DNSModifier, error)
	DNSModifierNamespaceListerExpansion
}

// dNSModifierNamespaceLister implements the DNSModifierNamespaceLister
// interface.
type dNSModifierNamespaceLister struct {
	listers.ResourceIndexer[*extensionv1alpha1.DNSModifier]
}
