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
	"context"
	time "time"

	policyattachmentv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	versioned "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned"
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/listers/policyattachment/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AccessControlPolicyInformer provides access to a shared informer and lister for
// AccessControlPolicies.
type AccessControlPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.AccessControlPolicyLister
}

type accessControlPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAccessControlPolicyInformer constructs a new informer for AccessControlPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAccessControlPolicyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAccessControlPolicyInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAccessControlPolicyInformer constructs a new informer for AccessControlPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAccessControlPolicyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.GatewayV1alpha1().AccessControlPolicies(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.GatewayV1alpha1().AccessControlPolicies(namespace).Watch(context.TODO(), options)
			},
		},
		&policyattachmentv1alpha1.AccessControlPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *accessControlPolicyInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAccessControlPolicyInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *accessControlPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&policyattachmentv1alpha1.AccessControlPolicy{}, f.defaultInformer)
}

func (f *accessControlPolicyInformer) Lister() v1alpha1.AccessControlPolicyLister {
	return v1alpha1.NewAccessControlPolicyLister(f.Informer().GetIndexer())
}
