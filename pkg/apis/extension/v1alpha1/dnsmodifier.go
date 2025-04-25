package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// DNSModifierSpec defines the desired state of DNSModifier
type DNSModifierSpec struct {
	Zones map[string]DNSZoneConfig `json:"zones,omitempty"`
}

type DNSZoneConfig struct {
	// +optional
	// +listType=map
	// +listMapKey=name
	// Domains is the list of whitelist domains to be resolved by the DNS modifier
	Domains []DNSDomain `json:"domains,omitempty"`
}

type DNSDomain struct {
	// Name is the fully qualified domain name of a network host. This
	// matches the RFC 1123 definition of a hostname with 1 notable exception that
	// numeric IP addresses are not allowed.
	Name gwv1.PreciseHostname `json:"name"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// Answer is the DNS answer to be returned for the domain Name
	Answer []DNSAnswer `json:"answer,omitempty"`
}

type DNSAnswer struct {
	// +optional
	// +kubebuilder:validation:Enum=A;AAAA
	// +kubebuilder:default=A
	// RType is the type of the DNS record, either A or AAAA, default is A
	RType *string `json:"type"`

	// +kubebuilder:validation:MinLength=1
	// RData is the resource record data to be returned for the domain Name
	// it should be a valid IP address, either IPv4 or IPv6
	RData string `json:"rdata"`
}

// DNSModifierStatus defines the observed state of DNSModifier
type DNSModifierStatus struct {
	// Conditions describe the current conditions of the DNSModifier.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.flomesh.io/extension=Filter}

// DNSModifier is the Schema for the DNSModifier API
type DNSModifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSModifierSpec   `json:"spec,omitempty"`
	Status DNSModifierStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DNSModifierList contains a list of DNSModifier
type DNSModifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSModifier `json:"items"`
}
