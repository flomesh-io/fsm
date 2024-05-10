package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// CircuitBreakingPolicySpec defines the desired state of CircuitBreakingPolicy
type CircuitBreakingPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the circuit breaking configuration for ports
	Ports []PortCircuitBreaking `json:"ports,omitempty"`

	// +optional
	// DefaultConfig is the default circuit breaking configuration for all ports
	DefaultConfig *CircuitBreakingConfig `json:"config,omitempty"`
}

type PortCircuitBreaking struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// Config is the circuit breaking configuration for the port
	Config *CircuitBreakingConfig `json:"config,omitempty"`
}

type CircuitBreakingConfig struct {
	// +kubebuilder:validation:Minimum=1
	// MinRequestAmount is the minimum number of requests in the StatTimeWindow
	MinRequestAmount int32 `json:"minRequestAmount"`

	// +kubebuilder:validation:Minimum=1
	// StatTimeWindow is the time window in seconds to collect statistics
	StatTimeWindow int32 `json:"statTimeWindow"`

	// +optional
	// +kubebuilder:validation:Minimum=0.0001
	// SlowTimeThreshold is the threshold in seconds to determine a slow request
	SlowTimeThreshold *float32 `json:"slowTimeThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// SlowAmountThreshold is the threshold of slow requests in the StatTimeWindow to trigger circuit breaking
	SlowAmountThreshold *int32 `json:"slowAmountThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=0.00
	// +kubebuilder:validation:Maximum=1.00
	// SlowRatioThreshold is the threshold of slow requests ratio in the StatTimeWindow to trigger circuit breaking
	SlowRatioThreshold *float32 `json:"slowRatioThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// ErrorAmountThreshold is the threshold of error requests in the StatTimeWindow to trigger circuit breaking
	ErrorAmountThreshold *int32 `json:"errorAmountThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=0.00
	// +kubebuilder:validation:Maximum=1.00
	// ErrorRatioThreshold is the threshold of error requests ratio in the StatTimeWindow to trigger circuit breaking
	ErrorRatioThreshold *float32 `json:"errorRatioThreshold,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// DegradedTimeWindow is the time window in seconds to degrade the service
	DegradedTimeWindow int32 `json:"degradedTimeWindow"`

	// +kubebuilder:default=503
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	// DegradedStatusCode is the status code to return when the service is degraded
	DegradedStatusCode int32 `json:"degradedStatusCode"`

	// +optional
	// DegradedResponseContent is the response content to return when the service is degraded
	DegradedResponseContent *string `json:"degradedResponseContent,omitempty"`
}

// CircuitBreakingPolicyStatus defines the observed state of CircuitBreakingPolicy
type CircuitBreakingPolicyStatus struct {
	// Conditions describe the current conditions of the CircuitBreakingPolicy.
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
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.networking.k8s.io/policy=Direct}

// CircuitBreakingPolicy is the Schema for the CircuitBreakingPolicy API
type CircuitBreakingPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CircuitBreakingPolicySpec   `json:"spec,omitempty"`
	Status CircuitBreakingPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CircuitBreakingPolicyList contains a list of CircuitBreakingPolicy
type CircuitBreakingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CircuitBreakingPolicy `json:"items"`
}
