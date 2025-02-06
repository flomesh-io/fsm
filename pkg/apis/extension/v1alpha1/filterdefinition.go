package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FilterDefinitionSpec defines the desired state of FilterDefinition
type FilterDefinitionSpec struct {
	// Scope is the scope of FilterDefinition
	// +optional
	// +kubebuilder:default=Route
	// +kubebuilder:validation:Enum=Route;Listener
	Scope *FilterScope `json:"scope,omitempty"`

	// Protocol is the protocol of FilterDefinition
	// +optional
	// +kubebuilder:default=http
	// +kubebuilder:validation:Enum=http;tcp;udp
	Protocol *FilterProtocol `json:"protocol,omitempty"`

	// Type is the type of the FilterDefinition in PascalCase, it should be unique within the namespace
	Type FilterType `json:"type"`

	// Script is the list of scripts to be executed
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Type=string
	Script string `json:"script"`
}

// FilterDefinition provides a way to configure FilterDefinitions for HTTP/HTTPS/GRPC/GRPCS/TCP protocols
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.flomesh.io/extension=Filter}
type FilterDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of FilterDefinition.
	Spec FilterDefinitionSpec `json:"spec,omitempty"`

	// Status defines the current state of FilterDefinition.
	Status FilterDefinitionStatus `json:"status,omitempty"`
}

// FilterDefinitionStatus defines the common attributes that all FilterDefinitions should include within
// their status.
type FilterDefinitionStatus struct {
	// Conditions describes the status of the FilterDefinition with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FilterDefinitionList contains a list of FilterDefinition
type FilterDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FilterDefinition `json:"items"`
}
