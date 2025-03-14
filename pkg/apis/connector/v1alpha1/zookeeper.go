package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=zookeeperconnector,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="HttpAddr",type=string,JSONPath=`.spec.httpAddr`
// +kubebuilder:printcolumn:name="SyncToK8S",type=string,JSONPath=`.spec.syncToK8S.enable`
// +kubebuilder:printcolumn:name="SyncFromK8S",type=string,JSONPath=`.spec.syncFromK8S.enable`
// +kubebuilder:printcolumn:name="toK8SServices",type=integer,JSONPath=`.status.toK8SServiceCnt`
// +kubebuilder:printcolumn:name="fromK8SServices",type=integer,JSONPath=`.status.fromK8SServiceCnt`

// ZookeeperConnector is the type used to represent a Zookeeper Connector resource.
type ZookeeperConnector struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Zookeeper Connector specification
	Spec ZookeeperSpec `json:"spec"`

	// Status is the status of the Zookeeper Connector configuration.
	// +optional
	Status ConnectorStatus `json:"status,omitempty"`
}

func (c *ZookeeperConnector) GetProvider() DiscoveryServiceProvider {
	return ZookeeperDiscoveryService
}

func (c *ZookeeperConnector) GetReplicas() *int32 {
	return c.Spec.Replicas
}

func (c *ZookeeperConnector) GetResources() *corev1.ResourceRequirements {
	return &c.Spec.Resources
}

func (c *ZookeeperConnector) GetImagePullSecrets() []corev1.LocalObjectReference {
	return c.Spec.ImagePullSecrets
}

func (c *ZookeeperConnector) GetLeaderElection() *bool {
	return c.Spec.LeaderElection
}

// ZookeeperSyncToK8SSpec is the type used to represent the sync from Zookeeper to K8S specification.
type ZookeeperSyncToK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=""
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +optional
	FilterIPRanges []string `json:"filterIpRanges,omitempty"`

	// +optional
	ExcludeIPRanges []string `json:"excludeIpRanges,omitempty"`

	// +optional
	FilterMetadatas []Metadata `json:"filterMetadatas,omitempty"`

	// +optional
	ExcludeMetadatas []Metadata `json:"excludeMetadatas,omitempty"`

	// +optional
	PrefixMetadata string `json:"prefixMetadata,omitempty"`

	// +optional
	SuffixMetadata string `json:"suffixMetadata,omitempty"`

	// +optional
	FixedHTTPServicePort *uint32 `json:"fixedHttpServicePort,omitempty"`

	// +kubebuilder:default={enable: false, multiGateways: true}
	// +optional
	WithGateway C2KGateway `json:"withGateway,omitempty"`

	// +optional
	MetadataStrategy *MetadataStrategy `json:"metadataStrategy,omitempty"`

	// +optional
	ConversionStrategy *ConversionStrategy `json:"conversionStrategy,omitempty"`
}

// ZookeeperSyncFromK8SSpec is the type used to represent the sync from K8S to Zookeeper specification.
type ZookeeperSyncFromK8SSpec struct {
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

	// +optional
	// +optional
	FilterIPRanges []string `json:"filterIpRanges,omitempty"`

	// +optional
	ExcludeIPRanges []string `json:"excludeIpRanges,omitempty"`

	// +kubebuilder:default={enable: false, gatewayMode: forward}
	// +optional
	WithGateway K2CGateway `json:"withGateway,omitempty"`
}

// ZookeeperSpec is the type used to represent the Zookeeper Connector specification.
type ZookeeperSpec struct {
	HTTPAddr        string `json:"httpAddr"`
	DeriveNamespace string `json:"deriveNamespace"`

	BasePath string `json:"basePath"`
	Category string `json:"category"`
	Adaptor  string `json:"adaptor"`

	// +kubebuilder:default=false
	// +optional
	Purge bool `json:"purge,omitempty"`

	// +kubebuilder:default=false
	// +optional
	AsInternalServices bool `json:"asInternalServices,omitempty"`

	// +kubebuilder:default={}
	// +optional
	Auth ZookeeperAuthSpec `json:"auth,omitempty"`

	// +kubebuilder:validation:Format="duration"
	// +kubebuilder:default="5s"
	// +optional
	SyncPeriod  metav1.Duration          `json:"syncPeriod"`
	SyncToK8S   ZookeeperSyncToK8SSpec   `json:"syncToK8S"`
	SyncFromK8S ZookeeperSyncFromK8SSpec `json:"syncFromK8S"`

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

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +kubebuilder:default=true
	// +optional
	LeaderElection *bool `json:"leaderElection,omitempty"`
}

// ZookeeperAuthSpec is the type used to represent the Zookeeper auth specification.
type ZookeeperAuthSpec struct {
	// +kubebuilder:default=""
	// +optional
	Password string `json:"password,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ZookeeperConnectorList contains a list of Zookeeper Connectors.
type ZookeeperConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ZookeeperConnector `json:"items"`
}
