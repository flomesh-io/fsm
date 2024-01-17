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
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// HostnameFaultInjectionApplyConfiguration represents an declarative configuration of the HostnameFaultInjection type for use
// with apply.
type HostnameFaultInjectionApplyConfiguration struct {
	Hostname *v1.Hostname                            `json:"hostname,omitempty"`
	Config   *FaultInjectionConfigApplyConfiguration `json:"config,omitempty"`
}

// HostnameFaultInjectionApplyConfiguration constructs an declarative configuration of the HostnameFaultInjection type for use with
// apply.
func HostnameFaultInjection() *HostnameFaultInjectionApplyConfiguration {
	return &HostnameFaultInjectionApplyConfiguration{}
}

// WithHostname sets the Hostname field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Hostname field is set to the value of the last call.
func (b *HostnameFaultInjectionApplyConfiguration) WithHostname(value v1.Hostname) *HostnameFaultInjectionApplyConfiguration {
	b.Hostname = &value
	return b
}

// WithConfig sets the Config field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Config field is set to the value of the last call.
func (b *HostnameFaultInjectionApplyConfiguration) WithConfig(value *FaultInjectionConfigApplyConfiguration) *HostnameFaultInjectionApplyConfiguration {
	b.Config = value
	return b
}
