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

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeZookeeperConnectors implements ZookeeperConnectorInterface
type FakeZookeeperConnectors struct {
	Fake *FakeConnectorV1alpha1
	ns   string
}

var zookeeperconnectorsResource = v1alpha1.SchemeGroupVersion.WithResource("zookeeperconnectors")

var zookeeperconnectorsKind = v1alpha1.SchemeGroupVersion.WithKind("ZookeeperConnector")

// Get takes name of the zookeeperConnector, and returns the corresponding zookeeperConnector object, and an error if there is any.
func (c *FakeZookeeperConnectors) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ZookeeperConnector, err error) {
	emptyResult := &v1alpha1.ZookeeperConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(zookeeperconnectorsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.ZookeeperConnector), err
}

// List takes label and field selectors, and returns the list of ZookeeperConnectors that match those selectors.
func (c *FakeZookeeperConnectors) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ZookeeperConnectorList, err error) {
	emptyResult := &v1alpha1.ZookeeperConnectorList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(zookeeperconnectorsResource, zookeeperconnectorsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ZookeeperConnectorList{ListMeta: obj.(*v1alpha1.ZookeeperConnectorList).ListMeta}
	for _, item := range obj.(*v1alpha1.ZookeeperConnectorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested zookeeperConnectors.
func (c *FakeZookeeperConnectors) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(zookeeperconnectorsResource, c.ns, opts))

}

// Create takes the representation of a zookeeperConnector and creates it.  Returns the server's representation of the zookeeperConnector, and an error, if there is any.
func (c *FakeZookeeperConnectors) Create(ctx context.Context, zookeeperConnector *v1alpha1.ZookeeperConnector, opts v1.CreateOptions) (result *v1alpha1.ZookeeperConnector, err error) {
	emptyResult := &v1alpha1.ZookeeperConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(zookeeperconnectorsResource, c.ns, zookeeperConnector, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.ZookeeperConnector), err
}

// Update takes the representation of a zookeeperConnector and updates it. Returns the server's representation of the zookeeperConnector, and an error, if there is any.
func (c *FakeZookeeperConnectors) Update(ctx context.Context, zookeeperConnector *v1alpha1.ZookeeperConnector, opts v1.UpdateOptions) (result *v1alpha1.ZookeeperConnector, err error) {
	emptyResult := &v1alpha1.ZookeeperConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(zookeeperconnectorsResource, c.ns, zookeeperConnector, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.ZookeeperConnector), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeZookeeperConnectors) UpdateStatus(ctx context.Context, zookeeperConnector *v1alpha1.ZookeeperConnector, opts v1.UpdateOptions) (result *v1alpha1.ZookeeperConnector, err error) {
	emptyResult := &v1alpha1.ZookeeperConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(zookeeperconnectorsResource, "status", c.ns, zookeeperConnector, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.ZookeeperConnector), err
}

// Delete takes name of the zookeeperConnector and deletes it. Returns an error if one occurs.
func (c *FakeZookeeperConnectors) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(zookeeperconnectorsResource, c.ns, name, opts), &v1alpha1.ZookeeperConnector{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeZookeeperConnectors) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(zookeeperconnectorsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ZookeeperConnectorList{})
	return err
}

// Patch applies the patch and returns the patched zookeeperConnector.
func (c *FakeZookeeperConnectors) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ZookeeperConnector, err error) {
	emptyResult := &v1alpha1.ZookeeperConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(zookeeperconnectorsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.ZookeeperConnector), err
}
