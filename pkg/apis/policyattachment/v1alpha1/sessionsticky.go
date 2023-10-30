package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// SessionStickyPolicySpec defines the desired state of SessionStickyPolicy
type SessionStickyPolicySpec struct {
	// TargetRef is the reference to the target resource to which the policy is applied
	TargetRef gwv1alpha2.PolicyTargetReference `json:"targetRef"`

	// Port is the port number of the target service
	Port gwv1beta1.PortNumber `json:"port"`

	// +optional
	// +kubebuilder:default=_srv_id
	// CookieName is the name of the cookie used for sticky session
	CookieName *string `json:"cookieName"`

	// +optional
	// +kubebuilder:default=3600
	// Expires is the expiration time of the cookie in seconds
	Expires *int32 `json:"expires"`
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
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io

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
