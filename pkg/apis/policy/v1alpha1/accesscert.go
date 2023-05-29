package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AccessCert is the type used to represent an AccessCert policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AccessCert struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Access Cert specification
	// +optional
	Spec AccessCertSpec `json:"spec,omitempty"`

	// Status is the status of the AccessCert configuration.
	// +optional
	Status AccessCertStatus `json:"status,omitempty"`
}

// AccessCertSpec is the type used to represent the AccessCert policy specification.
type AccessCertSpec struct {
	// SubjectAltNames defines the Subject Alternative Names (domain names and IP addresses) secured by the certificate.
	SubjectAltNames []string `json:"subjectAltNames"`

	// Secret defines the secret in which the certificate is stored.
	Secret corev1.SecretReference `json:"secret"`
}

// AccessCertList defines the list of AccessCert objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AccessCertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AccessCert `json:"items"`
}

// AccessCertStatus is the type used to represent the status of an AccessCert resource.
type AccessCertStatus struct {
	// CurrentStatus defines the current status of an AccessCert resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an AccessCert resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
