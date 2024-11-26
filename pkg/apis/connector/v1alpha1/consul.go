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
// +kubebuilder:resource:shortName=consulconnector,scope=Cluster
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="HttpAddr",type=string,JSONPath=`.spec.httpAddr`
// +kubebuilder:printcolumn:name="SyncToK8S",type=string,JSONPath=`.spec.syncToK8S.enable`
// +kubebuilder:printcolumn:name="SyncFromK8S",type=string,JSONPath=`.spec.syncFromK8S.enable`
// +kubebuilder:printcolumn:name="toK8SServices",type=integer,JSONPath=`.status.toK8SServiceCnt`
// +kubebuilder:printcolumn:name="fromK8SServices",type=integer,JSONPath=`.status.fromK8SServiceCnt`

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

func (c *ConsulConnector) GetReplicas() *int32 {
	return c.Spec.Replicas
}

func (c *ConsulConnector) GetResources() *corev1.ResourceRequirements {
	return &c.Spec.Resources
}

func (c *ConsulConnector) GetImagePullSecrets() []corev1.LocalObjectReference {
	return c.Spec.ImagePullSecrets
}

func (c *ConsulConnector) GetLeaderElection() *bool {
	return c.Spec.LeaderElection
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

	// +optional
	FilterIPRanges []string `json:"filterIpRanges,omitempty"`

	// +optional
	ExcludeIPRanges []string `json:"excludeIpRanges,omitempty"`

	// +optional
	FilterTag string `json:"filterTag,omitempty"`

	// +optional
	PrefixTag string `json:"prefixTag,omitempty"`

	// +optional
	SuffixTag string `json:"suffixTag,omitempty"`

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

	// +optional
	FixedGRPCServicePort *uint32 `json:"fixedGrpcServicePort,omitempty"`

	// +kubebuilder:default={enable: false, multiGateways: true}
	// +optional
	WithGateway C2KGateway `json:"withGateway,omitempty"`

	// +kubebuilder:default=false
	// +optional
	GenerateInternalServiceHealthCheck bool `json:"generateInternalServiceHealthCheck,omitempty"`
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

	// +optional
	ConsulNodeName string `json:"consulNodeName,omitempty"`

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
	SyncPeriod  metav1.Duration       `json:"syncPeriod"`
	SyncToK8S   ConsulSyncToK8SSpec   `json:"syncToK8S"`
	SyncFromK8S ConsulSyncFromK8SSpec `json:"syncFromK8S"`

	// +kubebuilder:default={limit:500, burst:750}
	// +optional
	Limiter *Limiter `json:"limiter,omitempty"`

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

// ConsulAuthSpec is the type used to represent the Consul auth specification.
type ConsulAuthSpec struct {
	// +kubebuilder:default=""
	// +optional
	Username string `json:"username,omitempty"`

	// +kubebuilder:default=""
	// +optional
	Password string `json:"password,omitempty"`
}

// ConsulStatus is the type used to represent the status of a Consul Connector resource.
type ConsulStatus struct {
	// CurrentStatus defines the current status of a Consul Connector resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a Consul Connector resource.
	// +optional
	Reason string `json:"reason,omitempty"`

	ToK8SServiceCnt int `json:"toK8SServiceCnt"`

	FromK8SServiceCnt int `json:"fromK8SServiceCnt"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConsulConnectorList contains a list of Consul Connectors.
type ConsulConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ConsulConnector `json:"items"`
}
