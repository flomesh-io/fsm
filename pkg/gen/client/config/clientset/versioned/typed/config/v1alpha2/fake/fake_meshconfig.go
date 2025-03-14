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
	v1alpha2 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha2"
	configv1alpha2 "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/typed/config/v1alpha2"
	gentype "k8s.io/client-go/gentype"
)

// fakeMeshConfigs implements MeshConfigInterface
type fakeMeshConfigs struct {
	*gentype.FakeClientWithList[*v1alpha2.MeshConfig, *v1alpha2.MeshConfigList]
	Fake *FakeConfigV1alpha2
}

func newFakeMeshConfigs(fake *FakeConfigV1alpha2, namespace string) configv1alpha2.MeshConfigInterface {
	return &fakeMeshConfigs{
		gentype.NewFakeClientWithList[*v1alpha2.MeshConfig, *v1alpha2.MeshConfigList](
			fake.Fake,
			namespace,
			v1alpha2.SchemeGroupVersion.WithResource("meshconfigs"),
			v1alpha2.SchemeGroupVersion.WithKind("MeshConfig"),
			func() *v1alpha2.MeshConfig { return &v1alpha2.MeshConfig{} },
			func() *v1alpha2.MeshConfigList { return &v1alpha2.MeshConfigList{} },
			func(dst, src *v1alpha2.MeshConfigList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha2.MeshConfigList) []*v1alpha2.MeshConfig { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha2.MeshConfigList, items []*v1alpha2.MeshConfig) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
