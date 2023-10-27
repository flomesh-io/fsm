package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type RateLimitPolicyMode string

const (
	// RateLimitPolicyModeLocal is the local mode
	RateLimitPolicyModeLocal RateLimitPolicyMode = "Local"

	// RateLimitPolicyModeGlobal is the global mode
	RateLimitPolicyModeGlobal RateLimitPolicyMode = "Global"
)

// RateLimitPolicySpec defines the desired state of RateLimitPolicy
type RateLimitPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +optional
	// Ports is the rate limit configuration for ports
	Ports []PortRateLimit `json:"ports,omitempty"`

	// +optional
	// DefaultBPS is the default rate limit for all ports
	DefaultBPS *int64 `json:"bps,omitempty"`

	// +optional
	// Hostnames is the rate limit configuration for hostnames
	Hostnames []HostnameRateLimit `json:"hostnames,omitempty"`

	// +optional
	// HTTPRateLimits is the rate limit configuration for HTTP routes
	HTTPRateLimits []HTTPRateLimit `json:"http,omitempty"`

	// +optional
	// GRPCRateLimits is the rate limit configuration for GRPC routes
	GRPCRateLimits []GRPCRateLimit `json:"grpc,omitempty"`

	// +optional
	// DefaultRateLimit is the default rate limit for all routes and hostnames
	DefaultL7RateLimit *L7RateLimit `json:"rateLimit,omitempty"`
}

// PortRateLimit defines the rate limit configuration for a port
type PortRateLimit struct {
	Port gwv1beta1.PortNumber `json:"port"`
	BPS  *int64               `json:"bps,omitempty"`
}

// HostnameRateLimit defines the rate limit configuration for a hostname
type HostnameRateLimit struct {
	Hostname  gwv1beta1.Hostname `json:"hostname"`
	RateLimit *L7RateLimit       `json:"rateLimit,omitempty"`
}

// RouteRateLimitConfig defines the rate limit configuration for routes
type RouteRateLimitConfig struct {
	HttpRateLimits   []HTTPRateLimit `json:"http,omitempty"`
	GrpcRateLimits   []GRPCRateLimit `json:"grpc,omitempty"`
	DefaultRateLimit *L7RateLimit    `json:"rateLimit,omitempty"`
}

// HTTPRateLimit defines the rate limit configuration for a HTTP route
type HTTPRateLimit struct {
	Match     gwv1beta1.HTTPRouteMatch `json:"match"`
	RateLimit *L7RateLimit             `json:"rateLimit,omitempty"`
}

// GRPCRateLimit defines the rate limit configuration for a GRPC route
type GRPCRateLimit struct {
	Match     gwv1alpha2.GRPCRouteMatch `json:"match"`
	RateLimit *L7RateLimit              `json:"rateLimit,omitempty"`
}

// L7RateLimit defines the rate limit configuration for a route
type L7RateLimit struct {
	// +optional
	// +kubebuilder:default=Local
	// +kubebuilder:validation:Enum=Local;Global
	// Mode is the mode of the rate limit policy, Local or Global, default is Local
	Mode *RateLimitPolicyMode `json:"mode"`

	// +optional
	// +kubebuilder:default=10
	// Backlog is the number of requests allowed to wait in the queue
	Backlog *int `json:"backlog,omitempty"`

	// Requests is the number of requests allowed per statTimeWindow
	Requests int `json:"requests"`

	// Burst is the number of requests allowed to be bursted, if not specified, it will be the same as Requests
	// +optional
	Burst *int `json:"burst,omitempty"`

	// StatTimeWindow is the time window in seconds
	StatTimeWindow int `json:"statTimeWindow"`

	// ResponseStatusCode is the response status code to be returned when the rate limit is exceeded
	// +optional
	// +kubebuilder:default=429
	ResponseStatusCode *int `json:"responseStatusCode"`

	// +optional
	// ResponseHeadersToAdd is the response headers to be added when the rate limit is exceeded
	ResponseHeadersToAdd map[string]string `json:"responseHeadersToAdd,omitempty"`
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
