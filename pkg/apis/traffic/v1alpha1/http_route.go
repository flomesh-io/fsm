package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io}

// HTTPTrafficRule is used to describe HTTP/1 and HTTP/2 traffic.
// It enumerates the routes that can be served by an application.
type HTTPTrafficRule struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HTTPTrafficRuleSpec `json:"spec"`

	// Status defines the current state of HTTPTrafficRule.
	Status HTTPTrafficRuleStatus `json:"status,omitempty"`
}

// HTTPTrafficRuleSpec is the specification for a HTTPTrafficRule
type HTTPTrafficRuleSpec struct {
	// Routes for inbound traffic
	Matches []HTTPMatch `json:"matches,omitempty"`
}

// HTTPMatch defines an individual route for HTTP traffic
type HTTPMatch struct {
	// Name is the name of the match for referencing in a TrafficTarget
	Name string `json:"name,omitempty"`

	// Methods for inbound traffic as defined in RFC 7231
	// https://tools.ietf.org/html/rfc7231#section-4
	Methods []string `json:"methods,omitempty"`

	// PathRegex is a regular expression defining the route
	PathRegex string `json:"pathRegex,omitempty"`

	// Headers is a list of headers used to match HTTP traffic
	Headers httpHeaders `json:"headers,omitempty"`
}

// httpHeaders is a map of key/value pairs which match HTTP header name and value
type httpHeaders map[string]string

// HTTPTrafficMethod are methods allowed by the route
type HTTPTrafficMethod string

const (
	// HTTPTrafficMethodAll is a wildcard for all HTTP methods
	HTTPTrafficMethodAll HTTPTrafficMethod = "*"
	// HTTPTrafficMethodGet HTTP GET method
	HTTPTrafficMethodGet HTTPTrafficMethod = "GET"
	// HTTPTrafficMethodHead HTTP HEAD method
	HTTPTrafficMethodHead HTTPTrafficMethod = "HEAD"
	// HTTPTrafficMethodPut HTTP PUT method
	HTTPTrafficMethodPut HTTPTrafficMethod = "PUT"
	// HTTPTrafficMethodPost HTTP POST method
	HTTPTrafficMethodPost HTTPTrafficMethod = "POST"
	// HTTPTrafficMethodDelete HTTP DELETE method
	HTTPTrafficMethodDelete HTTPTrafficMethod = "DELETE"
	// HTTPTrafficMethodConnect HTTP CONNECT method
	HTTPTrafficMethodConnect HTTPTrafficMethod = "CONNECT"
	// HTTPTrafficMethodOptions HTTP OPTIONS method
	HTTPTrafficMethodOptions HTTPTrafficMethod = "OPTIONS"
	// HTTPTrafficMethodTrace HTTP TRACE method
	HTTPTrafficMethodTrace HTTPTrafficMethod = "TRACE"
	// HTTPTrafficMethodPatch HTTP PATCH method
	HTTPTrafficMethodPatch HTTPTrafficMethod = "PATCH"
)

// HTTPTrafficRuleStatus defines the common attributes that all filters should include within
// their status.
type HTTPTrafficRuleStatus struct {
	// Conditions describes the status of the HTTPTrafficRule with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HTTPTrafficRuleList satisfy K8s code gen requirements
type HTTPTrafficRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []HTTPTrafficRule `json:"items"`
}
