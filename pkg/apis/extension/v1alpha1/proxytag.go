package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ProxyTagSpec defines the desired state of ProxyTag
type ProxyTagSpec struct {
	// +optional
	// +kubebuilder:default="proxy-tag"
	// DestinationHostHeader is the header name to be used for the destination host, default is "proxy-tag"
	DestinationHostHeader *gwv1.HeaderName `json:"dstHostHeader,omitempty"`

	// +optional
	// +kubebuilder:default="orig-host"
	// SourceHostHeader is the header name to be used for the source host, default is "orig-host"
	SourceHostHeader *gwv1.HeaderName `json:"srcHostHeader,omitempty"`
}

// ProxyTagStatus defines the observed state of ProxyTag
type ProxyTagStatus struct {
	// Conditions describe the current conditions of the ProxyTag.
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

// ProxyTag is the Schema for the ProxyTag API
type ProxyTag struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProxyTagSpec   `json:"spec,omitempty"`
	Status ProxyTagStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProxyTagList contains a list of ProxyTag
type ProxyTagList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyTag `json:"items"`
}
