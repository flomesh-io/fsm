package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ListenerFilterSpec defines the desired state of ListenerFilter
type ListenerFilterSpec struct {
	// Type is the type of the ListenerFilter in PascalCase, it should be unique within the namespace
	Type FilterType `json:"type"`

	// +optional
	// +kubebuilder:default=100
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10000
	// Priority is the priority of the ListenerFilter, it is used to determine the order of the ListenerFilters, the ListenerFilter with the lowest priority is applied first
	Priority *int32 `json:"priority,omitempty"`

	// +optional
	// +kubebuilder:default=Listener
	// +kubebuilder:validation:Enum=Route;Listener
	// Aspect is the aspect of the ListenerFilter, default is Listener
	// If the aspect is Route, the ListenerFilter is applied to all Routes attached to this listener and is executed before any Route level filters
	// If the aspect is Listener, the ListenerFilter is executed when request is received by the listener
	Aspect *FilterAspect `json:"aspect,omitempty"`

	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// TargetRefs is the references to the target resources to which the ListenerFilter is applied
	TargetRefs []LocalTargetReferenceWithPort `json:"targetRefs"`

	// +optional
	// +nullable
	// +kubebuilder:validation:Type=object
	// DefinitionRef is the reference to the FilterDefinition
	DefinitionRef *gwv1.LocalObjectReference `json:"definitionRef"`

	// +optional
	// +nullable
	// +kubebuilder:validation:Type=object
	// ConfigRef is the reference to the Configurations
	ConfigRef *gwv1.LocalObjectReference `json:"configRef,omitempty"`
}

// ListenerFilter provides a way to configure ListenerFilters for HTTP/HTTPS/GRPC/GRPCS/TCP protocols
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.flomesh.io/extension=Filter}
type ListenerFilter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of ListenerFilter.
	Spec ListenerFilterSpec `json:"spec,omitempty"`

	// Status defines the current state of ListenerFilter.
	Status ListenerFilterStatus `json:"status,omitempty"`
}

// ListenerFilterStatus defines the common attributes that all ListenerFilters should include within
// their status.
type ListenerFilterStatus struct {
	// Conditions describes the status of the ListenerFilter with respect to the given Ancestor.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ListenerFilterList contains a list of ListenerFilter
type ListenerFilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ListenerFilter `json:"items"`
}
