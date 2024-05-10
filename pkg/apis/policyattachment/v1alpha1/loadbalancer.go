package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type LoadBalancerType string

const (
	RoundRobinLoadBalancer      LoadBalancerType = "RoundRobinLoadBalancer"
	HashingLoadBalancer         LoadBalancerType = "HashingLoadBalancer"
	LeastConnectionLoadBalancer LoadBalancerType = "LeastConnectionLoadBalancer"
)

// LoadBalancerPolicySpec defines the desired state of LoadBalancerPolicy
type LoadBalancerPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRef"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the load balancer configuration for ports
	Ports []PortLoadBalancer `json:"ports,omitempty"`

	// +optional
	// +kubebuilder:default=RoundRobinLoadBalancer
	// +kubebuilder:validation:Enum=RoundRobinLoadBalancer;HashingLoadBalancer;LeastConnectionLoadBalancer
	// DefaultType is the default type of the load balancer for all ports
	DefaultType *LoadBalancerType `json:"type,omitempty"`
}

// PortLoadBalancer defines the load balancer configuration for a port
type PortLoadBalancer struct {
	// Port is the port number for matching the load balancer
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// +kubebuilder:default=RoundRobinLoadBalancer
	// +kubebuilder:validation:Enum=RoundRobinLoadBalancer;HashingLoadBalancer;LeastConnectionLoadBalancer
	// Type is the type of the load balancer
	Type *LoadBalancerType `json:"type,omitempty"`
}

// LoadBalancerPolicyStatus defines the observed state of LoadBalancerPolicy
type LoadBalancerPolicyStatus struct {
	// Conditions describe the current conditions of the LoadBalancerPolicy.
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

// LoadBalancerPolicy is the Schema for the LoadBalancerPolicy API
type LoadBalancerPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadBalancerPolicySpec   `json:"spec,omitempty"`
	Status LoadBalancerPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadBalancerPolicyList contains a list of LoadBalancerPolicy
type LoadBalancerPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadBalancerPolicy `json:"items"`
}
