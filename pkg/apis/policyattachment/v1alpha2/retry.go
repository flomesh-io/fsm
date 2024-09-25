package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// RetryPolicySpec defines the desired state of RetryPolicy
type RetryPolicySpec struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// TargetRefs is the references to the target resources to which the policy is applied
	TargetRefs []gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRefs"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the retry configuration for ports
	Ports []PortRetry `json:"ports,omitempty"`

	// +optional
	// DefaultRetry is the default retry configuration for all ports
	DefaultRetry *RetryConfig `json:"retry,omitempty"`
}

// PortRetry defines the retry configuration for a port
type PortRetry struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// Config is the retry configuration for the port
	Retry *RetryConfig `json:"retry,omitempty"`
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
	// +kubebuilder:default="1s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// BackoffBaseInterval is the base interval for computing backoff time between retries, default is 1s
	BackoffBaseInterval *metav1.Duration `json:"backoffBaseInterval,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api,shortName=rtpolicy
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.networking.k8s.io/policy=Direct}

// RetryPolicy provides a way to configure how a Gateway
// tries to re-invoke failed backends.
type RetryPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of RetryPolicy.
	Spec RetryPolicySpec `json:"spec,omitempty"`

	// Status defines the current state of RetryPolicy.
	Status gwv1alpha2.PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RetryPolicyList contains a list of RetryPolicy
type RetryPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RetryPolicy `json:"items"`
}
