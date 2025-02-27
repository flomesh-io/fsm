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

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// ConsulConnectorsGetter has a method to return a ConsulConnectorInterface.
// A group's client should implement this interface.
type ConsulConnectorsGetter interface {
	ConsulConnectors(namespace string) ConsulConnectorInterface
}

// ConsulConnectorInterface has methods to work with ConsulConnector resources.
type ConsulConnectorInterface interface {
	Create(ctx context.Context, consulConnector *v1alpha1.ConsulConnector, opts v1.CreateOptions) (*v1alpha1.ConsulConnector, error)
	Update(ctx context.Context, consulConnector *v1alpha1.ConsulConnector, opts v1.UpdateOptions) (*v1alpha1.ConsulConnector, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, consulConnector *v1alpha1.ConsulConnector, opts v1.UpdateOptions) (*v1alpha1.ConsulConnector, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ConsulConnector, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ConsulConnectorList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ConsulConnector, err error)
	ConsulConnectorExpansion
}

// consulConnectors implements ConsulConnectorInterface
type consulConnectors struct {
	*gentype.ClientWithList[*v1alpha1.ConsulConnector, *v1alpha1.ConsulConnectorList]
}

// newConsulConnectors returns a ConsulConnectors
func newConsulConnectors(c *ConnectorV1alpha1Client, namespace string) *consulConnectors {
	return &consulConnectors{
		gentype.NewClientWithList[*v1alpha1.ConsulConnector, *v1alpha1.ConsulConnectorList](
			"consulconnectors",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1alpha1.ConsulConnector { return &v1alpha1.ConsulConnector{} },
			func() *v1alpha1.ConsulConnectorList { return &v1alpha1.ConsulConnectorList{} }),
	}
}
