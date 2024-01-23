package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// HealthCheckPolicySpec defines the desired state of HealthCheckPolicy
type HealthCheckPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the health check configuration for ports
	Ports []PortHealthCheck `json:"ports,omitempty"`

	// +optional
	// DefaultConfig is the default health check configuration for all ports
	DefaultConfig *HealthCheckConfig `json:"config,omitempty"`
}

type PortHealthCheck struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// Config is the health check configuration for the port
	Config *HealthCheckConfig `json:"config,omitempty"`
}

type HealthCheckConfig struct {
	// +kubebuilder:validation:Minimum=1
	// Interval is the interval in seconds to check the health of the service
	Interval int32 `json:"interval"`

	// +kubebuilder:validation:Minimum=0
	// MaxFails is the maximum number of consecutive failed health checks before considering the service as unhealthy
	MaxFails int32 `json:"maxFails"`

	// +optional
	// +kubebuilder:validation:Minimum=0
	// FailTimeout is the time in seconds before considering the service as healthy if it's marked as unhealthy, even if it's already healthy
	FailTimeout *int32 `json:"failTimeout,omitempty"`

	// +optional
	// Path is the path to check the health of the HTTP service, if it's not set, the health check will be TCP based
	Path *string `json:"path,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// Matches is the list of health check match conditions of HTTP service
	Matches []HealthCheckMatch `json:"matches,omitempty"`
}

type HealthCheckMatch struct {
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=16
	// StatusCodes is the list of status codes to match
	StatusCodes []int32 `json:"statusCodes,omitempty"`

	// +optional
	// Body is the content of response body to match
	Body *string `json:"body,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// Headers is the list of response headers to match
	Headers []gwv1.HTTPHeader `json:"headers,omitempty"`
}

//type HealthCheckMatchType string
//
//const (
//	HealthCheckMatchTypeStatus  HealthCheckMatchType = "Status"
//	HealthCheckMatchTypeHeaders HealthCheckMatchType = "Headers"
//	HealthCheckMatchTypeBody    HealthCheckMatchType = "Body"
//)

// HealthCheckPolicyStatus defines the observed state of HealthCheckPolicy
type HealthCheckPolicyStatus struct {
	// Conditions describe the current conditions of the HealthCheckPolicy.
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

// HealthCheckPolicy is the Schema for the HealthCheckPolicy API
type HealthCheckPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealthCheckPolicySpec   `json:"spec,omitempty"`
	Status HealthCheckPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HealthCheckPolicyList contains a list of HealthCheckPolicy
type HealthCheckPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealthCheckPolicy `json:"items"`
}
