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

// GatewayStatusApplyConfiguration represents an declarative configuration of the GatewayStatus type for use
// with apply.
type GatewayStatusApplyConfiguration struct {
	CurrentStatus *string `json:"currentStatus,omitempty"`
	Reason        *string `json:"reason,omitempty"`
}

// GatewayStatusApplyConfiguration constructs an declarative configuration of the GatewayStatus type for use with
// apply.
func GatewayStatus() *GatewayStatusApplyConfiguration {
	return &GatewayStatusApplyConfiguration{}
}

// WithCurrentStatus sets the CurrentStatus field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CurrentStatus field is set to the value of the last call.
func (b *GatewayStatusApplyConfiguration) WithCurrentStatus(value string) *GatewayStatusApplyConfiguration {
	b.CurrentStatus = &value
	return b
}

// WithReason sets the Reason field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Reason field is set to the value of the last call.
func (b *GatewayStatusApplyConfiguration) WithReason(value string) *GatewayStatusApplyConfiguration {
	b.Reason = &value
	return b
}
