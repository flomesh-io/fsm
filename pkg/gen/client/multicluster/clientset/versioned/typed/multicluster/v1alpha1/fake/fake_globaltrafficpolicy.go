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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	multiclusterv1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned/typed/multicluster/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeGlobalTrafficPolicies implements GlobalTrafficPolicyInterface
type fakeGlobalTrafficPolicies struct {
	*gentype.FakeClientWithList[*v1alpha1.GlobalTrafficPolicy, *v1alpha1.GlobalTrafficPolicyList]
	Fake *FakeMulticlusterV1alpha1
}

func newFakeGlobalTrafficPolicies(fake *FakeMulticlusterV1alpha1, namespace string) multiclusterv1alpha1.GlobalTrafficPolicyInterface {
	return &fakeGlobalTrafficPolicies{
		gentype.NewFakeClientWithList[*v1alpha1.GlobalTrafficPolicy, *v1alpha1.GlobalTrafficPolicyList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("globaltrafficpolicies"),
			v1alpha1.SchemeGroupVersion.WithKind("GlobalTrafficPolicy"),
			func() *v1alpha1.GlobalTrafficPolicy { return &v1alpha1.GlobalTrafficPolicy{} },
			func() *v1alpha1.GlobalTrafficPolicyList { return &v1alpha1.GlobalTrafficPolicyList{} },
			func(dst, src *v1alpha1.GlobalTrafficPolicyList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.GlobalTrafficPolicyList) []*v1alpha1.GlobalTrafficPolicy {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.GlobalTrafficPolicyList, items []*v1alpha1.GlobalTrafficPolicy) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
