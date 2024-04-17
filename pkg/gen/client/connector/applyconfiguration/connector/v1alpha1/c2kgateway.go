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

// C2KGatewayApplyConfiguration represents an declarative configuration of the C2KGateway type for use
// with apply.
type C2KGatewayApplyConfiguration struct {
	Enable        *bool `json:"enable,omitempty"`
	MultiGateways *bool `json:"multiGateways,omitempty"`
}

// C2KGatewayApplyConfiguration constructs an declarative configuration of the C2KGateway type for use with
// apply.
func C2KGateway() *C2KGatewayApplyConfiguration {
	return &C2KGatewayApplyConfiguration{}
}

// WithEnable sets the Enable field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enable field is set to the value of the last call.
func (b *C2KGatewayApplyConfiguration) WithEnable(value bool) *C2KGatewayApplyConfiguration {
	b.Enable = &value
	return b
}

// WithMultiGateways sets the MultiGateways field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MultiGateways field is set to the value of the last call.
func (b *C2KGatewayApplyConfiguration) WithMultiGateways(value bool) *C2KGatewayApplyConfiguration {
	b.MultiGateways = &value
	return b
}
