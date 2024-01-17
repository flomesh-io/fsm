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

package v1alpha2

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha2 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha2"
	configv1alpha2 "github.com/flomesh-io/fsm/pkg/gen/client/config/applyconfiguration/config/v1alpha2"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MeshRootCertificatesGetter has a method to return a MeshRootCertificateInterface.
// A group's client should implement this interface.
type MeshRootCertificatesGetter interface {
	MeshRootCertificates(namespace string) MeshRootCertificateInterface
}

// MeshRootCertificateInterface has methods to work with MeshRootCertificate resources.
type MeshRootCertificateInterface interface {
	Create(ctx context.Context, meshRootCertificate *v1alpha2.MeshRootCertificate, opts v1.CreateOptions) (*v1alpha2.MeshRootCertificate, error)
	Update(ctx context.Context, meshRootCertificate *v1alpha2.MeshRootCertificate, opts v1.UpdateOptions) (*v1alpha2.MeshRootCertificate, error)
	UpdateStatus(ctx context.Context, meshRootCertificate *v1alpha2.MeshRootCertificate, opts v1.UpdateOptions) (*v1alpha2.MeshRootCertificate, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha2.MeshRootCertificate, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha2.MeshRootCertificateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.MeshRootCertificate, err error)
	Apply(ctx context.Context, meshRootCertificate *configv1alpha2.MeshRootCertificateApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.MeshRootCertificate, err error)
	ApplyStatus(ctx context.Context, meshRootCertificate *configv1alpha2.MeshRootCertificateApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.MeshRootCertificate, err error)
	MeshRootCertificateExpansion
}

// meshRootCertificates implements MeshRootCertificateInterface
type meshRootCertificates struct {
	client rest.Interface
	ns     string
}

// newMeshRootCertificates returns a MeshRootCertificates
func newMeshRootCertificates(c *ConfigV1alpha2Client, namespace string) *meshRootCertificates {
	return &meshRootCertificates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the meshRootCertificate, and returns the corresponding meshRootCertificate object, and an error if there is any.
func (c *meshRootCertificates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.MeshRootCertificate, err error) {
	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MeshRootCertificates that match those selectors.
func (c *meshRootCertificates) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.MeshRootCertificateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha2.MeshRootCertificateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested meshRootCertificates.
func (c *meshRootCertificates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a meshRootCertificate and creates it.  Returns the server's representation of the meshRootCertificate, and an error, if there is any.
func (c *meshRootCertificates) Create(ctx context.Context, meshRootCertificate *v1alpha2.MeshRootCertificate, opts v1.CreateOptions) (result *v1alpha2.MeshRootCertificate, err error) {
	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(meshRootCertificate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a meshRootCertificate and updates it. Returns the server's representation of the meshRootCertificate, and an error, if there is any.
func (c *meshRootCertificates) Update(ctx context.Context, meshRootCertificate *v1alpha2.MeshRootCertificate, opts v1.UpdateOptions) (result *v1alpha2.MeshRootCertificate, err error) {
	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(meshRootCertificate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(meshRootCertificate).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *meshRootCertificates) UpdateStatus(ctx context.Context, meshRootCertificate *v1alpha2.MeshRootCertificate, opts v1.UpdateOptions) (result *v1alpha2.MeshRootCertificate, err error) {
	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(meshRootCertificate.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(meshRootCertificate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the meshRootCertificate and deletes it. Returns an error if one occurs.
func (c *meshRootCertificates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *meshRootCertificates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("meshrootcertificates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched meshRootCertificate.
func (c *meshRootCertificates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.MeshRootCertificate, err error) {
	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied meshRootCertificate.
func (c *meshRootCertificates) Apply(ctx context.Context, meshRootCertificate *configv1alpha2.MeshRootCertificateApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.MeshRootCertificate, err error) {
	if meshRootCertificate == nil {
		return nil, fmt.Errorf("meshRootCertificate provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(meshRootCertificate)
	if err != nil {
		return nil, err
	}
	name := meshRootCertificate.Name
	if name == nil {
		return nil, fmt.Errorf("meshRootCertificate.Name must be provided to Apply")
	}
	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *meshRootCertificates) ApplyStatus(ctx context.Context, meshRootCertificate *configv1alpha2.MeshRootCertificateApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.MeshRootCertificate, err error) {
	if meshRootCertificate == nil {
		return nil, fmt.Errorf("meshRootCertificate provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(meshRootCertificate)
	if err != nil {
		return nil, err
	}

	name := meshRootCertificate.Name
	if name == nil {
		return nil, fmt.Errorf("meshRootCertificate.Name must be provided to Apply")
	}

	result = &v1alpha2.MeshRootCertificate{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("meshrootcertificates").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
