package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
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
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// Port is the port number of the target service
	Port gwv1beta1.PortNumber `json:"port"`

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
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io

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
