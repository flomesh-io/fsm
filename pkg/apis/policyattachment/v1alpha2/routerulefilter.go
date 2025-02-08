package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api,shortName=rfpolicy
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.networking.k8s.io/policy=Direct}

// RouteRuleFilterPolicy provides a way to define load balancing rules
// for a backend.
type RouteRuleFilterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of RouteRuleFilterPolicy.
	Spec RouteRuleFilterPolicySpec `json:"spec"`

	// Status defines the current state of RouteRuleFilterPolicy.
	Status gwv1alpha2.PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RouteRuleFilterPolicyList contains a list of RouteRuleFilterPolicies
type RouteRuleFilterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RouteRuleFilterPolicy `json:"items"`
}

// RouteRuleFilterPolicySpec defines the desired state of
// RouteRuleFilterPolicy.
// Note: there is no Override or Default policy configuration.
type RouteRuleFilterPolicySpec struct {
	// TargetRef identifies an API object to apply policy to.
	// Currently, TCPRoute and UDPRoute are the only valid API
	// target references.
	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +listMapKey=rule
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []LocalFilterPolicyTargetReference `json:"targetRefs"`

	// FilterRefs identifies an API object to apply policy to.
	// Currently, Filter are the only valid API
	// target references.
	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	FilterRefs []LocalFilterReference `json:"filterRefs"`
}
