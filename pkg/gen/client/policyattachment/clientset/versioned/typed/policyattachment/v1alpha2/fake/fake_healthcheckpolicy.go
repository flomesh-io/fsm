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

	v1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeHealthCheckPolicies implements HealthCheckPolicyInterface
type FakeHealthCheckPolicies struct {
	Fake *FakeGatewayV1alpha2
	ns   string
}

var healthcheckpoliciesResource = v1alpha2.SchemeGroupVersion.WithResource("healthcheckpolicies")

var healthcheckpoliciesKind = v1alpha2.SchemeGroupVersion.WithKind("HealthCheckPolicy")

// Get takes name of the healthCheckPolicy, and returns the corresponding healthCheckPolicy object, and an error if there is any.
func (c *FakeHealthCheckPolicies) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.HealthCheckPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(healthcheckpoliciesResource, c.ns, name), &v1alpha2.HealthCheckPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealthCheckPolicy), err
}

// List takes label and field selectors, and returns the list of HealthCheckPolicies that match those selectors.
func (c *FakeHealthCheckPolicies) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.HealthCheckPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(healthcheckpoliciesResource, healthcheckpoliciesKind, c.ns, opts), &v1alpha2.HealthCheckPolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.HealthCheckPolicyList{ListMeta: obj.(*v1alpha2.HealthCheckPolicyList).ListMeta}
	for _, item := range obj.(*v1alpha2.HealthCheckPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested healthCheckPolicies.
func (c *FakeHealthCheckPolicies) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(healthcheckpoliciesResource, c.ns, opts))

}

// Create takes the representation of a healthCheckPolicy and creates it.  Returns the server's representation of the healthCheckPolicy, and an error, if there is any.
func (c *FakeHealthCheckPolicies) Create(ctx context.Context, healthCheckPolicy *v1alpha2.HealthCheckPolicy, opts v1.CreateOptions) (result *v1alpha2.HealthCheckPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(healthcheckpoliciesResource, c.ns, healthCheckPolicy), &v1alpha2.HealthCheckPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealthCheckPolicy), err
}

// Update takes the representation of a healthCheckPolicy and updates it. Returns the server's representation of the healthCheckPolicy, and an error, if there is any.
func (c *FakeHealthCheckPolicies) Update(ctx context.Context, healthCheckPolicy *v1alpha2.HealthCheckPolicy, opts v1.UpdateOptions) (result *v1alpha2.HealthCheckPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(healthcheckpoliciesResource, c.ns, healthCheckPolicy), &v1alpha2.HealthCheckPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealthCheckPolicy), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeHealthCheckPolicies) UpdateStatus(ctx context.Context, healthCheckPolicy *v1alpha2.HealthCheckPolicy, opts v1.UpdateOptions) (*v1alpha2.HealthCheckPolicy, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(healthcheckpoliciesResource, "status", c.ns, healthCheckPolicy), &v1alpha2.HealthCheckPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealthCheckPolicy), err
}

// Delete takes name of the healthCheckPolicy and deletes it. Returns an error if one occurs.
func (c *FakeHealthCheckPolicies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(healthcheckpoliciesResource, c.ns, name, opts), &v1alpha2.HealthCheckPolicy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeHealthCheckPolicies) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(healthcheckpoliciesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha2.HealthCheckPolicyList{})
	return err
}

// Patch applies the patch and returns the patched healthCheckPolicy.
func (c *FakeHealthCheckPolicies) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.HealthCheckPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(healthcheckpoliciesResource, c.ns, name, pt, data, subresources...), &v1alpha2.HealthCheckPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealthCheckPolicy), err
}
