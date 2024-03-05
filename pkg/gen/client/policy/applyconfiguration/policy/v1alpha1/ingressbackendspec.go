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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

// IngressBackendSpecApplyConfiguration represents an declarative configuration of the IngressBackendSpec type for use
// with apply.
type IngressBackendSpecApplyConfiguration struct {
	Backends []BackendSpecApplyConfiguration       `json:"backends,omitempty"`
	Sources  []IngressSourceSpecApplyConfiguration `json:"sources,omitempty"`
	Matches  []v1.TypedLocalObjectReference        `json:"matches,omitempty"`
}

// IngressBackendSpecApplyConfiguration constructs an declarative configuration of the IngressBackendSpec type for use with
// apply.
func IngressBackendSpec() *IngressBackendSpecApplyConfiguration {
	return &IngressBackendSpecApplyConfiguration{}
}

// WithBackends adds the given value to the Backends field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Backends field.
func (b *IngressBackendSpecApplyConfiguration) WithBackends(values ...*BackendSpecApplyConfiguration) *IngressBackendSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithBackends")
		}
		b.Backends = append(b.Backends, *values[i])
	}
	return b
}

// WithSources adds the given value to the Sources field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Sources field.
func (b *IngressBackendSpecApplyConfiguration) WithSources(values ...*IngressSourceSpecApplyConfiguration) *IngressBackendSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSources")
		}
		b.Sources = append(b.Sources, *values[i])
	}
	return b
}

// WithMatches adds the given value to the Matches field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Matches field.
func (b *IngressBackendSpecApplyConfiguration) WithMatches(values ...v1.TypedLocalObjectReference) *IngressBackendSpecApplyConfiguration {
	for i := range values {
		b.Matches = append(b.Matches, values[i])
	}
	return b
}