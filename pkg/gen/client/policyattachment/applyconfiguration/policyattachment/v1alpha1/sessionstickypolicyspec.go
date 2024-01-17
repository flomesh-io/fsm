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
	v1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// SessionStickyPolicySpecApplyConfiguration represents an declarative configuration of the SessionStickyPolicySpec type for use
// with apply.
type SessionStickyPolicySpecApplyConfiguration struct {
	TargetRef     *v1alpha2.PolicyTargetReference        `json:"targetRef,omitempty"`
	Ports         []PortSessionStickyApplyConfiguration  `json:"ports,omitempty"`
	DefaultConfig *SessionStickyConfigApplyConfiguration `json:"config,omitempty"`
}

// SessionStickyPolicySpecApplyConfiguration constructs an declarative configuration of the SessionStickyPolicySpec type for use with
// apply.
func SessionStickyPolicySpec() *SessionStickyPolicySpecApplyConfiguration {
	return &SessionStickyPolicySpecApplyConfiguration{}
}

// WithTargetRef sets the TargetRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TargetRef field is set to the value of the last call.
func (b *SessionStickyPolicySpecApplyConfiguration) WithTargetRef(value v1alpha2.PolicyTargetReference) *SessionStickyPolicySpecApplyConfiguration {
	b.TargetRef = &value
	return b
}

// WithPorts adds the given value to the Ports field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Ports field.
func (b *SessionStickyPolicySpecApplyConfiguration) WithPorts(values ...*PortSessionStickyApplyConfiguration) *SessionStickyPolicySpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithPorts")
		}
		b.Ports = append(b.Ports, *values[i])
	}
	return b
}

// WithDefaultConfig sets the DefaultConfig field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DefaultConfig field is set to the value of the last call.
func (b *SessionStickyPolicySpecApplyConfiguration) WithDefaultConfig(value *SessionStickyConfigApplyConfiguration) *SessionStickyPolicySpecApplyConfiguration {
	b.DefaultConfig = value
	return b
}
