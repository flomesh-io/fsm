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

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/multicluster/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// ClustersGetter has a method to return a ClusterInterface.
// A group's client should implement this interface.
type ClustersGetter interface {
	Clusters() ClusterInterface
}

// ClusterInterface has methods to work with Cluster resources.
type ClusterInterface interface {
	Create(ctx context.Context, cluster *v1alpha1.Cluster, opts v1.CreateOptions) (*v1alpha1.Cluster, error)
	Update(ctx context.Context, cluster *v1alpha1.Cluster, opts v1.UpdateOptions) (*v1alpha1.Cluster, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, cluster *v1alpha1.Cluster, opts v1.UpdateOptions) (*v1alpha1.Cluster, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Cluster, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ClusterList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Cluster, err error)
	ClusterExpansion
}

// clusters implements ClusterInterface
type clusters struct {
	*gentype.ClientWithList[*v1alpha1.Cluster, *v1alpha1.ClusterList]
}

// newClusters returns a Clusters
func newClusters(c *MulticlusterV1alpha1Client) *clusters {
	return &clusters{
		gentype.NewClientWithList[*v1alpha1.Cluster, *v1alpha1.ClusterList](
			"clusters",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v1alpha1.Cluster { return &v1alpha1.Cluster{} },
			func() *v1alpha1.ClusterList { return &v1alpha1.ClusterList{} }),
	}
}
