package v1alpha1

import (
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type LocalTargetReferenceWithPort struct {
	// Group is the group of the target resource.
	Group gwv1.Group `json:"group"`

	// Kind is kind of the target resource.
	Kind gwv1.Kind `json:"kind"`

	// Name is the name of the target resource.
	Name gwv1.ObjectName `json:"name"`

	// port is the port of the target listener.
	Port gwv1.PortNumber `json:"port"`
}

// +kubebuilder:validation:Pattern=`^[A-Z]+(([A-Z]*[a-z0-9]+[A-Z]*)*)$`
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=63

// FilterType defines the type of filter
type FilterType string

// FilterScope defines the scope of filter
type FilterScope string

const (
	// FilterScopeListener is the type of filter for listener
	FilterScopeListener FilterScope = "Listener"

	// FilterScopeRoute is the type of filter for route
	FilterScopeRoute FilterScope = "Route"
)

// FilterProtocol defines the protocol of filter
type FilterProtocol string

const (
	// FilterProtocolHTTP is the type of filter for HTTP/HTTPS/GRPC/GRPCS protocols
	FilterProtocolHTTP FilterProtocol = "http"

	// FilterProtocolTCP is the type of filter for TCP protocol
	FilterProtocolTCP FilterProtocol = "tcp"

	// FilterProtocolUDP is the type of filter for UDP protocol
	FilterProtocolUDP FilterProtocol = "udp"
)

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

type FilterAspect string

const (
	// FilterAspectListener is the aspect of filter for listener
	FilterAspectListener FilterAspect = "Listener"

	// FilterAspectRoute is the aspect of filter for route
	FilterAspectRoute FilterAspect = "Route"
)

// HostPort is a host name with optional port number
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*(:[0-9]{1,5})?$`
type HostPort string
