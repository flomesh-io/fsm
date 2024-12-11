package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:validation:Enum=ExternalOnly;InternalOnly;ExternalFirst
type NodePortSyncType string

const (
	// ExternalOnly only sync NodePort services with a node's ExternalIP address.
	// Doesn't sync if an ExternalIP doesn't exist.
	ExternalOnly NodePortSyncType = "ExternalOnly"

	// InternalOnly sync NodePort services using.
	InternalOnly NodePortSyncType = "InternalOnly"

	// ExternalFirst sync with an ExternalIP first, if it doesn't exist, use the
	// node's InternalIP address instead.
	ExternalFirst NodePortSyncType = "ExternalFirst"
)

// +kubebuilder:validation:Enum=ExternalIP;ClusterIP
type AddrSelector string

const (
	ExternalIP AddrSelector = "ExternalIP"
	ClusterIP  AddrSelector = "ClusterIP"
)

// +kubebuilder:validation:Enum=proxy;forward
type WithGatewayMode string

const (
	Proxy   WithGatewayMode = "proxy"
	Forward WithGatewayMode = "forward"
)

type K2CGateway struct {
	// +kubebuilder:default=false
	// +optional
	Enable bool `json:"enable,omitempty"`

	// +kubebuilder:default=forward
	// +optional
	GatewayMode WithGatewayMode `json:"gatewayMode,omitempty"`
}

type C2KGateway struct {
	// +kubebuilder:default=false
	// +optional
	Enable bool `json:"enable,omitempty"`

	// +kubebuilder:default=true
	// +optional
	MultiGateways bool `json:"multiGateways,omitempty"`
}

type Connector interface {
	runtime.Object
	metav1.Object
	GetProvider() DiscoveryServiceProvider
	GetReplicas() *int32
	GetResources() *corev1.ResourceRequirements
	GetImagePullSecrets() []corev1.LocalObjectReference
	GetLeaderElection() *bool
}

type DiscoveryServiceProvider string

const (
	//ConsulDiscoveryService defines consul discovery service name
	ConsulDiscoveryService DiscoveryServiceProvider = "consul"

	//EurekaDiscoveryService defines eureka discovery service name
	EurekaDiscoveryService DiscoveryServiceProvider = "eureka"

	//NacosDiscoveryService defines nacos discovery service name
	NacosDiscoveryService DiscoveryServiceProvider = "nacos"

	//ZookeeperDiscoveryService defines zookeeper discovery service name
	ZookeeperDiscoveryService DiscoveryServiceProvider = "zookeeper"

	//MachineDiscoveryService defines machine discovery service name
	MachineDiscoveryService DiscoveryServiceProvider = "machine"

	//GatewayDiscoveryService defines gateway integrated service name
	GatewayDiscoveryService DiscoveryServiceProvider = "gateway"
)

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Limiter struct {
	Limit uint32 `json:"limit"`
	Burst uint32 `json:"burst"`
}
