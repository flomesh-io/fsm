package v1alpha2

import gwv1 "sigs.k8s.io/gateway-api/apis/v1"

type LoadBalancerAlgorithm string

const (
	LoadBalancerAlgorithmRoundRobin LoadBalancerAlgorithm = "RoundRobin"
	LoadBalancerAlgorithmLeastLoad  LoadBalancerAlgorithm = "LeastLoad"
)

type LocalFilterPolicyTargetReference struct {
	// Group is the group of the target resource.
	// +kubebuilder:validation:Enum=gateway.networking.k8s.io
	Group gwv1.Group `json:"group"`

	// Kind is kind of the target resource.
	// +kubebuilder:validation:Enum=TCPRoute;UDPRoute
	Kind gwv1.Kind `json:"kind"`

	// Name is the name of the target resource.
	Name gwv1.ObjectName `json:"name"`

	// Rule is the rule name of the target resource.
	Rule gwv1.SectionName `json:"rule"`
}

type LocalFilterReference struct {
	// Group is the group of the target resource.
	// +kubebuilder:validation:Enum=extension.gateway.flomesh.io
	Group gwv1.Group `json:"group"`

	// Kind is kind of the target resource.
	// +kubebuilder:validation:Enum=Filter
	Kind gwv1.Kind `json:"kind"`

	// Name is the name of the target resource.
	Name gwv1.ObjectName `json:"name"`

	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10000
	// +kubebuilder:default=100
	// Priority is the priority of the policy, lower value means higher priority.
	Priority *int32 `json:"priority,omitempty"`
}
