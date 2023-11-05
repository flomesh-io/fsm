package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// AccessControlPolicySpec defines the desired state of AccessControlPolicy
type AccessControlPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +optional
	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MaxItems=16
	// Ports is the access control configuration for ports
	Ports []PortAccessControl `json:"ports,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=hostname
	// +kubebuilder:validation:MaxItems=16
	// Hostnames is the access control configuration for hostnames
	Hostnames []HostnameAccessControl `json:"hostnames,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// HTTPAccessControls is the access control configuration for HTTP routes
	HTTPAccessControls []HTTPAccessControl `json:"http,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// GRPCAccessControls is the access control configuration for GRPC routes
	GRPCAccessControls []GRPCAccessControl `json:"grpc,omitempty"`

	// +optional
	// DefaultConfig is the default access control for all ports, routes and hostnames
	DefaultConfig *AccessControlConfig `json:"config,omitempty"`
}

// PortAccessControl defines the access control configuration for a port
type PortAccessControl struct {
	// Port is the port number for matching the access control
	Port gwv1beta1.PortNumber `json:"port"`

	// +optional
	// Config is the access control configuration for the port
	Config *AccessControlConfig `json:"config,omitempty"`
}

// HostnameAccessControl defines the access control configuration for a hostname
type HostnameAccessControl struct {
	// Hostname is the hostname for matching the access control
	Hostname gwv1beta1.Hostname `json:"hostname"`

	// +optional
	// Config is the access control configuration for the hostname
	Config *AccessControlConfig `json:"config,omitempty"`
}

// HTTPAccessControl defines the access control configuration for a HTTP route
type HTTPAccessControl struct {
	// Match is the match condition for the HTTP route
	Match gwv1beta1.HTTPRouteMatch `json:"match"`

	// +optional
	// Config is the access control configuration for the HTTP route
	Config *AccessControlConfig `json:"config,omitempty"`
}

// GRPCAccessControl defines the access control configuration for a GRPC route
type GRPCAccessControl struct {
	// Match is the match condition for the GRPC route
	Match gwv1alpha2.GRPCRouteMatch `json:"match"`

	// +optional
	// Config is the access control configuration for the GRPC route
	Config *AccessControlConfig `json:"config,omitempty"`
}

// AccessControlConfig defines the access control configuration for a route
type AccessControlConfig struct {
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=32
	// Blacklist is the list of IP addresses to be blacklisted
	Blacklist []string `json:"blacklist,omitempty"`

	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=32
	// Whitelist is the list of IP addresses to be whitelisted
	Whitelist []string `json:"whitelist,omitempty"`

	// +optional
	// +kubebuilder:default=false
	// EnableXFF is the flag to enable X-Forwarded-For header
	EnableXFF *bool `json:"enableXFF,omitempty"`

	// +optional
	// +kubebuilder:default=403
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	// StatusCode is the response status code to be returned when the access control is exceeded
	StatusCode *int32 `json:"statusCode,omitempty"`

	// +optional
	// +kubebuilder:default=""
	// Message is the response message to be returned when the access control is exceeded
	Message *string `json:"message,omitempty"`
}

// AccessControlPolicyStatus defines the observed state of AccessControlPolicy
type AccessControlPolicyStatus struct {
	// Conditions describe the current conditions of the AccessControlPolicy.
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

// AccessControlPolicy is the Schema for the AccessControlPolicy API
type AccessControlPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessControlPolicySpec   `json:"spec,omitempty"`
	Status AccessControlPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessControlPolicyList contains a list of AccessControlPolicy
type AccessControlPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessControlPolicy `json:"items"`
}
