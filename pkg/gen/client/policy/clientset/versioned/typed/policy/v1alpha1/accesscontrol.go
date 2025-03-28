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

	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	scheme "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// AccessControlsGetter has a method to return a AccessControlInterface.
// A group's client should implement this interface.
type AccessControlsGetter interface {
	AccessControls(namespace string) AccessControlInterface
}

// AccessControlInterface has methods to work with AccessControl resources.
type AccessControlInterface interface {
	Create(ctx context.Context, accessControl *policyv1alpha1.AccessControl, opts v1.CreateOptions) (*policyv1alpha1.AccessControl, error)
	Update(ctx context.Context, accessControl *policyv1alpha1.AccessControl, opts v1.UpdateOptions) (*policyv1alpha1.AccessControl, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, accessControl *policyv1alpha1.AccessControl, opts v1.UpdateOptions) (*policyv1alpha1.AccessControl, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*policyv1alpha1.AccessControl, error)
	List(ctx context.Context, opts v1.ListOptions) (*policyv1alpha1.AccessControlList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *policyv1alpha1.AccessControl, err error)
	AccessControlExpansion
}

// accessControls implements AccessControlInterface
type accessControls struct {
	*gentype.ClientWithList[*policyv1alpha1.AccessControl, *policyv1alpha1.AccessControlList]
}

// newAccessControls returns a AccessControls
func newAccessControls(c *PolicyV1alpha1Client, namespace string) *accessControls {
	return &accessControls{
		gentype.NewClientWithList[*policyv1alpha1.AccessControl, *policyv1alpha1.AccessControlList](
			"accesscontrols",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *policyv1alpha1.AccessControl { return &policyv1alpha1.AccessControl{} },
			func() *policyv1alpha1.AccessControlList { return &policyv1alpha1.AccessControlList{} },
		),
	}
}
