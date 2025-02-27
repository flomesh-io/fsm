/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/flomesh-io/fsm/pkg/gen/client/connector/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ConsulConnectors returns a ConsulConnectorInformer.
	ConsulConnectors() ConsulConnectorInformer
	// EurekaConnectors returns a EurekaConnectorInformer.
	EurekaConnectors() EurekaConnectorInformer
	// GatewayConnectors returns a GatewayConnectorInformer.
	GatewayConnectors() GatewayConnectorInformer
	// MachineConnectors returns a MachineConnectorInformer.
	MachineConnectors() MachineConnectorInformer
	// NacosConnectors returns a NacosConnectorInformer.
	NacosConnectors() NacosConnectorInformer
	// ZookeeperConnectors returns a ZookeeperConnectorInformer.
	ZookeeperConnectors() ZookeeperConnectorInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// ConsulConnectors returns a ConsulConnectorInformer.
func (v *version) ConsulConnectors() ConsulConnectorInformer {
	return &consulConnectorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// EurekaConnectors returns a EurekaConnectorInformer.
func (v *version) EurekaConnectors() EurekaConnectorInformer {
	return &eurekaConnectorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// GatewayConnectors returns a GatewayConnectorInformer.
func (v *version) GatewayConnectors() GatewayConnectorInformer {
	return &gatewayConnectorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// MachineConnectors returns a MachineConnectorInformer.
func (v *version) MachineConnectors() MachineConnectorInformer {
	return &machineConnectorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// NacosConnectors returns a NacosConnectorInformer.
func (v *version) NacosConnectors() NacosConnectorInformer {
	return &nacosConnectorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ZookeeperConnectors returns a ZookeeperConnectorInformer.
func (v *version) ZookeeperConnectors() ZookeeperConnectorInformer {
	return &zookeeperConnectorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
