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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	namespacedingressv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned/typed/namespacedingress/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeNamespacedIngresses implements NamespacedIngressInterface
type fakeNamespacedIngresses struct {
	*gentype.FakeClientWithList[*v1alpha1.NamespacedIngress, *v1alpha1.NamespacedIngressList]
	Fake *FakeNetworkingV1alpha1
}

func newFakeNamespacedIngresses(fake *FakeNetworkingV1alpha1, namespace string) namespacedingressv1alpha1.NamespacedIngressInterface {
	return &fakeNamespacedIngresses{
		gentype.NewFakeClientWithList[*v1alpha1.NamespacedIngress, *v1alpha1.NamespacedIngressList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("namespacedingresses"),
			v1alpha1.SchemeGroupVersion.WithKind("NamespacedIngress"),
			func() *v1alpha1.NamespacedIngress { return &v1alpha1.NamespacedIngress{} },
			func() *v1alpha1.NamespacedIngressList { return &v1alpha1.NamespacedIngressList{} },
			func(dst, src *v1alpha1.NamespacedIngressList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.NamespacedIngressList) []*v1alpha1.NamespacedIngress {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.NamespacedIngressList, items []*v1alpha1.NamespacedIngress) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
