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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned/typed/policy/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeIsolations implements IsolationInterface
type fakeIsolations struct {
	*gentype.FakeClientWithList[*v1alpha1.Isolation, *v1alpha1.IsolationList]
	Fake *FakePolicyV1alpha1
}

func newFakeIsolations(fake *FakePolicyV1alpha1, namespace string) policyv1alpha1.IsolationInterface {
	return &fakeIsolations{
		gentype.NewFakeClientWithList[*v1alpha1.Isolation, *v1alpha1.IsolationList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("isolations"),
			v1alpha1.SchemeGroupVersion.WithKind("Isolation"),
			func() *v1alpha1.Isolation { return &v1alpha1.Isolation{} },
			func() *v1alpha1.IsolationList { return &v1alpha1.IsolationList{} },
			func(dst, src *v1alpha1.IsolationList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.IsolationList) []*v1alpha1.Isolation { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha1.IsolationList, items []*v1alpha1.Isolation) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
