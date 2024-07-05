package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// FilterType defines the type of filter
type FilterType string

const (
	// FilterTypeHTTP is the type of filter for HTTP/HTTPS/GRPC/GRPCS protocols
	FilterTypeHTTP FilterType = "http"

	// FilterTypeTCP is the type of filter for TCP protocol
	FilterTypeTCP FilterType = "tcp"
)

// FilterSpec defines the desired state of Filter
type FilterSpec struct {
	// TargetRefs identifies an API object to attach the Filter to. Usually is the Gateway.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []gwv1.LocalObjectReference `json:"targetRefs"`

	// Type is the type of filter
	// +kubebuilder:default=http
	// +kubebuilder:validation:Enum=http;tcp
	Type FilterType `json:"type"`

	// Scripts is the list of scripts to be executed, key is the script name and value is the script content
	Scripts map[string]string `json:"scripts,omitempty"`
}

// Filter provides a way to configure filters for HTTP/HTTPS/GRPC/GRPCS/TCP protocols
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io}
type Filter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Filter.
	Spec FilterSpec `json:"spec,omitempty"`

	// Status defines the current state of Filter.
	Status FilterStatus `json:"status,omitempty"`
}

// FilterConditionType is a type of condition for a filter. This type should be
// used with a Filter resource Status.Conditions field.
type FilterConditionType string

// FilterConditionReason is a reason for a policy condition.
type FilterConditionReason string

const (
	// FilterConditionAccepted indicates whether the filter has been accepted or
	// rejected by a targeted resource, and why.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "Accepted"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "Conflicted"
	// * "Invalid"
	// * "TargetNotFound"
	//
	FilterConditionAccepted FilterConditionType = "Accepted"

	// FilterReasonAccepted is used with the "Accepted" condition when the policy
	// has been accepted by the targeted resource.
	FilterReasonAccepted FilterConditionReason = "Accepted"

	// FilterReasonConflicted is used with the "Accepted" condition when the
	// policy has not been accepted by a targeted resource because there is
	// another policy that targets the same resource and a merge is not possible.
	FilterReasonConflicted FilterConditionReason = "Conflicted"

	// FilterReasonInvalid is used with the "Accepted" condition when the policy
	// is syntactically or semantically invalid.
	FilterReasonInvalid FilterConditionReason = "Invalid"

	// FilterReasonTargetNotFound is used with the "Accepted" condition when the
	// policy is attached to an invalid target resource.
	FilterReasonTargetNotFound FilterConditionReason = "TargetNotFound"
)

// FilterAncestorStatus describes the status of a filter with respect to an
// associated Ancestor(Gateway).
type FilterAncestorStatus struct {
	// AncestorRef corresponds with a ParentRef in the spec that this
	// FilterStatus struct describes the status of.
	AncestorRef gwv1.ParentReference `json:"ancestorRef"`

	// Conditions describes the status of the Filter with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// FilterStatus defines the common attributes that all filters should include within
// their status.
type FilterStatus struct {
	// Ancestors is a list of ancestor resources (usually Gateways) that are
	// associated with the filter, and the status of the filter with respect to
	// each ancestor.
	// +kubebuilder:validation:MaxItems=16
	Ancestors []FilterAncestorStatus `json:"ancestors"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FilterList contains a list of Filter
type FilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Filter `json:"items"`
}
