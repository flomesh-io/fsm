package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:validation:Enum=ExternalOnly;InternalOnly;ExternalFirst
type NodePortSyncType string

const (
	ExternalOnly  NodePortSyncType = "ExternalOnly"
	InternalOnly  NodePortSyncType = "InternalOnly"
	ExternalFirst NodePortSyncType = "ExternalFirst"
)

// +kubebuilder:validation:Enum=ExternalIP;ClusterIP
type AddrSelector string

const (
	ExternalIP AddrSelector = "ExternalIP"
	ClusterIP  AddrSelector = "ClusterIP"
)

type Connector interface {
	runtime.Object
	metav1.Object
	GetProvider() DiscoveryServiceProvider
}

type DiscoveryServiceProvider string

const (
	//ConsulDiscoveryService defines consul discovery service name
	ConsulDiscoveryService DiscoveryServiceProvider = "consul"

	//EurekaDiscoveryService defines eureka discovery service name
	EurekaDiscoveryService DiscoveryServiceProvider = "eureka"

	//NacosDiscoveryService defines nacos discovery service name
	NacosDiscoveryService DiscoveryServiceProvider = "nacos"

	//MachineDiscoveryService defines machine discovery service name
	MachineDiscoveryService DiscoveryServiceProvider = "machine"

	//GatewayDiscoveryService defines gateway integrated service name
	GatewayDiscoveryService DiscoveryServiceProvider = "gateway"
)

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
