package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FilterProtocol defines the protocol of filter
type FilterProtocol string

const (
	// FilterProtocolHTTP is the type of filter for HTTP/HTTPS/GRPC/GRPCS protocols
	FilterProtocolHTTP FilterProtocol = "http"

	// FilterProtocolTCP is the type of filter for TCP protocol
	FilterProtocolTCP FilterProtocol = "tcp"
)

// FilterSpec defines the desired state of Filter
type FilterSpec struct {
	// Protocol is the protocol of filter
	// +kubebuilder:default=http
	// +kubebuilder:validation:Enum=http;tcp
	Protocol FilterProtocol `json:"protocol"`

	// Type is the type of the filter in PascalCase, it should be unique within the namespace
	// +kubebuilder:validation:Pattern=`^[A-Z](([a-z0-9]+[A-Z]?)*)$`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Type string `json:"type"`

	// Script is the list of scripts to be executed, key is the script name and value is the script content
	// +kubebuilder:validation:MinLength=1
	Script string `json:"script"`

	// Config is the list of configurations to be used by the filter
	Config map[string]string `json:"config,omitempty"`
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

// FilterStatus defines the common attributes that all filters should include within
// their status.
type FilterStatus struct {
	// Conditions describes the status of the Filter with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FilterList contains a list of Filter
type FilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Filter `json:"items"`
}
