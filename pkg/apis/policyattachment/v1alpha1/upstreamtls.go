package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// UpstreamTLSPolicySpec defines the desired state of UpstreamTLSPolicy
type UpstreamTLSPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the session sticky configuration for ports
	Ports []PortUpstreamTLS `json:"ports,omitempty"`

	// +optional
	// DefaultConfig is the default session sticky configuration for all ports
	DefaultConfig *UpstreamTLSConfig `json:"config,omitempty"`
}

// PortUpstreamTLS defines the session sticky configuration for a port
type PortUpstreamTLS struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// Config is the session sticky configuration for the port
	Config *UpstreamTLSConfig `json:"config,omitempty"`
}

// UpstreamTLSConfig defines the session sticky configuration
type UpstreamTLSConfig struct {
	// CertificateRef is the reference to the certificate used for TLS connection to upstream
	CertificateRef gwv1.SecretObjectReference `json:"certificateRef"`

	// +optional
	// +kubebuilder:default=false
	// MTLS is the flag to enable mutual TLS to upstream
	MTLS *bool `json:"mTLS,omitempty"`
}

// UpstreamTLSPolicyStatus defines the observed state of UpstreamTLSPolicy
type UpstreamTLSPolicyStatus struct {
	// Conditions describe the current conditions of the UpstreamTLSPolicy.
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

// UpstreamTLSPolicy is the Schema for the UpstreamTLSPolicy API
type UpstreamTLSPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UpstreamTLSPolicySpec   `json:"spec,omitempty"`
	Status UpstreamTLSPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UpstreamTLSPolicyList contains a list of UpstreamTLSPolicy
type UpstreamTLSPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UpstreamTLSPolicy `json:"items"`
}
