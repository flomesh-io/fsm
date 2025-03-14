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

	apisextensionv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	versioned "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned"
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/extension/informers/externalversions/internalinterfaces"
	extensionv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/extension/listers/extension/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// FilterDefinitionInformer provides access to a shared informer and lister for
// FilterDefinitions.
type FilterDefinitionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() extensionv1alpha1.FilterDefinitionLister
}

type filterDefinitionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewFilterDefinitionInformer constructs a new informer for FilterDefinition type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilterDefinitionInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredFilterDefinitionInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredFilterDefinitionInformer constructs a new informer for FilterDefinition type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredFilterDefinitionInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().FilterDefinitions().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().FilterDefinitions().Watch(context.TODO(), options)
			},
		},
		&apisextensionv1alpha1.FilterDefinition{},
		resyncPeriod,
		indexers,
	)
}

func (f *filterDefinitionInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredFilterDefinitionInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *filterDefinitionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisextensionv1alpha1.FilterDefinition{}, f.defaultInformer)
}

func (f *filterDefinitionInformer) Lister() extensionv1alpha1.FilterDefinitionLister {
	return extensionv1alpha1.NewFilterDefinitionLister(f.Informer().GetIndexer())
}
