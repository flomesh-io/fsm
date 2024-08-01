package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FilterSpec defines the desired state of Filter
type FilterSpec struct {
	// +optional
	// +kubebuilder:validation:MaxItems=16
	// TargetRefs is the references to the target resources to which the filter is applied
	TargetRefs []LocalPolicyTargetReferenceWithPort `json:"targetRefs"`

	// Scope is the scope of filter
	// +optional
	// +kubebuilder:default=Route
	// +kubebuilder:validation:Enum=Route;Listener
	Scope *FilterScope `json:"scope"`

	// Protocol is the protocol of filter
	// +optional
	// +kubebuilder:default=http
	// +kubebuilder:validation:Enum=http;tcp
	Protocol *FilterProtocol `json:"protocol"`

	// Type is the type of the filter in PascalCase, it should be unique within the namespace
	// +kubebuilder:validation:Pattern=`^[A-Z](([a-z0-9]+[A-Z]?)*)$`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Type string `json:"type"`

	// Script is the list of scripts to be executed, key is the script name and value is the script content
	// +kubebuilder:validation:MinLength=1
	Script string `json:"script"`

	// Config is the list of configurations to be used by the filter
	Config map[string]string `json:"config,omitempty"`
}

// Filter provides a way to configure filters for HTTP/HTTPS/GRPC/GRPCS/TCP protocols
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io}
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
