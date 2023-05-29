package plugin

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	pluginv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/plugin/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// NewPluginController returns a plugin.Controller interface related to functionality provided by the resources in the plugin.flomesh.io API group
func NewPluginController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObservePlugin := func(obj interface{}) bool {
		return true
	}

	pluginEventTypes := k8s.EventTypes{
		Add:    announcements.PluginAdded,
		Update: announcements.PluginUpdated,
		Delete: announcements.PluginDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyPlugin, k8s.GetEventHandlerFuncs(shouldObservePlugin, pluginEventTypes, msgBroker))

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	pluginChainEventTypes := k8s.EventTypes{
		Add:    announcements.PluginChainAdded,
		Update: announcements.PluginChainUpdated,
		Delete: announcements.PluginChainDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyPluginChain, k8s.GetEventHandlerFuncs(shouldObserve, pluginChainEventTypes, msgBroker))

	pluginServiceEventTypes := k8s.EventTypes{
		Add:    announcements.PluginConfigAdded,
		Update: announcements.PluginConfigUpdated,
		Delete: announcements.PluginConfigDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyPluginConfig, k8s.GetEventHandlerFuncs(shouldObserve, pluginServiceEventTypes, msgBroker))

	return client
}

// GetPlugins lists plugins
func (c *Client) GetPlugins() []*pluginv1alpha1.Plugin {
	var plugins []*pluginv1alpha1.Plugin
	for _, pluginIface := range c.informers.List(informers.InformerKeyPlugin) {
		plugin := pluginIface.(*pluginv1alpha1.Plugin)
		plugins = append(plugins, plugin)
	}
	return plugins
}

// GetPluginConfigs lists plugin configs
func (c *Client) GetPluginConfigs() []*pluginv1alpha1.PluginConfig {
	var pluginConfigs []*pluginv1alpha1.PluginConfig
	for _, pluginConfigIface := range c.informers.List(informers.InformerKeyPluginConfig) {
		pluginConfig := pluginConfigIface.(*pluginv1alpha1.PluginConfig)
		pluginConfigs = append(pluginConfigs, pluginConfig)
	}
	return pluginConfigs
}

// GetPluginChains lists plugin chains
func (c *Client) GetPluginChains() []*pluginv1alpha1.PluginChain {
	var pluginChains []*pluginv1alpha1.PluginChain
	for _, pluginChainIface := range c.informers.List(informers.InformerKeyPluginChain) {
		pluginChain := pluginChainIface.(*pluginv1alpha1.PluginChain)
		pluginChains = append(pluginChains, pluginChain)
	}
	return pluginChains
}
