package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// RetryPolicySpec defines the desired state of RetryPolicy
type RetryPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the retry configuration for ports
	Ports []PortRetry `json:"ports,omitempty"`

	// +optional
	// DefaultConfig is the default retry configuration for all ports
	DefaultConfig *RetryConfig `json:"config,omitempty"`
}

// PortRetry defines the retry configuration for a port
type PortRetry struct {
	// Port is the port number of the target service
	Port gwv1beta1.PortNumber `json:"port"`

	// +optional
	// Config is the retry configuration for the port
	Config *RetryConfig `json:"config,omitempty"`
}

// RetryConfig defines the retry configuration
type RetryConfig struct {
	// +listType=set
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// RetryOn is the list of retryable response codes, e.g. 5xx matches 500-599, or 500 matches just 500
	RetryOn []string `json:"retryOn,omitempty"`

	// +optional
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	// NumRetries is the number of retries
	NumRetries *int32 `json:"numRetries,omitempty"`

	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// BackoffBaseInterval is the base interval for computing backoff in seconds
	BackoffBaseInterval *int32 `json:"backoffBaseInterval,omitempty"`
}

// RetryPolicyStatus defines the observed state of RetryPolicy
type RetryPolicyStatus struct {
	// Conditions describe the current conditions of the RetryPolicy.
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

// RetryPolicy is the Schema for the RetryPolicy API
type RetryPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RetryPolicySpec   `json:"spec,omitempty"`
	Status RetryPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RetryPolicyList contains a list of RetryPolicy
type RetryPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RetryPolicy `json:"items"`
}
