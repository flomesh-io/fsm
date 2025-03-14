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

// fakeCircuitBreakers implements CircuitBreakerInterface
type fakeCircuitBreakers struct {
	*gentype.FakeClientWithList[*v1alpha1.CircuitBreaker, *v1alpha1.CircuitBreakerList]
	Fake *FakeExtensionV1alpha1
}

func newFakeCircuitBreakers(fake *FakeExtensionV1alpha1, namespace string) extensionv1alpha1.CircuitBreakerInterface {
	return &fakeCircuitBreakers{
		gentype.NewFakeClientWithList[*v1alpha1.CircuitBreaker, *v1alpha1.CircuitBreakerList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("circuitbreakers"),
			v1alpha1.SchemeGroupVersion.WithKind("CircuitBreaker"),
			func() *v1alpha1.CircuitBreaker { return &v1alpha1.CircuitBreaker{} },
			func() *v1alpha1.CircuitBreakerList { return &v1alpha1.CircuitBreakerList{} },
			func(dst, src *v1alpha1.CircuitBreakerList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.CircuitBreakerList) []*v1alpha1.CircuitBreaker {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.CircuitBreakerList, items []*v1alpha1.CircuitBreaker) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
