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

	apisconfigv1alpha2 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha2"
	versioned "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/config/informers/externalversions/internalinterfaces"
	configv1alpha2 "github.com/flomesh-io/fsm/pkg/gen/client/config/listers/config/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// MeshConfigInformer provides access to a shared informer and lister for
// MeshConfigs.
type MeshConfigInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() configv1alpha2.MeshConfigLister
}

type meshConfigInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewMeshConfigInformer constructs a new informer for MeshConfig type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMeshConfigInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMeshConfigInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredMeshConfigInformer constructs a new informer for MeshConfig type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMeshConfigInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigV1alpha2().MeshConfigs(namespace).List(context.Background(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigV1alpha2().MeshConfigs(namespace).Watch(context.Background(), options)
			},
			ListWithContextFunc: func(ctx context.Context, options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigV1alpha2().MeshConfigs(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigV1alpha2().MeshConfigs(namespace).Watch(ctx, options)
			},
		},
		&apisconfigv1alpha2.MeshConfig{},
		resyncPeriod,
		indexers,
	)
}

func (f *meshConfigInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMeshConfigInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *meshConfigInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisconfigv1alpha2.MeshConfig{}, f.defaultInformer)
}

func (f *meshConfigInformer) Lister() configv1alpha2.MeshConfigLister {
	return configv1alpha2.NewMeshConfigLister(f.Informer().GetIndexer())
}
