package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=nacosconnector,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="HttpAddr",type=string,JSONPath=`.spec.httpAddr`
// +kubebuilder:printcolumn:name="SyncToK8S",type=string,JSONPath=`.spec.syncToK8S.enable`
// +kubebuilder:printcolumn:name="SyncFromK8S",type=string,JSONPath=`.spec.syncFromK8S.enable`
// +kubebuilder:printcolumn:name="toK8SServices",type=integer,JSONPath=`.status.toK8SServiceCnt`
// +kubebuilder:printcolumn:name="fromK8SServices",type=integer,JSONPath=`.status.fromK8SServiceCnt`

// NacosConnector is the type used to represent a Nacos Connector resource.
type NacosConnector struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Nacos Connector specification
	Spec NacosSpec `json:"spec"`

	// Status is the status of the Nacos Connector configuration.
	// +optional
	Status ConnectorStatus `json:"status,omitempty"`
}

func (c *NacosConnector) GetProvider() DiscoveryServiceProvider {
	return NacosDiscoveryService
}

func (c *NacosConnector) GetReplicas() *int32 {
	return c.Spec.Replicas
}

func (c *NacosConnector) GetResources() *corev1.ResourceRequirements {
	return &c.Spec.Resources
}

func (c *NacosConnector) GetImagePullSecrets() []corev1.LocalObjectReference {
	return c.Spec.ImagePullSecrets
}

func (c *NacosConnector) GetLeaderElection() *bool {
	return c.Spec.LeaderElection
}

// NacosSyncToK8SSpec is the type used to represent the sync from Nacos to K8S specification.
type NacosSyncToK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=""
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +kubebuilder:default=true
	// +optional
	PassingOnly bool `json:"passingOnly,omitempty"`

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

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:default={"DEFAULT"}
	// +optional
	ClusterSet []string `json:"clusterSet,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:default={"DEFAULT_GROUP"}
	// +optional
	GroupSet []string `json:"groupSet,omitempty"`

	// +kubebuilder:default={enable: false, multiGateways: true}
	// +optional
	WithGateway C2KGateway `json:"withGateway,omitempty"`

	// +optional
	AppendLabels map[string]string `json:"appendLabels,omitempty"`

	// +optional
	AppendAnnotations map[string]string `json:"appendAnnotations,omitempty"`

	// +optional
	MetadataStrategy *MetadataStrategy `json:"metadataStrategy,omitempty"`

	// +optional
	ConversionStrategy *ConversionStrategy `json:"conversionStrategy,omitempty"`
}

// NacosSyncFromK8SSpec is the type used to represent the sync from K8S to Nacos specification.
type NacosSyncFromK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=DEFAULT
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +kubebuilder:default=DEFAULT_GROUP
	// +optional
	GroupId string `json:"groupId,omitempty"`

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

	// +kubebuilder:default=""
	// +optional
	AddServicePrefix string `json:"addServicePrefix,omitempty"`

	// +kubebuilder:default=false
	// +optional
	AddK8SNamespaceAsServiceSuffix bool `json:"addK8SNamespaceAsServiceSuffix,omitempty"`

	// +optional
	AppendMetadatas []Metadata `json:"appendMetadatas,omitempty"`

	// +optional
	MetadataStrategy *MetadataStrategy `json:"metadataStrategy,omitempty"`
}

// NacosSpec is the type used to represent the Nacos Connector specification.
type NacosSpec struct {
	HTTPAddr        string `json:"httpAddr"`
	DeriveNamespace string `json:"deriveNamespace"`

	// +kubebuilder:default=false
	// +optional
	Purge bool `json:"purge,omitempty"`

	// +kubebuilder:default=false
	// +optional
	AsInternalServices bool `json:"asInternalServices,omitempty"`

	// +kubebuilder:default={}
	// +optional
	Auth NacosAuthSpec `json:"auth,omitempty"`

	// +kubebuilder:validation:Format="duration"
	// +kubebuilder:default="5s"
	// +optional
	SyncPeriod  metav1.Duration      `json:"syncPeriod"`
	SyncToK8S   NacosSyncToK8SSpec   `json:"syncToK8S"`
	SyncFromK8S NacosSyncFromK8SSpec `json:"syncFromK8S"`

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

// NacosAuthSpec is the type used to represent the Nacos auth specification.
type NacosAuthSpec struct {
	// +kubebuilder:default=""
	// +optional
	Username string `json:"username,omitempty"`

	// +kubebuilder:default=""
	// +optional
	Password string `json:"password,omitempty"`

	// +kubebuilder:default=""
	// +optional
	AccessKey string `json:"accessKey,omitempty"`

	// +kubebuilder:default=""
	// +optional
	SecretKey string `json:"secretKey,omitempty"`

	// +kubebuilder:default=public
	// +optional
	NamespaceId string `json:"namespaceId,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NacosConnectorList contains a list of Nacos Connectors.
type NacosConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NacosConnector `json:"items"`
}
