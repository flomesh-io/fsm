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

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeUpstreamTrafficSettings implements UpstreamTrafficSettingInterface
type FakeUpstreamTrafficSettings struct {
	Fake *FakePolicyV1alpha1
	ns   string
}

var upstreamtrafficsettingsResource = v1alpha1.SchemeGroupVersion.WithResource("upstreamtrafficsettings")

var upstreamtrafficsettingsKind = v1alpha1.SchemeGroupVersion.WithKind("UpstreamTrafficSetting")

// Get takes name of the upstreamTrafficSetting, and returns the corresponding upstreamTrafficSetting object, and an error if there is any.
func (c *FakeUpstreamTrafficSettings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.UpstreamTrafficSetting, err error) {
	emptyResult := &v1alpha1.UpstreamTrafficSetting{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(upstreamtrafficsettingsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.UpstreamTrafficSetting), err
}

// List takes label and field selectors, and returns the list of UpstreamTrafficSettings that match those selectors.
func (c *FakeUpstreamTrafficSettings) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.UpstreamTrafficSettingList, err error) {
	emptyResult := &v1alpha1.UpstreamTrafficSettingList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(upstreamtrafficsettingsResource, upstreamtrafficsettingsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.UpstreamTrafficSettingList{ListMeta: obj.(*v1alpha1.UpstreamTrafficSettingList).ListMeta}
	for _, item := range obj.(*v1alpha1.UpstreamTrafficSettingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested upstreamTrafficSettings.
func (c *FakeUpstreamTrafficSettings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(upstreamtrafficsettingsResource, c.ns, opts))

}

// Create takes the representation of a upstreamTrafficSetting and creates it.  Returns the server's representation of the upstreamTrafficSetting, and an error, if there is any.
func (c *FakeUpstreamTrafficSettings) Create(ctx context.Context, upstreamTrafficSetting *v1alpha1.UpstreamTrafficSetting, opts v1.CreateOptions) (result *v1alpha1.UpstreamTrafficSetting, err error) {
	emptyResult := &v1alpha1.UpstreamTrafficSetting{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(upstreamtrafficsettingsResource, c.ns, upstreamTrafficSetting, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.UpstreamTrafficSetting), err
}

// Update takes the representation of a upstreamTrafficSetting and updates it. Returns the server's representation of the upstreamTrafficSetting, and an error, if there is any.
func (c *FakeUpstreamTrafficSettings) Update(ctx context.Context, upstreamTrafficSetting *v1alpha1.UpstreamTrafficSetting, opts v1.UpdateOptions) (result *v1alpha1.UpstreamTrafficSetting, err error) {
	emptyResult := &v1alpha1.UpstreamTrafficSetting{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(upstreamtrafficsettingsResource, c.ns, upstreamTrafficSetting, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.UpstreamTrafficSetting), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeUpstreamTrafficSettings) UpdateStatus(ctx context.Context, upstreamTrafficSetting *v1alpha1.UpstreamTrafficSetting, opts v1.UpdateOptions) (result *v1alpha1.UpstreamTrafficSetting, err error) {
	emptyResult := &v1alpha1.UpstreamTrafficSetting{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(upstreamtrafficsettingsResource, "status", c.ns, upstreamTrafficSetting, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.UpstreamTrafficSetting), err
}

// Delete takes name of the upstreamTrafficSetting and deletes it. Returns an error if one occurs.
func (c *FakeUpstreamTrafficSettings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(upstreamtrafficsettingsResource, c.ns, name, opts), &v1alpha1.UpstreamTrafficSetting{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUpstreamTrafficSettings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(upstreamtrafficsettingsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.UpstreamTrafficSettingList{})
	return err
}

// Patch applies the patch and returns the patched upstreamTrafficSetting.
func (c *FakeUpstreamTrafficSettings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.UpstreamTrafficSetting, err error) {
	emptyResult := &v1alpha1.UpstreamTrafficSetting{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(upstreamtrafficsettingsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.UpstreamTrafficSetting), err
}
