package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// GatewayTLSPolicySpec defines the desired state of GatewayTLSPolicy
type GatewayTLSPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the Gateway TLS configuration for ports
	Ports []PortGatewayTLS `json:"ports,omitempty"`

	// +optional
	// DefaultConfig is the default Gateway TLS configuration for all ports
	DefaultConfig *GatewayTLSConfig `json:"config,omitempty"`
}

// PortGatewayTLS defines the Gateway TLS configuration for a port
type PortGatewayTLS struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// Config is the Gateway TLS configuration for the port
	Config *GatewayTLSConfig `json:"config,omitempty"`
}

// GatewayTLSConfig defines the Gateway TLS configuration
type GatewayTLSConfig struct {
	// +optional
	// +kubebuilder:default=false
	// MTLS defines if the gateway port should use mTLS or not
	MTLS *bool `json:"mTLS,omitempty"`
}

// GatewayTLSPolicyStatus defines the observed state of GatewayTLSPolicy
type GatewayTLSPolicyStatus struct {
	// Conditions describe the current conditions of the GatewayTLSPolicy.
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

// GatewayTLSPolicy is the Schema for the GatewayTLSPolicy API
type GatewayTLSPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewayTLSPolicySpec   `json:"spec,omitempty"`
	Status GatewayTLSPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GatewayTLSPolicyList contains a list of GatewayTLSPolicy
type GatewayTLSPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayTLSPolicy `json:"items"`
}
