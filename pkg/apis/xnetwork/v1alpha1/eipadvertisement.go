package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EIPAdvertisement is the type used to represent an EIPAdvertisement policy.
// An EIPAdvertisement policy authorizes one or more backends to accept
// ingress traffic from one or more sources.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=eipadvertisement,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
type EIPAdvertisement struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Ingress backend policy specification
	// +optional
	Spec EIPAdvertisementSpec `json:"spec,omitempty"`

	// +optional
	Status EIPAdvertisementStatus `json:"status,omitempty"`
}

// EIPAdvertisementStatus is the type used to represent the status.
type EIPAdvertisementStatus struct {
	// +optional
	Announce map[string]string `json:"announce,omitempty"`
}

// EIPAdvertisementSpec is the type used to represent the EIPAdvertisement policy specification.
type EIPAdvertisementSpec struct {
	// Service defines the name of the service.
	Service ElbServiceSpec `json:"service"`

	// EIPs defines the 4-layer ips for the service.
	// +kubebuilder:validation:MinItems=1
	EIPs []string `json:"eips"`

	// +optional
	Nodes []string `json:"nodes"`
}

type ElbServiceSpec struct {
	// Name defines the name of the source for the given Kind.
	Name string `json:"name"`

	// Namespace defines the namespace for the given source.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Hosts defines aliases for the given service.
	// +optional
	Hosts []string `json:"hosts,omitempty"`
}

// EIPAdvertisementList defines the list of EIPAdvertisement objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EIPAdvertisementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EIPAdvertisement `json:"items"`
}
