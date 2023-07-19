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
	"context"

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNamespacedIngresses implements NamespacedIngressInterface
type FakeNamespacedIngresses struct {
	Fake *FakeFlomeshV1alpha1
	ns   string
}

var namespacedingressesResource = schema.GroupVersionResource{Group: "flomesh.io", Version: "v1alpha1", Resource: "namespacedingresses"}

var namespacedingressesKind = schema.GroupVersionKind{Group: "flomesh.io", Version: "v1alpha1", Kind: "NamespacedIngress"}

// Get takes name of the namespacedIngress, and returns the corresponding namespacedIngress object, and an error if there is any.
func (c *FakeNamespacedIngresses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NamespacedIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(namespacedingressesResource, c.ns, name), &v1alpha1.NamespacedIngress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NamespacedIngress), err
}

// List takes label and field selectors, and returns the list of NamespacedIngresses that match those selectors.
func (c *FakeNamespacedIngresses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NamespacedIngressList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(namespacedingressesResource, namespacedingressesKind, c.ns, opts), &v1alpha1.NamespacedIngressList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NamespacedIngressList{ListMeta: obj.(*v1alpha1.NamespacedIngressList).ListMeta}
	for _, item := range obj.(*v1alpha1.NamespacedIngressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested namespacedIngresses.
func (c *FakeNamespacedIngresses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(namespacedingressesResource, c.ns, opts))

}

// Create takes the representation of a namespacedIngress and creates it.  Returns the server's representation of the namespacedIngress, and an error, if there is any.
func (c *FakeNamespacedIngresses) Create(ctx context.Context, namespacedIngress *v1alpha1.NamespacedIngress, opts v1.CreateOptions) (result *v1alpha1.NamespacedIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(namespacedingressesResource, c.ns, namespacedIngress), &v1alpha1.NamespacedIngress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NamespacedIngress), err
}

// Update takes the representation of a namespacedIngress and updates it. Returns the server's representation of the namespacedIngress, and an error, if there is any.
func (c *FakeNamespacedIngresses) Update(ctx context.Context, namespacedIngress *v1alpha1.NamespacedIngress, opts v1.UpdateOptions) (result *v1alpha1.NamespacedIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(namespacedingressesResource, c.ns, namespacedIngress), &v1alpha1.NamespacedIngress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NamespacedIngress), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeNamespacedIngresses) UpdateStatus(ctx context.Context, namespacedIngress *v1alpha1.NamespacedIngress, opts v1.UpdateOptions) (*v1alpha1.NamespacedIngress, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(namespacedingressesResource, "status", c.ns, namespacedIngress), &v1alpha1.NamespacedIngress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NamespacedIngress), err
}

// Delete takes name of the namespacedIngress and deletes it. Returns an error if one occurs.
func (c *FakeNamespacedIngresses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(namespacedingressesResource, c.ns, name, opts), &v1alpha1.NamespacedIngress{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNamespacedIngresses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(namespacedingressesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.NamespacedIngressList{})
	return err
}

// Patch applies the patch and returns the patched namespacedIngress.
func (c *FakeNamespacedIngresses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NamespacedIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(namespacedingressesResource, c.ns, name, pt, data, subresources...), &v1alpha1.NamespacedIngress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NamespacedIngress), err
}
