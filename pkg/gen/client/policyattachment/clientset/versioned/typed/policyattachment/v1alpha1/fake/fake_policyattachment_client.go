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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/policyattachment/clientset/versioned/typed/policyattachment/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeGatewayV1alpha1 struct {
	*testing.Fake
}

func (c *FakeGatewayV1alpha1) AccessControlPolicies(namespace string) v1alpha1.AccessControlPolicyInterface {
	return &FakeAccessControlPolicies{c, namespace}
}

func (c *FakeGatewayV1alpha1) CircuitBreakingPolicies(namespace string) v1alpha1.CircuitBreakingPolicyInterface {
	return &FakeCircuitBreakingPolicies{c, namespace}
}

func (c *FakeGatewayV1alpha1) LoadBalancerPolicies(namespace string) v1alpha1.LoadBalancerPolicyInterface {
	return &FakeLoadBalancerPolicies{c, namespace}
}

func (c *FakeGatewayV1alpha1) RateLimitPolicies(namespace string) v1alpha1.RateLimitPolicyInterface {
	return &FakeRateLimitPolicies{c, namespace}
}

func (c *FakeGatewayV1alpha1) SessionStickyPolicies(namespace string) v1alpha1.SessionStickyPolicyInterface {
	return &FakeSessionStickyPolicies{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeGatewayV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
