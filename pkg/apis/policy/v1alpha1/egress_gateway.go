package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressGateway is the type used to represent an Egress Gateway policy.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EgressGateway struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the EgressGateway policy specification
	// +optional
	Spec EgressGatewaySpec `json:"spec,omitempty"`
}

// EgressGatewaySpec is the type used to represent the Egress Gateway specification.
type EgressGatewaySpec struct {
	// GlobalEgressGateways defines the list of Global egress gateway.
	GlobalEgressGateways []GatewayBindingSubject `json:"global"`

	// EgressPolicyGatewayRules defines the rules of gateway based egress policies.
	EgressPolicyGatewayRules []EgressPolicyGatewayRule `json:"rules"`

	// Matches defines the list of object references the EgressGateway policy should match on.
	// +optional
	Matches []corev1.TypedLocalObjectReference `json:"matches,omitempty"`
}

// EgressPolicyGatewayRule is the type used to represent the rule of Egress Gateway specification based egress policies.
type EgressPolicyGatewayRule struct {
	EgressPolicies []EgressBindingSubject  `json:"egressPolicies"`
	EgressGateways []GatewayBindingSubject `json:"egressGateways"`
}

// EgressBindingSubject is a Kubernetes objects which should be allowed egress
type EgressBindingSubject struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// GatewayBindingSubject is a Kubernetes objects which should be allowed forward to
type GatewayBindingSubject struct {
	Service   string `json:"service"`
	Namespace string `json:"namespace"`
	Weight    *int   `json:"weight,omitempty"`
}

// EgressGatewayList defines the list of EgressGateway objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EgressGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGateway `json:"items"`
}
