package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Plugin is the type used to represent a Plugin policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Plugin struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the PlugIn specification
	// +optional
	Spec PluginSpec `json:"spec,omitempty"`

	// Status is the status of the Plugin configuration.
	// +optional
	Status PluginStatus `json:"status,omitempty"`
}

// PluginSpec is the type used to represent the Plugin policy specification.
type PluginSpec struct {
	// priority defines the priority of the plugin.
	Priority *float32 `json:"priority,omitempty"`

	// Script defines the Script of the plugin.
	Script string `json:"pipyscript"`
}

// PluginList defines the list of Plugin objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Plugin `json:"items"`
}

// PluginStatus is the type used to represent the status of a Plugin resource.
type PluginStatus struct {
	// CurrentStatus defines the current status of a Plugin resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a Plugin resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
