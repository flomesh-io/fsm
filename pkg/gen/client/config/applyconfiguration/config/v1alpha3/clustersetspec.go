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

package v1alpha3

// ClusterSetSpecApplyConfiguration represents an declarative configuration of the ClusterSetSpec type for use
// with apply.
type ClusterSetSpecApplyConfiguration struct {
	IsManaged       *bool                                   `json:"isManaged,omitempty"`
	UID             *string                                 `json:"uid,omitempty"`
	Region          *string                                 `json:"region,omitempty"`
	Zone            *string                                 `json:"zone,omitempty"`
	Group           *string                                 `json:"group,omitempty"`
	Name            *string                                 `json:"name,omitempty"`
	ControlPlaneUID *string                                 `json:"controlPlaneUID,omitempty"`
	Properties      []ClusterPropertySpecApplyConfiguration `json:"properties,omitempty"`
}

// ClusterSetSpecApplyConfiguration constructs an declarative configuration of the ClusterSetSpec type for use with
// apply.
func ClusterSetSpec() *ClusterSetSpecApplyConfiguration {
	return &ClusterSetSpecApplyConfiguration{}
}

// WithIsManaged sets the IsManaged field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IsManaged field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithIsManaged(value bool) *ClusterSetSpecApplyConfiguration {
	b.IsManaged = &value
	return b
}

// WithUID sets the UID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UID field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithUID(value string) *ClusterSetSpecApplyConfiguration {
	b.UID = &value
	return b
}

// WithRegion sets the Region field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Region field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithRegion(value string) *ClusterSetSpecApplyConfiguration {
	b.Region = &value
	return b
}

// WithZone sets the Zone field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Zone field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithZone(value string) *ClusterSetSpecApplyConfiguration {
	b.Zone = &value
	return b
}

// WithGroup sets the Group field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Group field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithGroup(value string) *ClusterSetSpecApplyConfiguration {
	b.Group = &value
	return b
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithName(value string) *ClusterSetSpecApplyConfiguration {
	b.Name = &value
	return b
}

// WithControlPlaneUID sets the ControlPlaneUID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ControlPlaneUID field is set to the value of the last call.
func (b *ClusterSetSpecApplyConfiguration) WithControlPlaneUID(value string) *ClusterSetSpecApplyConfiguration {
	b.ControlPlaneUID = &value
	return b
}

// WithProperties adds the given value to the Properties field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Properties field.
func (b *ClusterSetSpecApplyConfiguration) WithProperties(values ...*ClusterPropertySpecApplyConfiguration) *ClusterSetSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithProperties")
		}
		b.Properties = append(b.Properties, *values[i])
	}
	return b
}
