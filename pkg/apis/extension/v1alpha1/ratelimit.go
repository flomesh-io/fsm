package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RateLimitSpec defines the desired state of RateLimit
type RateLimitSpec struct {
	// +optional
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=0
	// Burst is the maximum number of requests that can be made in Interval, default is 10
	Burst *int32 `json:"burst,omitempty"`

	// +optional
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=0
	// Requests is the number of requests that can be made in Interval, default is 5
	Requests *int32 `json:"requests,omitempty"`

	// +optional
	// +kubebuilder:default="10s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// Interval is the time period in which the requests are counted
	Interval *metav1.Duration `json:"interval,omitempty"`

	// +optional
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=0
	// Backlog is the maximum number of requests that can be queued
	Backlog *int32 `json:"backlog,omitempty"`

	// +optional
	// +kubebuilder:default={status: 429, body: "Rate limit reached"}
	// RateLimitResponse is the response when Rate limit reached
	RateLimitResponse *RateLimitResponse `json:"response,omitempty"`
}

type RateLimitResponse struct {
	// +optional
	// +kubebuilder:default=429
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=600
	// StatusCode is the HTTP status code of the response, default is 429
	StatusCode *int32 `json:"status,omitempty"`

	// +optional
	// Headers is the HTTP headers of response
	Headers map[string]string `json:"headers,omitempty"`

	// +optional
	// +kubebuilder:default="Rate limit reached"
	// Body is the content of response body, default is "Rate limit reached"
	Body *string `json:"body,omitempty"`
}

// RateLimitStatus defines the observed state of RateLimit
type RateLimitStatus struct {
	// Conditions describe the current conditions of the RateLimit.
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
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io}

// RateLimit is the Schema for the RateLimit API
type RateLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RateLimitSpec   `json:"spec,omitempty"`
	Status RateLimitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RateLimitList contains a list of RateLimit
type RateLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RateLimit `json:"items"`
}
