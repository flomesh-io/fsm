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

// IPRestrictionsGetter has a method to return a IPRestrictionInterface.
// A group's client should implement this interface.
type IPRestrictionsGetter interface {
	IPRestrictions(namespace string) IPRestrictionInterface
}

// IPRestrictionInterface has methods to work with IPRestriction resources.
type IPRestrictionInterface interface {
	Create(ctx context.Context, iPRestriction *extensionv1alpha1.IPRestriction, opts v1.CreateOptions) (*extensionv1alpha1.IPRestriction, error)
	Update(ctx context.Context, iPRestriction *extensionv1alpha1.IPRestriction, opts v1.UpdateOptions) (*extensionv1alpha1.IPRestriction, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, iPRestriction *extensionv1alpha1.IPRestriction, opts v1.UpdateOptions) (*extensionv1alpha1.IPRestriction, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*extensionv1alpha1.IPRestriction, error)
	List(ctx context.Context, opts v1.ListOptions) (*extensionv1alpha1.IPRestrictionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *extensionv1alpha1.IPRestriction, err error)
	IPRestrictionExpansion
}

// iPRestrictions implements IPRestrictionInterface
type iPRestrictions struct {
	*gentype.ClientWithList[*extensionv1alpha1.IPRestriction, *extensionv1alpha1.IPRestrictionList]
}

// newIPRestrictions returns a IPRestrictions
func newIPRestrictions(c *ExtensionV1alpha1Client, namespace string) *iPRestrictions {
	return &iPRestrictions{
		gentype.NewClientWithList[*extensionv1alpha1.IPRestriction, *extensionv1alpha1.IPRestrictionList](
			"iprestrictions",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *extensionv1alpha1.IPRestriction { return &extensionv1alpha1.IPRestriction{} },
			func() *extensionv1alpha1.IPRestrictionList { return &extensionv1alpha1.IPRestrictionList{} },
		),
	}
}
