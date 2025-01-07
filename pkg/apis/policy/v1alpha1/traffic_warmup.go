package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
)

// TrafficWarmup is the type used to represent a traffic warmup policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=trafficwarmup,scope=Namespaced
type TrafficWarmup struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the traffic raffic warmup.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec configv1alpha3.TrafficWarmupSpec `json:"spec,omitempty"`

	// Status defines the current state of TrafficWarmup.
	Status TrafficWarmupStatus `json:"status,omitempty"`
}

// TrafficWarmupStatus defines the common attributes that all filters should include within
// their status.
type TrafficWarmupStatus struct {
	// CurrentStatus defines the current status of a traffic warmup resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a raffic warmup resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TrafficWarmupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TrafficWarmup `json:"items"`
}
