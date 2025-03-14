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
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha2

import (
	context "context"
	time "time"

	apispolicyattachmentv1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	versioned "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/informers/externalversions/internalinterfaces"
	policyattachmentv1alpha2 "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/listers/policyattachment/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// RouteRuleFilterPolicyInformer provides access to a shared informer and lister for
// RouteRuleFilterPolicies.
type RouteRuleFilterPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() policyattachmentv1alpha2.RouteRuleFilterPolicyLister
}

type routeRuleFilterPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewRouteRuleFilterPolicyInformer constructs a new informer for RouteRuleFilterPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRouteRuleFilterPolicyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRouteRuleFilterPolicyInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredRouteRuleFilterPolicyInformer constructs a new informer for RouteRuleFilterPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRouteRuleFilterPolicyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.GatewayV1alpha2().RouteRuleFilterPolicies(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.GatewayV1alpha2().RouteRuleFilterPolicies(namespace).Watch(context.TODO(), options)
			},
		},
		&apispolicyattachmentv1alpha2.RouteRuleFilterPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *routeRuleFilterPolicyInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRouteRuleFilterPolicyInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *routeRuleFilterPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apispolicyattachmentv1alpha2.RouteRuleFilterPolicy{}, f.defaultInformer)
}

func (f *routeRuleFilterPolicyInformer) Lister() policyattachmentv1alpha2.RouteRuleFilterPolicyLister {
	return policyattachmentv1alpha2.NewRouteRuleFilterPolicyLister(f.Informer().GetIndexer())
}
