package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// FilterSpec defines the desired state of Filter
type FilterSpec struct {
	// Type is the type of the Filter in PascalCase, it should be unique within the namespace
	Type FilterType `json:"type"`

	// +optional
	// +nullable
	// +kubebuilder:validation:Type=object
	// DefinitionRef is the reference to the FilterDefinition
	DefinitionRef *gwv1.LocalObjectReference `json:"definitionRef"`

	// +optional
	// +nullable
	// +kubebuilder:validation:Type=object
	// ConfigRef is the reference to the Configurations
	ConfigRef *gwv1.LocalObjectReference `json:"configRef,omitempty"`
}

// Filter provides a way to configure filters for HTTP/HTTPS/GRPC/GRPCS/TCP protocols
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.flomesh.io/extension=Filter}
type Filter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Filter.
	Spec FilterSpec `json:"spec,omitempty"`

	// Status defines the current state of Filter.
	Status FilterStatus `json:"status,omitempty"`
}

// FilterStatus defines the common attributes that all filters should include within
// their status.
type FilterStatus struct {
	// Conditions describes the status of the Filter with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FilterList contains a list of Filter
type FilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Filter `json:"items"`
}
