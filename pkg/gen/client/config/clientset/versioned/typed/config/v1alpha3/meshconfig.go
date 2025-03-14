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

package v1alpha3

import (
	context "context"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// MeshConfigsGetter has a method to return a MeshConfigInterface.
// A group's client should implement this interface.
type MeshConfigsGetter interface {
	MeshConfigs(namespace string) MeshConfigInterface
}

// MeshConfigInterface has methods to work with MeshConfig resources.
type MeshConfigInterface interface {
	Create(ctx context.Context, meshConfig *configv1alpha3.MeshConfig, opts v1.CreateOptions) (*configv1alpha3.MeshConfig, error)
	Update(ctx context.Context, meshConfig *configv1alpha3.MeshConfig, opts v1.UpdateOptions) (*configv1alpha3.MeshConfig, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*configv1alpha3.MeshConfig, error)
	List(ctx context.Context, opts v1.ListOptions) (*configv1alpha3.MeshConfigList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *configv1alpha3.MeshConfig, err error)
	MeshConfigExpansion
}

// meshConfigs implements MeshConfigInterface
type meshConfigs struct {
	*gentype.ClientWithList[*configv1alpha3.MeshConfig, *configv1alpha3.MeshConfigList]
}

// newMeshConfigs returns a MeshConfigs
func newMeshConfigs(c *ConfigV1alpha3Client, namespace string) *meshConfigs {
	return &meshConfigs{
		gentype.NewClientWithList[*configv1alpha3.MeshConfig, *configv1alpha3.MeshConfigList](
			"meshconfigs",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *configv1alpha3.MeshConfig { return &configv1alpha3.MeshConfig{} },
			func() *configv1alpha3.MeshConfigList { return &configv1alpha3.MeshConfigList{} },
		),
	}
}
