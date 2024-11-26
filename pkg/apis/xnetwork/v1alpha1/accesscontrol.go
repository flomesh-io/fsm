package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AccessControl is the type used to represent an AccessControl policy.
// An AccessControl policy authorizes one or more backends to accept
// ingress traffic from one or more sources.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=accesscontrol,scope=Namespaced
type AccessControl struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Ingress backend policy specification
	// +optional
	Spec AccessControlSpec `json:"spec,omitempty"`
}

// AccessControlSpec is the type used to represent the AccessControl policy specification.
type AccessControlSpec struct {
	// Services defines the list of sources the AccessControl policy applies to.
	Services []AccessControlServiceSpec `json:"services"`
}

// AccessControlServiceSpec is the type used to represent the Source in the list of Sources specified in an
// AccessControl policy specification.
type AccessControlServiceSpec struct {
	// Name defines the name of the source for the given Kind.
	Name string `json:"name"`

	// Namespace defines the namespace for the given source.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +kubebuilder:default=true
	// +optional
	WithClusterIPs bool `json:"withClusterIPs,omitempty"`

	// +kubebuilder:default=false
	// +optional
	WithExternalIPs bool `json:"withExternalIPs,omitempty"`

	// +kubebuilder:default=false
	// +optional
	WithEndpointIPs bool `json:"withEndpointIPs,omitempty"`
}

// AccessControlList defines the list of AccessControl objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AccessControlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AccessControl `json:"items"`
}
