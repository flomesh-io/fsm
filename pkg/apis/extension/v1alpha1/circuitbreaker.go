package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CircuitBreakerSpec defines the desired state of CircuitBreaker
type CircuitBreakerSpec struct {
	// +optional
	// +kubebuilder:default="1s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// LatencyThreshold is the threshold in milliseconds to determine a slow request, default is 1s
	LatencyThreshold *metav1.Duration `json:"latencyThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	// ErrorCountThreshold is the threshold of error requests in the StatTimeWindow to trigger circuit breaking, default is 10
	ErrorCountThreshold *int32 `json:"errorCountThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=0.00
	// +kubebuilder:validation:Maximum=1.00
	// +kubebuilder:default=0.50
	// ErrorRatioThreshold is the threshold of error requests ratio in the StatTimeWindow to trigger circuit breaking, default is 0.5(50%)
	ErrorRatioThreshold *float32 `json:"errorRatioThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// ConcurrencyThreshold is the threshold of concurrent requests to trigger circuit breaking, default is 0
	ConcurrencyThreshold *int32 `json:"concurrencyThreshold,omitempty"`

	// +optional
	// +kubebuilder:default="5s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// CheckInterval is the interval to check the health of the service, default is 5s
	CheckInterval *metav1.Duration `json:"checkInterval,omitempty"`

	// +optional
	// +kubebuilder:default="30s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// BreakInterval is the interval to break the service, default is 30s
	BreakInterval *metav1.Duration `json:"breakInterval,omitempty"`

	// +optional
	// +kubebuilder:default={status: 429, body: "Circuit breaker triggered"}
	// CircuitBreakerResponse is the response when circuit breaker triggered
	CircuitBreakerResponse *CircuitBreakerResponse `json:"response,omitempty"`
}

type CircuitBreakerResponse struct {
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
	// +kubebuilder:default="Circuit breaker triggered"
	// Body is the content of response body, default is "Circuit breaker triggered"
	Body *string `json:"body,omitempty"`
}

// CircuitBreakerStatus defines the observed state of CircuitBreaker
type CircuitBreakerStatus struct {
	// Conditions describe the current conditions of the CircuitBreaker.
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

// CircuitBreaker is the Schema for the CircuitBreaker API
type CircuitBreaker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CircuitBreakerSpec   `json:"spec,omitempty"`
	Status CircuitBreakerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CircuitBreakerList contains a list of CircuitBreaker
type CircuitBreakerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CircuitBreaker `json:"items"`
}
