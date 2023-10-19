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

	// Match defines the match condition of the rate limit policy
	Match RateLimitPolicyMatch `json:"match"`

	// RateLimit defines the rate limit details
	RateLimit RateLimitPolicyConfig `json:"rateLimit"`
}

type RateLimitPolicyMatch struct {
	// +optional
	// Port defines the match condition of port for the rate limit
	Port *gwv1beta1.PortNumber `json:"port,omitempty"`

	// +optional
	// Hostnames defines the match condition of hostnames for the rate limit
	Hostnames []gwv1beta1.Hostname `json:"hostnames,omitempty"`

	// +optional
	// Route defines the match condition of route for the rate limit
	Route *RouteBasedRateLimitMatch `json:"route,omitempty"`
}

// RateLimitPolicyConfig defines the rate limit configuration
type RateLimitPolicyConfig struct {
	// +optional
	// L4RateLimit is the rate limit in bytes per second
	L4RateLimit *int64 `json:"bps,omitempty"`

	// +optional
	// L7RateLimit defines the rate limit details for Layer 7 protocols, for now it's HTTP/GRPC
	L7RateLimit *L7RateLimitPolicy `json:"config,omitempty"`
}

type L7RateLimitPolicy struct {
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

// RouteBasedRateLimitMatch defines the route based rate limit
type RouteBasedRateLimitMatch struct {
	// +optional
	HTTPRouteBasedRateLimit *HTTPRouteBasedRateLimit `json:"http,omitempty"`

	// +optional
	GRPCRouteBasedRateLimit *GRPCRouteBasedRateLimit `json:"grpc,omitempty"`
}

type HTTPRouteBasedRateLimit struct {
	// Path specifies a HTTP request path matcher. If this field is not
	// specified, a default prefix match on the "/" path is provided.
	//
	// +optional
	// +kubebuilder:default={type: "PathPrefix", value: "/"}
	Path *gwv1beta1.HTTPPathMatch `json:"path,omitempty"`

	// Headers specifies HTTP request header matchers. Multiple match values are
	// ANDed together, meaning, a request must match all the specified headers
	// to select the route.
	//
	// +listType=map
	// +listMapKey=name
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Headers []gwv1beta1.HTTPHeaderMatch `json:"headers,omitempty"`

	// QueryParams specifies HTTP query parameter matchers. Multiple match
	// values are ANDed together, meaning, a request must match all the
	// specified query parameters to select the route.
	//
	// Support: Extended
	//
	// +listType=map
	// +listMapKey=name
	// +optional
	// +kubebuilder:validation:MaxItems=16
	QueryParams []gwv1beta1.HTTPQueryParamMatch `json:"queryParams,omitempty"`

	// Method specifies HTTP method matcher.
	// When specified, this route will be matched only if the request has the
	// specified method.
	//
	// Support: Extended
	//
	// +optional
	Method *gwv1beta1.HTTPMethod `json:"method,omitempty"`
}

type GRPCRouteBasedRateLimit struct {
	// Method specifies a gRPC request service/method matcher. If this field is
	// not specified, all services and methods will match.
	//
	// +optional
	Method *gwv1alpha2.GRPCMethodMatch `json:"method,omitempty"`

	// Headers specifies gRPC request header matchers. Multiple match values are
	// ANDed together, meaning, a request MUST match all the specified headers
	// to select the route.
	//
	// +listType=map
	// +listMapKey=name
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Headers []gwv1alpha2.GRPCHeaderMatch `json:"headers,omitempty"`
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
