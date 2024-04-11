package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=eurekaconnector,scope=Cluster
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="HttpAddr",type=string,JSONPath=`.spec.httpAddr`
// +kubebuilder:printcolumn:name="SyncToK8S",type=string,JSONPath=`.spec.syncToK8S.enable`
// +kubebuilder:printcolumn:name="SyncFromK8S",type=string,JSONPath=`.spec.syncFromK8S.enable`
// +kubebuilder:printcolumn:name="toK8SServices",type=integer,JSONPath=`.status.toK8SServiceCnt`
// +kubebuilder:printcolumn:name="fromK8SServices",type=integer,JSONPath=`.status.fromK8SServiceCnt`

// EurekaConnector is the type used to represent a Eureka Connector resource.
type EurekaConnector struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Eureka Connector specification
	Spec EurekaSpec `json:"spec"`

	// Status is the status of the Eureka Connector configuration.
	// +optional
	Status EurekaStatus `json:"status,omitempty"`
}

func (c *EurekaConnector) GetProvider() DiscoveryServiceProvider {
	return EurekaDiscoveryService
}

func (c *EurekaConnector) GetReplicas() *int32 {
	return c.Spec.Replicas
}

func (c *EurekaConnector) GetResources() *corev1.ResourceRequirements {
	return &c.Spec.Resources
}

// EurekaSyncToK8SSpec is the type used to represent the sync from Eureka to K8S specification.
type EurekaSyncToK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=""
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +optional
	FilterMetadatas []Metadata `json:"filterMetadatas,omitempty"`

	// +optional
	PrefixMetadata string `json:"prefixMetadata,omitempty"`

	// +optional
	SuffixMetadata string `json:"suffixMetadata,omitempty"`

	// +kubebuilder:default=false
	// +optional
	WithGateway bool `json:"withGateway,omitempty"`
}

// EurekaSyncFromK8SSpec is the type used to represent the sync from K8S to Eureka specification.
type EurekaSyncFromK8SSpec struct {
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
	AppendMetadatas []Metadata `json:"appendMetadatas,omitempty"`

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

	// +kubebuilder:default=forward
	// +optional
	WithGatewayMode WithGatewayMode `json:"withGatewayMode,omitempty"`
}

// EurekaSpec is the type used to represent the Eureka Connector specification.
type EurekaSpec struct {
	HTTPAddr        string `json:"httpAddr"`
	DeriveNamespace string `json:"deriveNamespace"`

	// +kubebuilder:default=false
	// +optional
	AsInternalServices bool `json:"asInternalServices,omitempty"`

	// +kubebuilder:validation:Format="duration"
	// +kubebuilder:default="5s"
	// +optional
	SyncPeriod  metav1.Duration       `json:"syncPeriod"`
	SyncToK8S   EurekaSyncToK8SSpec   `json:"syncToK8S"`
	SyncFromK8S EurekaSyncFromK8SSpec `json:"syncFromK8S"`

	// +kubebuilder:default={limit:500, burst:750}
	// +optional
	Limiter *Limiter `json:"Limiter,omitempty"`

	// Compute Resources required by connector container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

// EurekaStatus is the type used to represent the status of a Eureka Connector resource.
type EurekaStatus struct {
	// CurrentStatus defines the current status of a Eureka Connector resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a Eureka Connector resource.
	// +optional
	Reason string `json:"reason,omitempty"`

	ToK8SServiceCnt int `json:"toK8SServiceCnt"`

	FromK8SServiceCnt int `json:"fromK8SServiceCnt"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EurekaConnectorList contains a list of Eureka Connectors.
type EurekaConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EurekaConnector `json:"items"`
}
