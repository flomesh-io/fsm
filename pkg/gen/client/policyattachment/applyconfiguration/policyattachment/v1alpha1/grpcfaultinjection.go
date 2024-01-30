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
	v1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// GRPCFaultInjectionApplyConfiguration represents an declarative configuration of the GRPCFaultInjection type for use
// with apply.
type GRPCFaultInjectionApplyConfiguration struct {
	Match  *v1alpha2.GRPCRouteMatch                `json:"match,omitempty"`
	Config *FaultInjectionConfigApplyConfiguration `json:"config,omitempty"`
}

// GRPCFaultInjectionApplyConfiguration constructs an declarative configuration of the GRPCFaultInjection type for use with
// apply.
func GRPCFaultInjection() *GRPCFaultInjectionApplyConfiguration {
	return &GRPCFaultInjectionApplyConfiguration{}
}

// WithMatch sets the Match field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Match field is set to the value of the last call.
func (b *GRPCFaultInjectionApplyConfiguration) WithMatch(value v1alpha2.GRPCRouteMatch) *GRPCFaultInjectionApplyConfiguration {
	b.Match = &value
	return b
}

// WithConfig sets the Config field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Config field is set to the value of the last call.
func (b *GRPCFaultInjectionApplyConfiguration) WithConfig(value *FaultInjectionConfigApplyConfiguration) *GRPCFaultInjectionApplyConfiguration {
	b.Config = value
	return b
}