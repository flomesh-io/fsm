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
// +kubebuilder:resource:shortName=machineconnector,scope=Cluster
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
	Status MachineStatus `json:"status,omitempty"`
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

// MachineSyncToK8SSpec is the type used to represent the sync from Machine to K8S specification.
type MachineSyncToK8SSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=""
	// +optional
	ClusterId string `json:"clusterId,omitempty"`

	// +kubebuilder:default=true
	// +optional
	PassingOnly bool `json:"passingOnly,omitempty"`

	// +kubebuilder:default=""
	// +optional
	FilterLabel string `json:"filterLabel,omitempty"`

	// +kubebuilder:default=""
	// +optional
	PrefixLabel string `json:"prefixLabel,omitempty"`

	// +kubebuilder:default=""
	// +optional
	SuffixLabel string `json:"suffixLabel,omitempty"`

	// +kubebuilder:default=false
	// +optional
	WithGateway bool `json:"withGateway,omitempty"`
}

// MachineSpec is the type used to represent the Machine Connector specification.
type MachineSpec struct {
	DeriveNamespace string `json:"deriveNamespace"`

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
}

// MachineStatus is the type used to represent the status of a Machine Connector resource.
type MachineStatus struct {
	// CurrentStatus defines the current status of a Machine Connector resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a Machine Connector resource.
	// +optional
	Reason string `json:"reason,omitempty"`

	ToK8SServiceCnt int `json:"toK8SServiceCnt"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineConnectorList contains a list of Machine Connectors.
type MachineConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineConnector `json:"items"`
}
