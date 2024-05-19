package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// SessionStickyPolicySpec defines the desired state of SessionStickyPolicy
type SessionStickyPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the session sticky configuration for ports
	Ports []PortSessionSticky `json:"ports,omitempty"`

	// +optional
	// DefaultConfig is the default session sticky configuration for all ports
	DefaultConfig *SessionStickyConfig `json:"config,omitempty"`
}

// PortSessionSticky defines the session sticky configuration for a port
type PortSessionSticky struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// Config is the session sticky configuration for the port
	Config *SessionStickyConfig `json:"config,omitempty"`
}

// SessionStickyConfig defines the session sticky configuration
type SessionStickyConfig struct {
	// +optional
	// +kubebuilder:default=_srv_id
	// CookieName is the name of the cookie used for sticky session
	CookieName *string `json:"cookieName,omitempty"`

	// +optional
	// +kubebuilder:default=3600
	// +kubebuilder:validation:Minimum=1
	// Expires is the expiration time of the cookie in seconds
	Expires *int32 `json:"expires,omitempty"`
}

// SessionStickyPolicyStatus defines the observed state of SessionStickyPolicy
type SessionStickyPolicyStatus struct {
	// Conditions describe the current conditions of the SessionStickyPolicy.
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

// SessionStickyPolicy is the Schema for the SessionStickyPolicy API
type SessionStickyPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionStickyPolicySpec   `json:"spec,omitempty"`
	Status SessionStickyPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SessionStickyPolicyList contains a list of SessionStickyPolicy
type SessionStickyPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SessionStickyPolicy `json:"items"`
}
