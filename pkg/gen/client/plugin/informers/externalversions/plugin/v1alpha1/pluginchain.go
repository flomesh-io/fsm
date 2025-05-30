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

package v1alpha1

import (
	context "context"
	time "time"

	apispluginv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/plugin/v1alpha1"
	versioned "github.com/flomesh-io/fsm/pkg/gen/client/plugin/clientset/versioned"
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/plugin/informers/externalversions/internalinterfaces"
	pluginv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/plugin/listers/plugin/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// PluginChainInformer provides access to a shared informer and lister for
// PluginChains.
type PluginChainInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() pluginv1alpha1.PluginChainLister
}

type pluginChainInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewPluginChainInformer constructs a new informer for PluginChain type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPluginChainInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPluginChainInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredPluginChainInformer constructs a new informer for PluginChain type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPluginChainInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PluginV1alpha1().PluginChains(namespace).List(context.Background(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PluginV1alpha1().PluginChains(namespace).Watch(context.Background(), options)
			},
			ListWithContextFunc: func(ctx context.Context, options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PluginV1alpha1().PluginChains(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PluginV1alpha1().PluginChains(namespace).Watch(ctx, options)
			},
		},
		&apispluginv1alpha1.PluginChain{},
		resyncPeriod,
		indexers,
	)
}

func (f *pluginChainInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPluginChainInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *pluginChainInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apispluginv1alpha1.PluginChain{}, f.defaultInformer)
}

func (f *pluginChainInformer) Lister() pluginv1alpha1.PluginChainLister {
	return pluginv1alpha1.NewPluginChainLister(f.Informer().GetIndexer())
}
