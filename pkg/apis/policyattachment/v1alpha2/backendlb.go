/*
Copyright 2024 The Kubernetes Authors.

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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api,shortName=blbpolicy
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.networking.k8s.io/policy=Direct}

// BackendLBPolicy provides a way to define load balancing rules
// for a backend.
type BackendLBPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of BackendLBPolicy.
	Spec BackendLBPolicySpec `json:"spec"`

	// Status defines the current state of BackendLBPolicy.
	Status gwv1alpha2.PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackendLBPolicyList contains a list of BackendLBPolicies
type BackendLBPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackendLBPolicy `json:"items"`
}

// BackendLBPolicySpec defines the desired state of
// BackendLBPolicy.
// Note: there is no Override or Default policy configuration.
type BackendLBPolicySpec struct {
	// TargetRef identifies an API object to apply policy to.
	// Currently, Backends (i.e. Service, ServiceImport, or any
	// implementation-specific backendRef) are the only valid API
	// target references.
	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []gwv1alpha2.LocalPolicyTargetReference `json:"targetRefs"`

	// SessionPersistence defines and configures session persistence
	// for the backend.
	//
	// Support: Extended
	//
	// +optional
	SessionPersistence *gwv1alpha2.SessionPersistence `json:"sessionPersistence,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum=RoundRobin;LeastLoad
	// Algorithm is the load balancing algorithm, default is RoundRobin
	Algorithm *LoadBalancerAlgorithm `json:"algorithm,omitempty"`
}
