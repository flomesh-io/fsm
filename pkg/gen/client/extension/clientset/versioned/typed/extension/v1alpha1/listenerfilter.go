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
	"context"

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// ListenerFiltersGetter has a method to return a ListenerFilterInterface.
// A group's client should implement this interface.
type ListenerFiltersGetter interface {
	ListenerFilters(namespace string) ListenerFilterInterface
}

// ListenerFilterInterface has methods to work with ListenerFilter resources.
type ListenerFilterInterface interface {
	Create(ctx context.Context, listenerFilter *v1alpha1.ListenerFilter, opts v1.CreateOptions) (*v1alpha1.ListenerFilter, error)
	Update(ctx context.Context, listenerFilter *v1alpha1.ListenerFilter, opts v1.UpdateOptions) (*v1alpha1.ListenerFilter, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, listenerFilter *v1alpha1.ListenerFilter, opts v1.UpdateOptions) (*v1alpha1.ListenerFilter, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ListenerFilter, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ListenerFilterList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ListenerFilter, err error)
	ListenerFilterExpansion
}

// listenerFilters implements ListenerFilterInterface
type listenerFilters struct {
	*gentype.ClientWithList[*v1alpha1.ListenerFilter, *v1alpha1.ListenerFilterList]
}

// newListenerFilters returns a ListenerFilters
func newListenerFilters(c *ExtensionV1alpha1Client, namespace string) *listenerFilters {
	return &listenerFilters{
		gentype.NewClientWithList[*v1alpha1.ListenerFilter, *v1alpha1.ListenerFilterList](
			"listenerfilters",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1alpha1.ListenerFilter { return &v1alpha1.ListenerFilter{} },
			func() *v1alpha1.ListenerFilterList { return &v1alpha1.ListenerFilterList{} }),
	}
}
