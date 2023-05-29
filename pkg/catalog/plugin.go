package catalog

import (
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

// GetPlugins returns the plugin policies
func (mc *MeshCatalog) GetPlugins() []*trafficpolicy.Plugin {
	if !mc.configurator.GetFeatureFlags().EnablePluginPolicy {
		return nil
	}

	plugins := mc.pluginController.GetPlugins()
	if plugins == nil {
		log.Trace().Msg("Did not find any plugin")
		return nil
	}

	var pluginPolicies []*trafficpolicy.Plugin
	for _, plugin := range plugins {
		policy := new(trafficpolicy.Plugin)
		policy.Name = plugin.Name
		if plugin.Spec.Priority != nil {
			policy.Priority = *plugin.Spec.Priority
		}
		policy.Script = plugin.Spec.Script
		pluginPolicies = append(pluginPolicies, policy)
	}
	return pluginPolicies
}

// GetPluginConfigs lists plugin configs
func (mc *MeshCatalog) GetPluginConfigs() []*trafficpolicy.PluginConfig {
	if !mc.configurator.GetFeatureFlags().EnablePluginPolicy {
		return nil
	}

	pluginConfigs := mc.pluginController.GetPluginConfigs()
	if pluginConfigs == nil {
		log.Trace().Msg("Did not find any plugin config")
		return nil
	}

	var pluginConfigPolicies []*trafficpolicy.PluginConfig
	for _, pluginConfig := range pluginConfigs {
		policy := new(trafficpolicy.PluginConfig)
		policy.Namespace = pluginConfig.Namespace
		policy.Name = pluginConfig.Name
		policy.PluginConfigSpec = pluginConfig.Spec
		pluginConfigPolicies = append(pluginConfigPolicies, policy)
	}

	return pluginConfigPolicies
}

// GetPluginChains lists plugin chains
func (mc *MeshCatalog) GetPluginChains() []*trafficpolicy.PluginChain {
	if !mc.configurator.GetFeatureFlags().EnablePluginPolicy {
		return nil
	}

	pluginChains := mc.pluginController.GetPluginChains()
	if pluginChains == nil {
		log.Trace().Msg("Did not find any plugin chain")
		return nil
	}

	var pluginChainPolicies []*trafficpolicy.PluginChain
	for _, pluginChain := range pluginChains {
		policy := new(trafficpolicy.PluginChain)
		policy.Namespace = pluginChain.Namespace
		policy.Name = pluginChain.Name
		policy.PluginChainSpec = pluginChain.Spec
		pluginChainPolicies = append(pluginChainPolicies, policy)
	}

	return pluginChainPolicies
}
