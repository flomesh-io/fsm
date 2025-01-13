package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConcurrencyLimitSpec defines the desired state of ConcurrencyLimit
type ConcurrencyLimitSpec struct {
	// +kubebuilder:default=100
	// +kubebuilder:validation:Minimum=1
	// MaxConnections is the maximum number of concurrent connections, default is 100
	MaxConnections *int32 `json:"maxConnections,omitempty"`
}

// ConcurrencyLimitStatus defines the observed state of ConcurrencyLimit
type ConcurrencyLimitStatus struct {
	// Conditions describe the current conditions of the ConcurrencyLimit.
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

// ConcurrencyLimit is the Schema for the ConcurrencyLimit API
type ConcurrencyLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConcurrencyLimitSpec   `json:"spec,omitempty"`
	Status ConcurrencyLimitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConcurrencyLimitList contains a list of ConcurrencyLimit
type ConcurrencyLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConcurrencyLimit `json:"items"`
}
