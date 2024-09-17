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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/gen/client/extension/clientset/versioned/typed/extension/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeExtensionV1alpha1 struct {
	*testing.Fake
}

func (c *FakeExtensionV1alpha1) CircuitBreakers(namespace string) v1alpha1.CircuitBreakerInterface {
	return &FakeCircuitBreakers{c, namespace}
}

func (c *FakeExtensionV1alpha1) FaultInjections(namespace string) v1alpha1.FaultInjectionInterface {
	return &FakeFaultInjections{c, namespace}
}

func (c *FakeExtensionV1alpha1) Filters(namespace string) v1alpha1.FilterInterface {
	return &FakeFilters{c, namespace}
}

func (c *FakeExtensionV1alpha1) FilterDefinitions() v1alpha1.FilterDefinitionInterface {
	return &FakeFilterDefinitions{c}
}

func (c *FakeExtensionV1alpha1) HTTPLogs(namespace string) v1alpha1.HTTPLogInterface {
	return &FakeHTTPLogs{c, namespace}
}

func (c *FakeExtensionV1alpha1) ListenerFilters(namespace string) v1alpha1.ListenerFilterInterface {
	return &FakeListenerFilters{c, namespace}
}

func (c *FakeExtensionV1alpha1) Metricses(namespace string) v1alpha1.MetricsInterface {
	return &FakeMetricses{c, namespace}
}

func (c *FakeExtensionV1alpha1) RateLimits(namespace string) v1alpha1.RateLimitInterface {
	return &FakeRateLimits{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExtensionV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
