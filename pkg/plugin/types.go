// Package plugin implements the Kubernetes client for the resources in the plugin.flomesh.io API group
package plugin

import (
	"k8s.io/client-go/kubernetes"

	pluginv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/plugin/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
)

// Client is the type used to represent the Kubernetes Client for the plugin.flomesh.io API group
type Client struct {
	informers      *informers.InformerCollection
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the plugin.flomesh.io API group
type Controller interface {
	// GetPlugins lists plugins
	GetPlugins() []*pluginv1alpha1.Plugin

	// GetPluginConfigs lists plugin configs
	GetPluginConfigs() []*pluginv1alpha1.PluginConfig

	// GetPluginChains lists plugin chains
	GetPluginChains() []*pluginv1alpha1.PluginChain
}
