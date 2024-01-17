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

// AccessControlConfigApplyConfiguration represents an declarative configuration of the AccessControlConfig type for use
// with apply.
type AccessControlConfigApplyConfiguration struct {
	Blacklist  []string `json:"blacklist,omitempty"`
	Whitelist  []string `json:"whitelist,omitempty"`
	EnableXFF  *bool    `json:"enableXFF,omitempty"`
	StatusCode *int32   `json:"statusCode,omitempty"`
	Message    *string  `json:"message,omitempty"`
}

// AccessControlConfigApplyConfiguration constructs an declarative configuration of the AccessControlConfig type for use with
// apply.
func AccessControlConfig() *AccessControlConfigApplyConfiguration {
	return &AccessControlConfigApplyConfiguration{}
}

// WithBlacklist adds the given value to the Blacklist field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Blacklist field.
func (b *AccessControlConfigApplyConfiguration) WithBlacklist(values ...string) *AccessControlConfigApplyConfiguration {
	for i := range values {
		b.Blacklist = append(b.Blacklist, values[i])
	}
	return b
}

// WithWhitelist adds the given value to the Whitelist field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Whitelist field.
func (b *AccessControlConfigApplyConfiguration) WithWhitelist(values ...string) *AccessControlConfigApplyConfiguration {
	for i := range values {
		b.Whitelist = append(b.Whitelist, values[i])
	}
	return b
}

// WithEnableXFF sets the EnableXFF field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EnableXFF field is set to the value of the last call.
func (b *AccessControlConfigApplyConfiguration) WithEnableXFF(value bool) *AccessControlConfigApplyConfiguration {
	b.EnableXFF = &value
	return b
}

// WithStatusCode sets the StatusCode field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StatusCode field is set to the value of the last call.
func (b *AccessControlConfigApplyConfiguration) WithStatusCode(value int32) *AccessControlConfigApplyConfiguration {
	b.StatusCode = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *AccessControlConfigApplyConfiguration) WithMessage(value string) *AccessControlConfigApplyConfiguration {
	b.Message = &value
	return b
}
