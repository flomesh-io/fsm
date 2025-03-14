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

package v1alpha1

import (
	context "context"

	extensionv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// FilterDefinitionsGetter has a method to return a FilterDefinitionInterface.
// A group's client should implement this interface.
type FilterDefinitionsGetter interface {
	FilterDefinitions() FilterDefinitionInterface
}

// FilterDefinitionInterface has methods to work with FilterDefinition resources.
type FilterDefinitionInterface interface {
	Create(ctx context.Context, filterDefinition *extensionv1alpha1.FilterDefinition, opts v1.CreateOptions) (*extensionv1alpha1.FilterDefinition, error)
	Update(ctx context.Context, filterDefinition *extensionv1alpha1.FilterDefinition, opts v1.UpdateOptions) (*extensionv1alpha1.FilterDefinition, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, filterDefinition *extensionv1alpha1.FilterDefinition, opts v1.UpdateOptions) (*extensionv1alpha1.FilterDefinition, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*extensionv1alpha1.FilterDefinition, error)
	List(ctx context.Context, opts v1.ListOptions) (*extensionv1alpha1.FilterDefinitionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *extensionv1alpha1.FilterDefinition, err error)
	FilterDefinitionExpansion
}

// filterDefinitions implements FilterDefinitionInterface
type filterDefinitions struct {
	*gentype.ClientWithList[*extensionv1alpha1.FilterDefinition, *extensionv1alpha1.FilterDefinitionList]
}

// newFilterDefinitions returns a FilterDefinitions
func newFilterDefinitions(c *ExtensionV1alpha1Client) *filterDefinitions {
	return &filterDefinitions{
		gentype.NewClientWithList[*extensionv1alpha1.FilterDefinition, *extensionv1alpha1.FilterDefinitionList](
			"filterdefinitions",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *extensionv1alpha1.FilterDefinition { return &extensionv1alpha1.FilterDefinition{} },
			func() *extensionv1alpha1.FilterDefinitionList { return &extensionv1alpha1.FilterDefinitionList{} },
		),
	}
}
