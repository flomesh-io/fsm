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

// FakeGatewayConnectors implements GatewayConnectorInterface
type FakeGatewayConnectors struct {
	Fake *FakeConnectorV1alpha1
	ns   string
}

var gatewayconnectorsResource = v1alpha1.SchemeGroupVersion.WithResource("gatewayconnectors")

var gatewayconnectorsKind = v1alpha1.SchemeGroupVersion.WithKind("GatewayConnector")

// Get takes name of the gatewayConnector, and returns the corresponding gatewayConnector object, and an error if there is any.
func (c *FakeGatewayConnectors) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.GatewayConnector, err error) {
	emptyResult := &v1alpha1.GatewayConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(gatewayconnectorsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayConnector), err
}

// List takes label and field selectors, and returns the list of GatewayConnectors that match those selectors.
func (c *FakeGatewayConnectors) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.GatewayConnectorList, err error) {
	emptyResult := &v1alpha1.GatewayConnectorList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(gatewayconnectorsResource, gatewayconnectorsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.GatewayConnectorList{ListMeta: obj.(*v1alpha1.GatewayConnectorList).ListMeta}
	for _, item := range obj.(*v1alpha1.GatewayConnectorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested gatewayConnectors.
func (c *FakeGatewayConnectors) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(gatewayconnectorsResource, c.ns, opts))

}

// Create takes the representation of a gatewayConnector and creates it.  Returns the server's representation of the gatewayConnector, and an error, if there is any.
func (c *FakeGatewayConnectors) Create(ctx context.Context, gatewayConnector *v1alpha1.GatewayConnector, opts v1.CreateOptions) (result *v1alpha1.GatewayConnector, err error) {
	emptyResult := &v1alpha1.GatewayConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(gatewayconnectorsResource, c.ns, gatewayConnector, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayConnector), err
}

// Update takes the representation of a gatewayConnector and updates it. Returns the server's representation of the gatewayConnector, and an error, if there is any.
func (c *FakeGatewayConnectors) Update(ctx context.Context, gatewayConnector *v1alpha1.GatewayConnector, opts v1.UpdateOptions) (result *v1alpha1.GatewayConnector, err error) {
	emptyResult := &v1alpha1.GatewayConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(gatewayconnectorsResource, c.ns, gatewayConnector, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayConnector), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGatewayConnectors) UpdateStatus(ctx context.Context, gatewayConnector *v1alpha1.GatewayConnector, opts v1.UpdateOptions) (result *v1alpha1.GatewayConnector, err error) {
	emptyResult := &v1alpha1.GatewayConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(gatewayconnectorsResource, "status", c.ns, gatewayConnector, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayConnector), err
}

// Delete takes name of the gatewayConnector and deletes it. Returns an error if one occurs.
func (c *FakeGatewayConnectors) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(gatewayconnectorsResource, c.ns, name, opts), &v1alpha1.GatewayConnector{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGatewayConnectors) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(gatewayconnectorsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.GatewayConnectorList{})
	return err
}

// Patch applies the patch and returns the patched gatewayConnector.
func (c *FakeGatewayConnectors) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.GatewayConnector, err error) {
	emptyResult := &v1alpha1.GatewayConnector{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(gatewayconnectorsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayConnector), err
}
