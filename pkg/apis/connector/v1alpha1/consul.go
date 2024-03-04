package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=consulconnector,scope=Cluster
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="HttpAddr",type=string,JSONPath=`.spec.httpAddr`
// +kubebuilder:printcolumn:name="SyncToK8S",type=string,JSONPath=`.spec.syncToK8S.enable`
// +kubebuilder:printcolumn:name="SyncFromK8S",type=string,JSONPath=`.spec.syncFromK8S.enable`

// ConsulConnector is the type used to represent a Consul Connector resource.
type ConsulConnector struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Consul Connector specification
	Spec ConsulSpec `json:"spec"`

	// Status is the status of the Consul Connector configuration.
	// +optional
	Status ConsulStatus `json:"status,omitempty"`
}

func (c *ConsulConnector) GetProvider() DiscoveryServiceProvider {
	return ConsulDiscoveryService
}

// ConsulSyncToK8SSpec is the type used to represent the sync from Consul to K8S specification.
type ConsulSyncToK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=""
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +kubebuilder:default=true
	// +optional
	PassingOnly bool `json:"passingOnly,omitempty"`

	// +kubebuilder:default=""
	// +optional
	FilterTag string `json:"filterTag,omitempty"`

	// +kubebuilder:default=""
	// +optional
	PrefixTag string `json:"prefixTag,omitempty"`

	// +kubebuilder:default=""
	// +optional
	SuffixTag string `json:"suffixTag,omitempty"`

	// +kubebuilder:default=false
	// +optional
	WithGateway bool `json:"withGateway,omitempty"`
}

// ConsulSyncFromK8SSpec is the type used to represent the sync from K8S to Consul specification.
type ConsulSyncFromK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=true
	// +optional
	DefaultSync bool `json:"defaultSync,omitempty"`

	// +kubebuilder:default=true
	// +optional
	SyncClusterIPServices bool `json:"syncClusterIPServices,omitempty"`

	// +kubebuilder:default=false
	// +optional
	SyncLoadBalancerEndpoints bool `json:"syncLoadBalancerEndpoints,omitempty"`

	// +kubebuilder:default=ExternalOnly
	// +optional
	NodePortSyncType NodePortSyncType `json:"nodePortSyncType"`

	// +kubebuilder:default=false
	// +optional
	SyncIngress bool `json:"syncIngress,omitempty"`

	// +kubebuilder:default=false
	// +optional
	SyncIngressLoadBalancerIPs bool `json:"syncIngressLoadBalancerIPs,omitempty"`

	// +kubebuilder:default=""
	// +optional
	AddServicePrefix string `json:"addServicePrefix,omitempty"`

	// +kubebuilder:default=false
	// +optional
	AddK8SNamespaceAsServiceSuffix bool `json:"addK8SNamespaceAsServiceSuffix,omitempty"`

	// +optional
	AppendTags []string `json:"appendTags,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:default={"*"}
	// +optional
	AllowK8sNamespaces []string `json:"allowK8sNamespaces,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:default={""}
	// +optional
	DenyK8sNamespaces []string `json:"denyK8sNamespaces,omitempty"`

	// +kubebuilder:default=false
	// +optional
	WithGateway bool `json:"withGateway,omitempty"`

	// +kubebuilder:default=k8s-sync
	// +optional
	ConsulNodeName string `json:"consulNodeName,omitempty"`

	// +kubebuilder:default=k8s
	// +optional
	ConsulK8STag string `json:"consulK8STag,omitempty"`

	// +kubebuilder:default=false
	// +optional
	ConsulEnableNamespaces bool `json:"consulEnableNamespaces,omitempty"`

	// +kubebuilder:default=default
	// +optional
	ConsulDestinationNamespace string `json:"consulDestinationNamespace,omitempty"`

	// +kubebuilder:default=false
	// +optional
	ConsulEnableK8SNSMirroring bool `json:"consulEnableK8SNSMirroring,omitempty"`

	// +kubebuilder:default=""
	// +optional
	ConsulK8SNSMirroringPrefix string `json:"consulK8SNSMirroringPrefix,omitempty"`

	// +kubebuilder:default=""
	// +optional
	ConsulCrossNamespaceACLPolicy string `json:"consulCrossNamespaceACLPolicy,omitempty"`
}

// ConsulSpec is the type used to represent the Consul Connector specification.
type ConsulSpec struct {
	HTTPAddr        string `json:"httpAddr"`
	DeriveNamespace string `json:"deriveNamespace"`

	// +kubebuilder:default=false
	// +optional
	AsInternalServices bool `json:"asInternalServices,omitempty"`

	SyncToK8S   ConsulSyncToK8SSpec   `json:"syncToK8S"`
	SyncFromK8S ConsulSyncFromK8SSpec `json:"syncFromK8S"`
}

// ConsulStatus is the type used to represent the status of a Consul Connector resource.
type ConsulStatus struct {
	// CurrentStatus defines the current status of a Consul Connector resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a Consul Connector resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsulConnectorList contains a list of Consul Connectors.
type ConsulConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ConsulConnector `json:"items"`
}
