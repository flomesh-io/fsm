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

// PluginSpecApplyConfiguration represents an declarative configuration of the PluginSpec type for use
// with apply.
type PluginSpecApplyConfiguration struct {
	Priority *float32 `json:"priority,omitempty"`
	Script   *string  `json:"pipyscript,omitempty"`
}

// PluginSpecApplyConfiguration constructs an declarative configuration of the PluginSpec type for use with
// apply.
func PluginSpec() *PluginSpecApplyConfiguration {
	return &PluginSpecApplyConfiguration{}
}

// WithPriority sets the Priority field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Priority field is set to the value of the last call.
func (b *PluginSpecApplyConfiguration) WithPriority(value float32) *PluginSpecApplyConfiguration {
	b.Priority = &value
	return b
}

// WithScript sets the Script field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Script field is set to the value of the last call.
func (b *PluginSpecApplyConfiguration) WithScript(value string) *PluginSpecApplyConfiguration {
	b.Script = &value
	return b
}