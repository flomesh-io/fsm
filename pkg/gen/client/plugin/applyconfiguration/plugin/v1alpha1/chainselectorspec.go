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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChainSelectorSpecApplyConfiguration represents an declarative configuration of the ChainSelectorSpec type for use
// with apply.
type ChainSelectorSpecApplyConfiguration struct {
	PodSelector       *v1.LabelSelector `json:"podSelector,omitempty"`
	NamespaceSelector *v1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// ChainSelectorSpecApplyConfiguration constructs an declarative configuration of the ChainSelectorSpec type for use with
// apply.
func ChainSelectorSpec() *ChainSelectorSpecApplyConfiguration {
	return &ChainSelectorSpecApplyConfiguration{}
}

// WithPodSelector sets the PodSelector field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodSelector field is set to the value of the last call.
func (b *ChainSelectorSpecApplyConfiguration) WithPodSelector(value v1.LabelSelector) *ChainSelectorSpecApplyConfiguration {
	b.PodSelector = &value
	return b
}

// WithNamespaceSelector sets the NamespaceSelector field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NamespaceSelector field is set to the value of the last call.
func (b *ChainSelectorSpecApplyConfiguration) WithNamespaceSelector(value v1.LabelSelector) *ChainSelectorSpecApplyConfiguration {
	b.NamespaceSelector = &value
	return b
}
