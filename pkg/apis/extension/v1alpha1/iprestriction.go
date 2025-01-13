package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IPRestrictionSpec defines the desired state of IPRestriction
type IPRestrictionSpec struct {
	// +optional
	// +listType=set
	// Allowed is the list of allowed IP addresses or CIDR ranges.
	Allowed []string `json:"allowed,omitempty"`

	// +optional
	// +listType=set
	// Forbidden is the list of forbidden IP addresses or CIDR ranges.
	Forbidden []string `json:"forbidden,omitempty"`
}

// IPRestrictionStatus defines the observed state of IPRestriction
type IPRestrictionStatus struct {
	// Conditions describe the current conditions of the IPRestriction.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.flomesh.io/extension=Filter}

// IPRestriction is the Schema for the IPRestriction API
type IPRestriction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPRestrictionSpec   `json:"spec,omitempty"`
	Status IPRestrictionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IPRestrictionList contains a list of IPRestriction
type IPRestrictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPRestriction `json:"items"`
}
