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

// FaultInjectionDelayApplyConfiguration represents an declarative configuration of the FaultInjectionDelay type for use
// with apply.
type FaultInjectionDelayApplyConfiguration struct {
	Percent *int32                                 `json:"percent,omitempty"`
	Fixed   *int64                                 `json:"fixed,omitempty"`
	Range   *FaultInjectionRangeApplyConfiguration `json:"range,omitempty"`
	Unit    *string                                `json:"unit,omitempty"`
}

// FaultInjectionDelayApplyConfiguration constructs an declarative configuration of the FaultInjectionDelay type for use with
// apply.
func FaultInjectionDelay() *FaultInjectionDelayApplyConfiguration {
	return &FaultInjectionDelayApplyConfiguration{}
}

// WithPercent sets the Percent field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Percent field is set to the value of the last call.
func (b *FaultInjectionDelayApplyConfiguration) WithPercent(value int32) *FaultInjectionDelayApplyConfiguration {
	b.Percent = &value
	return b
}

// WithFixed sets the Fixed field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Fixed field is set to the value of the last call.
func (b *FaultInjectionDelayApplyConfiguration) WithFixed(value int64) *FaultInjectionDelayApplyConfiguration {
	b.Fixed = &value
	return b
}

// WithRange sets the Range field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Range field is set to the value of the last call.
func (b *FaultInjectionDelayApplyConfiguration) WithRange(value *FaultInjectionRangeApplyConfiguration) *FaultInjectionDelayApplyConfiguration {
	b.Range = value
	return b
}

// WithUnit sets the Unit field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Unit field is set to the value of the last call.
func (b *FaultInjectionDelayApplyConfiguration) WithUnit(value string) *FaultInjectionDelayApplyConfiguration {
	b.Unit = &value
	return b
}
