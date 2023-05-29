package trafficpolicy

import (
	pluginv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/plugin/v1alpha1"
)

// Plugin defines plugin
type Plugin struct {
	// Name defines the Name of the plugin.
	Name string

	// priority defines the priority of the plugin.
	Priority float32

	// Script defines pipy script used by the PlugIn.
	Script string

	// BuildIn indicates PlugIn type.
	BuildIn bool
}

// PluginChain defines plugin chain
type PluginChain struct {
	pluginv1alpha1.PluginChainSpec
	Name      string
	Namespace string
}

// PluginConfig defines plugin config
type PluginConfig struct {
	pluginv1alpha1.PluginConfigSpec
	Name      string
	Namespace string
}
