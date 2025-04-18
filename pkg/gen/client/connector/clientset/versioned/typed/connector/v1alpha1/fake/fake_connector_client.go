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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/connector/clientset/versioned/typed/connector/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeConnectorV1alpha1 struct {
	*testing.Fake
}

func (c *FakeConnectorV1alpha1) ConsulConnectors(namespace string) v1alpha1.ConsulConnectorInterface {
	return newFakeConsulConnectors(c, namespace)
}

func (c *FakeConnectorV1alpha1) EurekaConnectors(namespace string) v1alpha1.EurekaConnectorInterface {
	return newFakeEurekaConnectors(c, namespace)
}

func (c *FakeConnectorV1alpha1) GatewayConnectors(namespace string) v1alpha1.GatewayConnectorInterface {
	return newFakeGatewayConnectors(c, namespace)
}

func (c *FakeConnectorV1alpha1) MachineConnectors(namespace string) v1alpha1.MachineConnectorInterface {
	return newFakeMachineConnectors(c, namespace)
}

func (c *FakeConnectorV1alpha1) NacosConnectors(namespace string) v1alpha1.NacosConnectorInterface {
	return newFakeNacosConnectors(c, namespace)
}

func (c *FakeConnectorV1alpha1) ZookeeperConnectors(namespace string) v1alpha1.ZookeeperConnectorInterface {
	return newFakeZookeeperConnectors(c, namespace)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeConnectorV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
