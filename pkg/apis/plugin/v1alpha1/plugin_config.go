package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PluginConfig is the type used to represent a plugin config policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginConfig struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the PlugIn specification
	// +optional
	Spec PluginConfigSpec `json:"spec,omitempty"`

	// Status is the status of the plugin config configuration.
	// +optional
	Status PluginConfigStatus `json:"status,omitempty"`
}

// PluginConfigSpec is the type used to represent the plugin config specification.
type PluginConfigSpec struct {
	// Plugin is the name of plugin.
	Plugin string `json:"plugin"`

	// DestinationRefs is the destination references of plugin.
	DestinationRefs []corev1.ObjectReference `json:"destinationRefs"`

	// Config is the config of plugin.
	Config runtime.RawExtension `json:"config"`
}

// PluginConfigList defines the list of PluginConfig objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PluginConfig `json:"items"`
}

// PluginConfigStatus is the type used to represent the status of a PluginConfig resource.
type PluginConfigStatus struct {
	// CurrentStatus defines the current status of a PluginConfig resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a PluginConfig resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
