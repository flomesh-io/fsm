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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned/typed/policy/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakePolicyV1alpha1 struct {
	*testing.Fake
}

func (c *FakePolicyV1alpha1) AccessCerts(namespace string) v1alpha1.AccessCertInterface {
	return &FakeAccessCerts{c, namespace}
}

func (c *FakePolicyV1alpha1) AccessControls(namespace string) v1alpha1.AccessControlInterface {
	return &FakeAccessControls{c, namespace}
}

func (c *FakePolicyV1alpha1) Egresses(namespace string) v1alpha1.EgressInterface {
	return &FakeEgresses{c, namespace}
}

func (c *FakePolicyV1alpha1) EgressGateways(namespace string) v1alpha1.EgressGatewayInterface {
	return &FakeEgressGateways{c, namespace}
}

func (c *FakePolicyV1alpha1) IngressBackends(namespace string) v1alpha1.IngressBackendInterface {
	return &FakeIngressBackends{c, namespace}
}

func (c *FakePolicyV1alpha1) Isolations(namespace string) v1alpha1.IsolationInterface {
	return &FakeIsolations{c, namespace}
}

func (c *FakePolicyV1alpha1) Retries(namespace string) v1alpha1.RetryInterface {
	return &FakeRetries{c, namespace}
}

func (c *FakePolicyV1alpha1) TrafficWarmups(namespace string) v1alpha1.TrafficWarmupInterface {
	return &FakeTrafficWarmups{c, namespace}
}

func (c *FakePolicyV1alpha1) UpstreamTrafficSettings(namespace string) v1alpha1.UpstreamTrafficSettingInterface {
	return &FakeUpstreamTrafficSettings{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakePolicyV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
