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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	extensionv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned/typed/extension/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeListenerFilters implements ListenerFilterInterface
type fakeListenerFilters struct {
	*gentype.FakeClientWithList[*v1alpha1.ListenerFilter, *v1alpha1.ListenerFilterList]
	Fake *FakeExtensionV1alpha1
}

func newFakeListenerFilters(fake *FakeExtensionV1alpha1, namespace string) extensionv1alpha1.ListenerFilterInterface {
	return &fakeListenerFilters{
		gentype.NewFakeClientWithList[*v1alpha1.ListenerFilter, *v1alpha1.ListenerFilterList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("listenerfilters"),
			v1alpha1.SchemeGroupVersion.WithKind("ListenerFilter"),
			func() *v1alpha1.ListenerFilter { return &v1alpha1.ListenerFilter{} },
			func() *v1alpha1.ListenerFilterList { return &v1alpha1.ListenerFilterList{} },
			func(dst, src *v1alpha1.ListenerFilterList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.ListenerFilterList) []*v1alpha1.ListenerFilter {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.ListenerFilterList, items []*v1alpha1.ListenerFilter) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
