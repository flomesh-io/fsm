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

// ConnectionSettingsSpecApplyConfiguration represents an declarative configuration of the ConnectionSettingsSpec type for use
// with apply.
type ConnectionSettingsSpecApplyConfiguration struct {
	TCP  *TCPConnectionSettingsApplyConfiguration  `json:"tcp,omitempty"`
	HTTP *HTTPConnectionSettingsApplyConfiguration `json:"http,omitempty"`
}

// ConnectionSettingsSpecApplyConfiguration constructs an declarative configuration of the ConnectionSettingsSpec type for use with
// apply.
func ConnectionSettingsSpec() *ConnectionSettingsSpecApplyConfiguration {
	return &ConnectionSettingsSpecApplyConfiguration{}
}

// WithTCP sets the TCP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TCP field is set to the value of the last call.
func (b *ConnectionSettingsSpecApplyConfiguration) WithTCP(value *TCPConnectionSettingsApplyConfiguration) *ConnectionSettingsSpecApplyConfiguration {
	b.TCP = value
	return b
}

// WithHTTP sets the HTTP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the HTTP field is set to the value of the last call.
func (b *ConnectionSettingsSpecApplyConfiguration) WithHTTP(value *HTTPConnectionSettingsApplyConfiguration) *ConnectionSettingsSpecApplyConfiguration {
	b.HTTP = value
	return b
}
