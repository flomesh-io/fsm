package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// RateLimitPolicySpec defines the desired state of RateLimitPolicy
type RateLimitPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +optional
	// +kubebuilder:default=10
	// Backlog is the number of requests allowed to wait in the queue
	Backlog *int `json:"backlog" json:"backlog,omitempty"`

	// Requests is the number of requests allowed per statTimeWindow
	Requests int `json:"requests" json:"requests,omitempty"`

	// Burst is the number of requests allowed to be bursted
	// +optional
	Burst *int `json:"burst" json:"burst,omitempty"`

	// StatTimeWindow is the time window in seconds
	StatTimeWindow int `json:"statTimeWindow" json:"statTimeWindow,omitempty"`

	// ResponseStatusCode is the response status code to be returned when the rate limit is exceeded
	ResponseStatusCode int `json:"responseStatusCode" json:"responseStatusCode,omitempty"`

	// +optional
	// ResponseHeadersToAdd is the response headers to be added when the rate limit is exceeded
	ResponseHeadersToAdd map[string]string `json:"responseHeadersToAdd,omitempty" json:"responseHeadersToAdd,omitempty"`
}

// RateLimitPolicyStatus defines the observed state of RateLimitPolicy
type RateLimitPolicyStatus struct {
	// Conditions describe the current conditions of the RateLimitPolicy.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io

// RateLimitPolicy is the Schema for the RateLimitPolicys API
type RateLimitPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RateLimitPolicySpec   `json:"spec,omitempty"`
	Status RateLimitPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RateLimitPolicyList contains a list of RateLimitPolicy
type RateLimitPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RateLimitPolicy `json:"items"`
}
