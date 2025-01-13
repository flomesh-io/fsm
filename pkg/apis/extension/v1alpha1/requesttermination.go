package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// RequestTerminationSpec defines the desired state of RequestTermination
type RequestTerminationSpec struct {
	// RequestTerminationResponse is the response when circuit breaker triggered
	RequestTerminationResponse RequestTerminationResponse `json:"response,omitempty"`
}

type RequestTerminationResponse struct {
	// +kubebuilder:default=500
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=600
	// StatusCode is the HTTP status code of the response, default is 500
	StatusCode int32 `json:"status,omitempty"`

	// +optional
	// Headers is the HTTP headers of response
	Headers map[gwv1.HeaderName]string `json:"headers,omitempty"`

	// +optional
	// +kubebuilder:default="Request termination triggered"
	// Body is the content of response body, default is "Request termination triggered"
	Body *string `json:"body,omitempty"`
}

// RequestTerminationStatus defines the observed state of RequestTermination
type RequestTerminationStatus struct {
	// Conditions describe the current conditions of the RequestTermination.
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

// RequestTermination is the Schema for the RequestTermination API
type RequestTermination struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RequestTerminationSpec   `json:"spec,omitempty"`
	Status RequestTerminationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RequestTerminationList contains a list of RequestTermination
type RequestTerminationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RequestTermination `json:"items"`
}
