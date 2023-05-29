package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AccessControl is the type used to represent an AccessControl policy.
// An AccessControl policy authorizes one or more backends to accept
// ingress traffic from one or more sources.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AccessControl struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Ingress backend policy specification
	// +optional
	Spec AccessControlSpec `json:"spec,omitempty"`

	// Status is the status of the AccessControl configuration.
	// +optional
	Status AccessControlStatus `json:"status,omitempty"`
}

// AccessControlSpec is the type used to represent the AccessControl policy specification.
type AccessControlSpec struct {
	// Backends defines the list of backends the AccessControl policy applies to.
	Backends []AccessControlBackendSpec `json:"backends"`

	// Sources defines the list of sources the AccessControl policy applies to.
	Sources []AccessControlSourceSpec `json:"sources"`

	// Matches defines the list of object references the AccessControl policy should match on.
	// +optional
	Matches []corev1.TypedLocalObjectReference `json:"matches,omitempty"`
}

// AccessControlBackendSpec is the type used to represent a Backend specified in the AccessControl policy specification.
type AccessControlBackendSpec struct {
	// Name defines the name of the backend.
	Name string `json:"name"`

	// Port defines the specification for the backend's port.
	Port PortSpec `json:"port"`

	// TLS defines the specification for the backend's TLS configuration.
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`
}

// AccessControlSourceSpec is the type used to represent the Source in the list of Sources specified in an
// AccessControl policy specification.
type AccessControlSourceSpec struct {
	// Kind defines the kind for the source in the AccessControl policy.
	// Must be one of: Service, AuthenticatedPrincipal, IPRange
	Kind string `json:"kind"`

	// Name defines the name of the source for the given Kind.
	Name string `json:"name"`

	// Namespace defines the namespace for the given source.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// AccessControlList defines the list of AccessControl objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AccessControlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AccessControl `json:"items"`
}

// AccessControlStatus is the type used to represent the status of an AccessControl resource.
type AccessControlStatus struct {
	// CurrentStatus defines the current status of an AccessControl resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an AccessControl resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
