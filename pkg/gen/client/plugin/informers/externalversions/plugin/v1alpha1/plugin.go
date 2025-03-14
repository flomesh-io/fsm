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

// PluginInformer provides access to a shared informer and lister for
// Plugins.
type PluginInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() pluginv1alpha1.PluginLister
}

type pluginInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewPluginInformer constructs a new informer for Plugin type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPluginInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPluginInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredPluginInformer constructs a new informer for Plugin type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPluginInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PluginV1alpha1().Plugins().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PluginV1alpha1().Plugins().Watch(context.TODO(), options)
			},
		},
		&apispluginv1alpha1.Plugin{},
		resyncPeriod,
		indexers,
	)
}

func (f *pluginInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPluginInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *pluginInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apispluginv1alpha1.Plugin{}, f.defaultInformer)
}

func (f *pluginInformer) Lister() pluginv1alpha1.PluginLister {
	return pluginv1alpha1.NewPluginLister(f.Informer().GetIndexer())
}
