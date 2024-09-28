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
// +kubebuilder:resource:shortName=gatewayconnector,scope=Cluster
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SyncToFgw",type=string,JSONPath=`.spec.syncToFgw.enable`

// GatewayConnector is the type used to represent a Gateway Connector resource.
type GatewayConnector struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Gateway Connector specification
	Spec GatewaySpec `json:"spec"`

	// Status is the status of the Gateway Connector configuration.
	// +optional
	Status GatewayStatus `json:"status,omitempty"`
}

func (c *GatewayConnector) GetProvider() DiscoveryServiceProvider {
	return GatewayDiscoveryService
}

func (c *GatewayConnector) GetReplicas() *int32 {
	return c.Spec.Replicas
}

func (c *GatewayConnector) GetResources() *corev1.ResourceRequirements {
	return &c.Spec.Resources
}

func (c *GatewayConnector) GetLeaderElection() *bool {
	return c.Spec.LeaderElection
}

// IngressSelectorSpec is the type used to represent the ingress selector specification.
type IngressSelectorSpec struct {
	// +kubebuilder:default=ExternalIP
	// +optional
	IPSelector AddrSelector `json:"ipSelector,omitempty"`

	// +kubebuilder:default=10080
	// +optional
	HTTPPort int32 `json:"httpPort,omitempty"`

	// +optional
	GRPCPort int32 `json:"grpcPort,omitempty"`
}

// EgressSelectorSpec is the type used to represent the egress selector specification.
type EgressSelectorSpec struct {
	// +kubebuilder:default=ClusterIP
	// +optional
	IPSelector AddrSelector `json:"ipSelector,omitempty"`

	// +kubebuilder:default=10090
	// +optional
	HTTPPort int32 `json:"httpPort,omitempty"`

	// +optional
	GRPCPort int32 `json:"grpcPort,omitempty"`
}

// SyncToFgwSpec is the type used to represent the sync to Gateway specification.
type SyncToFgwSpec struct {
	Enable bool `json:"enable"`

	// +kubebuilder:default=false
	// +optional
	Purge bool `json:"purge,omitempty"`

	// +kubebuilder:validation:Format="duration"
	// +kubebuilder:default="5s"
	// +optional
	SyncPeriod metav1.Duration `json:"syncPeriod"`

	// +kubebuilder:default=true
	// +optional
	DefaultSync bool `json:"defaultSync,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:default={"*"}
	// +optional
	AllowK8sNamespaces []string `json:"allowK8sNamespaces,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:default={""}
	// +optional
	DenyK8sNamespaces []string `json:"denyK8sNamespaces,omitempty"`
}

// GatewaySpec is the type used to represent the Gateway Connector specification.
type GatewaySpec struct {
	GatewayName string `json:"gatewayName"`

	Ingress IngressSelectorSpec `json:"ingress"`

	Egress IngressSelectorSpec `json:"egress"`

	SyncToFgw SyncToFgwSpec `json:"syncToFgw"`

	// Compute Resources required by connector container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:default=true
	// +optional
	LeaderElection *bool `json:"leaderElection,omitempty"`
}

// GatewayStatus is the type used to represent the status of a Gateway Connector resource.
type GatewayStatus struct {
	// CurrentStatus defines the current status of a Gateway Connector resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a Gateway Connector resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GatewayConnectorList contains a list of Gateway Connectors.
type GatewayConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []GatewayConnector `json:"items"`
}
