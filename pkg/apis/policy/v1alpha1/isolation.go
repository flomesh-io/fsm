package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Isolation is the type used to represent an isolation policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=iso,scope=Cluster
type Isolation struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Isolation specification
	// +optional
	Spec IsolationSpec `json:"spec,omitempty"`

	// Status is the status of the Isolation configuration.
	// +optional
	Status IsolationStatus `json:"status,omitempty"`
}

// IsolationSpec is the type used to represent the IsolationSpec policy specification.
type IsolationSpec struct {
	// cidr is a string representing the IP Isolation
	// Valid examples are "192.168.1.0/24"
	// +kubebuilder:validation:MinItems=1
	CIDR []string `json:"cidrs"`
}

// IsolationList defines the list of Isolation objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IsolationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Isolation `json:"items"`
}

// IsolationStatus is the type used to represent the status of an Isolation resource.
type IsolationStatus struct {
	// CurrentStatus defines the current status of an Isolation resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an Isolation resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
