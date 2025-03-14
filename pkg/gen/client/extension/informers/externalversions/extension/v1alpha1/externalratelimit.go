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

// ExternalRateLimitInformer provides access to a shared informer and lister for
// ExternalRateLimits.
type ExternalRateLimitInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() extensionv1alpha1.ExternalRateLimitLister
}

type externalRateLimitInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewExternalRateLimitInformer constructs a new informer for ExternalRateLimit type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewExternalRateLimitInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredExternalRateLimitInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredExternalRateLimitInformer constructs a new informer for ExternalRateLimit type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredExternalRateLimitInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().ExternalRateLimits(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionV1alpha1().ExternalRateLimits(namespace).Watch(context.TODO(), options)
			},
		},
		&apisextensionv1alpha1.ExternalRateLimit{},
		resyncPeriod,
		indexers,
	)
}

func (f *externalRateLimitInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredExternalRateLimitInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *externalRateLimitInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisextensionv1alpha1.ExternalRateLimit{}, f.defaultInformer)
}

func (f *externalRateLimitInformer) Lister() extensionv1alpha1.ExternalRateLimitLister {
	return extensionv1alpha1.NewExternalRateLimitLister(f.Informer().GetIndexer())
}
