package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginChain is the type used to represent a PluginChain.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginChain struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the PluginChain specification
	// +optional
	Spec PluginChainSpec `json:"spec,omitempty"`

	// Status is the status of the PluginChain configuration.
	// +optional
	Status PluginChainStatus `json:"status,omitempty"`
}

// PluginChainSpec is the type used to represent the PluginChain specification.
type PluginChainSpec struct {
	// Chains defines the plugins within chains
	Chains []ChainPluginSpec `json:"chains"`

	// Selectors defines the selectors of chains.
	Selectors ChainSelectorSpec `json:"selectors"`
}

// ChainPluginSpec is the type used to represent plugins within chain.
type ChainPluginSpec struct {
	// Name defines the name of chain.
	Name string `json:"name"`

	// Plugins defines the plugins within chain.
	Plugins []string `json:"plugins"`
}

// ChainSelectorSpec is the type used to represent plugins for plugin chain.
type ChainSelectorSpec struct {
	// PodSelector for pods. Existing pods are selected by this will be the ones affected by this plugin chain.
	PodSelector *metav1.LabelSelector `json:"podSelector"`

	// NamespaceSelector for namespaces. Existing pods are selected by this will be the ones affected by this plugin chain.
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector"`
}

// PluginChainList defines the list of PluginChain objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginChainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PluginChain `json:"items"`
}

// PluginChainStatus is the type used to represent the status of a PluginChain resource.
type PluginChainStatus struct {
	// CurrentStatus defines the current status of a PluginChain resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a PluginChain resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
