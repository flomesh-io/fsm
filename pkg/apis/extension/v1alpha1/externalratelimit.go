package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ExternalRateLimitSpec defines the desired state of ExternalRateLimit
type ExternalRateLimitSpec struct {
	// ThrottleHost is the host to be throttled
	ThrottleHost HostPort `json:"throttleHost,omitempty"`

	// +optional
	// +listType=set
	// PassHeaders is the list of headers to be passed to the backend service
	PassHeaders []gwv1.HeaderName `json:"passHeaders,omitempty"`
}

// ExternalRateLimitStatus defines the observed state of ExternalRateLimit
type ExternalRateLimitStatus struct {
	// Conditions describe the current conditions of the ExternalRateLimit.
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

// ExternalRateLimit is the Schema for the ExternalRateLimit API
type ExternalRateLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalRateLimitSpec   `json:"spec,omitempty"`
	Status ExternalRateLimitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalRateLimitList contains a list of ExternalRateLimit
type ExternalRateLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalRateLimit `json:"items"`
}
