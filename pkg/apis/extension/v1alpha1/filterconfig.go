package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FilterConfigSpec defines the desired state of FilterConfig
type FilterConfigSpec struct {
	// +kube:validation:Required
	// +kubebuilder:validation:MinLength=1
	// Config is the filter configuration in YAML format
	Config string `json:"config"`
}

// FilterConfig provides a way to configure filters for HTTP/HTTPS/GRPC/GRPCS/TCP protocols
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io}
type FilterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of FilterConfig.
	Spec FilterConfigSpec `json:"spec,omitempty"`

	// Status defines the current state of FilterConfig.
	Status FilterConfigStatus `json:"status,omitempty"`
}

// FilterConfigStatus defines the common attributes that all filters should include within
// their status.
type FilterConfigStatus struct {
	// Conditions describes the status of the FilterConfig with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FilterConfigList contains a list of FilterConfig
type FilterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FilterConfig `json:"items"`
}
