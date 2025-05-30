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

// CircuitBreakerInformer provides access to a shared informer and lister for
// CircuitBreakers.
type CircuitBreakerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() extensionv1alpha1.CircuitBreakerLister
}

type circuitBreakerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewCircuitBreakerInformer constructs a new informer for CircuitBreaker type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCircuitBreakerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCircuitBreakerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredCircuitBreakerInformer constructs a new informer for CircuitBreaker type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCircuitBreakerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().CircuitBreakers(namespace).List(context.Background(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().CircuitBreakers(namespace).Watch(context.Background(), options)
			},
			ListWithContextFunc: func(ctx context.Context, options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().CircuitBreakers(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().CircuitBreakers(namespace).Watch(ctx, options)
			},
		},
		&apisextensionv1alpha1.CircuitBreaker{},
		resyncPeriod,
		indexers,
	)
}

func (f *circuitBreakerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCircuitBreakerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *circuitBreakerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisextensionv1alpha1.CircuitBreaker{}, f.defaultInformer)
}

func (f *circuitBreakerInformer) Lister() extensionv1alpha1.CircuitBreakerLister {
	return extensionv1alpha1.NewCircuitBreakerLister(f.Informer().GetIndexer())
}
