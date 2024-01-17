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

package v1alpha2

// VaultProviderSpecApplyConfiguration represents an declarative configuration of the VaultProviderSpec type for use
// with apply.
type VaultProviderSpecApplyConfiguration struct {
	Host     *string                           `json:"host,omitempty"`
	Port     *int                              `json:"port,omitempty"`
	Role     *string                           `json:"role,omitempty"`
	Protocol *string                           `json:"protocol,omitempty"`
	Token    *VaultTokenSpecApplyConfiguration `json:"token,omitempty"`
}

// VaultProviderSpecApplyConfiguration constructs an declarative configuration of the VaultProviderSpec type for use with
// apply.
func VaultProviderSpec() *VaultProviderSpecApplyConfiguration {
	return &VaultProviderSpecApplyConfiguration{}
}

// WithHost sets the Host field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Host field is set to the value of the last call.
func (b *VaultProviderSpecApplyConfiguration) WithHost(value string) *VaultProviderSpecApplyConfiguration {
	b.Host = &value
	return b
}

// WithPort sets the Port field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Port field is set to the value of the last call.
func (b *VaultProviderSpecApplyConfiguration) WithPort(value int) *VaultProviderSpecApplyConfiguration {
	b.Port = &value
	return b
}

// WithRole sets the Role field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Role field is set to the value of the last call.
func (b *VaultProviderSpecApplyConfiguration) WithRole(value string) *VaultProviderSpecApplyConfiguration {
	b.Role = &value
	return b
}

// WithProtocol sets the Protocol field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Protocol field is set to the value of the last call.
func (b *VaultProviderSpecApplyConfiguration) WithProtocol(value string) *VaultProviderSpecApplyConfiguration {
	b.Protocol = &value
	return b
}

// WithToken sets the Token field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Token field is set to the value of the last call.
func (b *VaultProviderSpecApplyConfiguration) WithToken(value *VaultTokenSpecApplyConfiguration) *VaultProviderSpecApplyConfiguration {
	b.Token = value
	return b
}
