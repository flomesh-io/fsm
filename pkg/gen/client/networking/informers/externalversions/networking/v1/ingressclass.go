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

package v1

import (
	context "context"
	time "time"

	apisnetworkingv1 "github.com/flomesh-io/fsm/pkg/apis/networking/v1"
	versioned "github.com/flomesh-io/fsm/pkg/gen/client/networking/clientset/versioned"
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/networking/informers/externalversions/internalinterfaces"
	networkingv1 "github.com/flomesh-io/fsm/pkg/gen/client/networking/listers/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// IngressClassInformer provides access to a shared informer and lister for
// IngressClasses.
type IngressClassInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() networkingv1.IngressClassLister
}

type ingressClassInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewIngressClassInformer constructs a new informer for IngressClass type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewIngressClassInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredIngressClassInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredIngressClassInformer constructs a new informer for IngressClass type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredIngressClassInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkingV1().IngressClasses().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkingV1().IngressClasses().Watch(context.TODO(), options)
			},
		},
		&apisnetworkingv1.IngressClass{},
		resyncPeriod,
		indexers,
	)
}

func (f *ingressClassInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredIngressClassInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *ingressClassInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisnetworkingv1.IngressClass{}, f.defaultInformer)
}

func (f *ingressClassInformer) Lister() networkingv1.IngressClassLister {
	return networkingv1.NewIngressClassLister(f.Informer().GetIndexer())
}
