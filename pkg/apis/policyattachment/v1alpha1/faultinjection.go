package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// FaultInjectionPolicySpec defines the desired state of FaultInjectionPolicy
type FaultInjectionPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// Hostnames is the access control configuration for hostnames
	Hostnames []HostnameFaultInjection `json:"hostnames,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// HTTPFaultInjections is the access control configuration for HTTP routes
	HTTPFaultInjections []HTTPFaultInjection `json:"http,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// GRPCFaultInjections is the access control configuration for GRPC routes
	GRPCFaultInjections []GRPCFaultInjection `json:"grpc,omitempty"`

	// +optional
	// DefaultConfig is the default access control for all ports, routes and hostnames
	DefaultConfig *FaultInjectionConfig `json:"config,omitempty"`
}

// HostnameFaultInjection defines the access control configuration for a hostname
type HostnameFaultInjection struct {
	// Hostname is the hostname for matching the access control
	Hostname gwv1beta1.Hostname `json:"hostname"`

	// +optional
	// Config is the access control configuration for the hostname
	Config *FaultInjectionConfig `json:"config,omitempty"`
}

// HTTPFaultInjection defines the access control configuration for a HTTP route
type HTTPFaultInjection struct {
	// Match is the match condition for the HTTP route
	Match gwv1beta1.HTTPRouteMatch `json:"match"`

	// +optional
	// Config is the access control configuration for the HTTP route
	Config *FaultInjectionConfig `json:"config,omitempty"`
}

// GRPCFaultInjection defines the access control configuration for a GRPC route
type GRPCFaultInjection struct {
	// Match is the match condition for the GRPC route
	Match gwv1alpha2.GRPCRouteMatch `json:"match"`

	// +optional
	// Config is the access control configuration for the GRPC route
	Config *FaultInjectionConfig `json:"config,omitempty"`
}

// FaultInjectionConfig defines the access control configuration for a route
type FaultInjectionConfig struct {
	// +optional
	// Delay defines the delay configuration
	Delay *FaultInjectionDelay `json:"delay,omitempty"`

	// +optional
	// Abort defines the abort configuration
	Abort *FaultInjectionAbort `json:"abort,omitempty"`
}

// FaultInjectionDelay defines the delay configuration
type FaultInjectionDelay struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Percent is the percentage of requests to delay
	Percent int32 `json:"percent,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=0
	// Fixed is the fixed delay duration, default Unit is ms
	Fixed *int64 `json:"fixed,omitempty"`

	// +optional
	// Range is the range of delay duration, default Unit is ms
	Range *string `json:"range,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum=ms;s;m;h;d
	// +kubebuilder:default=ms
	// Unit is the unit of delay duration, default Unit is ms
	Unit *string `json:"unit,omitempty"`
}

// FaultInjectionAbort defines the abort configuration
type FaultInjectionAbort struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Percent is the percentage of requests to abort
	Percent int32 `json:"percent,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	// Status is the HTTP status code to return for the aborted request
	Status *int32 `json:"status,omitempty"`

	// +optional
	// Message is the HTTP status message to return for the aborted request
	Message *string `json:"message,omitempty"`
}

// FaultInjectionPolicyStatus defines the observed state of FaultInjectionPolicy
type FaultInjectionPolicyStatus struct {
	// Conditions describe the current conditions of the FaultInjectionPolicy.
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

// FaultInjectionPolicy is the Schema for the FaultInjectionPolicy API
type FaultInjectionPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FaultInjectionPolicySpec   `json:"spec,omitempty"`
	Status FaultInjectionPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FaultInjectionPolicyList contains a list of FaultInjectionPolicy
type FaultInjectionPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FaultInjectionPolicy `json:"items"`
}
