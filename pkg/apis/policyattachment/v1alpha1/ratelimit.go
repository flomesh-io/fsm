package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
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
	TargetRef gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRef"`

	// +optional
	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MaxItems=16
	// Ports is the rate limit configuration for ports
	Ports []PortRateLimit `json:"ports,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// DefaultBPS is the default rate limit for all ports
	DefaultBPS *int64 `json:"bps,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=hostname
	// +kubebuilder:validation:MaxItems=16
	// Hostnames is the rate limit configuration for hostnames
	Hostnames []HostnameRateLimit `json:"hostnames,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// HTTPRateLimits is the rate limit configuration for HTTP routes
	HTTPRateLimits []HTTPRateLimit `json:"http,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// GRPCRateLimits is the rate limit configuration for GRPC routes
	GRPCRateLimits []GRPCRateLimit `json:"grpc,omitempty"`

	// +optional
	// DefaultConfig is the default rate limit for all routes and hostnames
	DefaultConfig *L7RateLimit `json:"config,omitempty"`
}

// PortRateLimit defines the rate limit configuration for a port
type PortRateLimit struct {
	// Port is the port number for matching the rate limit
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// BPS is the rate limit in bytes per second for the port
	BPS *int64 `json:"bps,omitempty"`
}

// HostnameRateLimit defines the rate limit configuration for a hostname
type HostnameRateLimit struct {
	// Hostname is the hostname for matching the rate limit
	Hostname gwv1.Hostname `json:"hostname"`

	// +optional
	// Config is the rate limit configuration for the hostname
	Config *L7RateLimit `json:"config,omitempty"`
}

// HTTPRateLimit defines the rate limit configuration for a HTTP route
type HTTPRateLimit struct {
	// Match is the match condition for the HTTP route
	Match gwv1.HTTPRouteMatch `json:"match"`

	// +optional
	// Config is the rate limit configuration for the HTTP route
	Config *L7RateLimit `json:"config,omitempty"`
}

// GRPCRateLimit defines the rate limit configuration for a GRPC route
type GRPCRateLimit struct {
	// Match is the match condition for the GRPC route
	Match gwv1.GRPCRouteMatch `json:"match"`

	// +optional
	// Config is the rate limit configuration for the GRPC route
	Config *L7RateLimit `json:"config,omitempty"`
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
	// +kubebuilder:validation:Minimum=1
	// Backlog is the number of requests allowed to wait in the queue
	Backlog *int32 `json:"backlog,omitempty"`

	// Requests is the number of requests allowed per statTimeWindow
	// +kubebuilder:validation:Minimum=1
	Requests int32 `json:"requests"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// Burst is the number of requests allowed to be bursted, if not specified, it will be the same as Requests
	Burst *int32 `json:"burst,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// StatTimeWindow is the time window in seconds
	StatTimeWindow int32 `json:"statTimeWindow"`

	// +optional
	// +kubebuilder:default=429
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	// ResponseStatusCode is the response status code to be returned when the rate limit is exceeded
	ResponseStatusCode *int32 `json:"responseStatusCode"`

	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// ResponseHeadersToAdd is the response headers to be added when the rate limit is exceeded
	ResponseHeadersToAdd []gwv1.HTTPHeader `json:"responseHeadersToAdd,omitempty"`
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
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.networking.k8s.io/policy=Direct}

// RateLimitPolicy is the Schema for the RateLimitPolicy API
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
