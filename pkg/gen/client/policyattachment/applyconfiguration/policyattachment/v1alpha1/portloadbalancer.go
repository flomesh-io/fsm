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
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// PortLoadBalancerApplyConfiguration represents an declarative configuration of the PortLoadBalancer type for use
// with apply.
type PortLoadBalancerApplyConfiguration struct {
	Port *v1.PortNumber             `json:"port,omitempty"`
	Type *v1alpha1.LoadBalancerType `json:"type,omitempty"`
}

// PortLoadBalancerApplyConfiguration constructs an declarative configuration of the PortLoadBalancer type for use with
// apply.
func PortLoadBalancer() *PortLoadBalancerApplyConfiguration {
	return &PortLoadBalancerApplyConfiguration{}
}

// WithPort sets the Port field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Port field is set to the value of the last call.
func (b *PortLoadBalancerApplyConfiguration) WithPort(value v1.PortNumber) *PortLoadBalancerApplyConfiguration {
	b.Port = &value
	return b
}

// WithType sets the Type field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Type field is set to the value of the last call.
func (b *PortLoadBalancerApplyConfiguration) WithType(value v1alpha1.LoadBalancerType) *PortLoadBalancerApplyConfiguration {
	b.Type = &value
	return b
}