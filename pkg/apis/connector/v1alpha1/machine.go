package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=machineconnector,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SyncToK8S",type=string,JSONPath=`.spec.syncToK8S.enable`
// +kubebuilder:printcolumn:name="toK8SServices",type=integer,JSONPath=`.status.toK8SServiceCnt`

// MachineConnector is the type used to represent a Machine Connector resource.
type MachineConnector struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Machine Connector specification
	Spec MachineSpec `json:"spec"`

	// Status is the status of the Machine Connector configuration.
	// +optional
	Status ConnectorStatus `json:"status,omitempty"`
}

func (c *MachineConnector) GetProvider() DiscoveryServiceProvider {
	return MachineDiscoveryService
}

func (c *MachineConnector) GetReplicas() *int32 {
	return c.Spec.Replicas
}

func (c *MachineConnector) GetResources() *corev1.ResourceRequirements {
	return &c.Spec.Resources
}

func (c *MachineConnector) GetImagePullSecrets() []corev1.LocalObjectReference {
	return c.Spec.ImagePullSecrets
}

func (c *MachineConnector) GetLeaderElection() *bool {
	return c.Spec.LeaderElection
}

// MachineSyncToK8SSpec is the type used to represent the sync from Machine to K8S specification.
type MachineSyncToK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=""
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +kubebuilder:default=true
	// +optional
	PassingOnly bool `json:"passingOnly,omitempty"`

	// +optional
	// +optional
	FilterIPRanges []string `json:"filterIpRanges,omitempty"`

	// +optional
	ExcludeIPRanges []string `json:"excludeIpRanges,omitempty"`

	// +kubebuilder:default=""
	// +optional
	FilterLabel string `json:"filterLabel,omitempty"`

	// +kubebuilder:default=""
	// +optional
	PrefixLabel string `json:"prefixLabel,omitempty"`

	// +kubebuilder:default=""
	// +optional
	SuffixLabel string `json:"suffixLabel,omitempty"`

	// +kubebuilder:default={enable: false, multiGateways: true}
	// +optional
	WithGateway C2KGateway `json:"withGateway,omitempty"`

	// +optional
	AppendLabels map[string]string `json:"appendLabels,omitempty"`

	// +optional
	AppendAnnotations map[string]string `json:"appendAnnotations,omitempty"`

	// +optional
	ConversionStrategy *ConversionStrategy `json:"conversionStrategy,omitempty"`
}

// MachineSpec is the type used to represent the Machine Connector specification.
type MachineSpec struct {
	DeriveNamespace string `json:"deriveNamespace"`

	// +kubebuilder:default=false
	// +optional
	Purge bool `json:"purge,omitempty"`

	// +kubebuilder:default=false
	// +optional
	AsInternalServices bool `json:"asInternalServices,omitempty"`

	SyncToK8S MachineSyncToK8SSpec `json:"syncToK8S"`

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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineConnectorList contains a list of Machine Connectors.
type MachineConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineConnector `json:"items"`
}
