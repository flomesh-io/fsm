package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FaultInjectionSpec defines the desired state of FaultInjection
type FaultInjectionSpec struct {
	// +optional
	// Delay defines the delay configuration
	Delay *FaultInjectionDelay `json:"delay,omitempty"`

	// +optional
	// Abort defines the abort configuration
	Abort *FaultInjectionAbort `json:"abort,omitempty"`
}

// FaultInjectionDelay defines the delay configuration
type FaultInjectionDelay struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Percentage is the percentage of requests to delay
	Percentage int32 `json:"percentage"`

	// +optional
	// +kubebuilder:default="1s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// Min is the minimum delay duration, default is 1s
	Min *metav1.Duration `json:"min,omitempty"`

	// +optional
	// +kubebuilder:default="1s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// Max is the maximum delay duration, default is 1s
	Max *metav1.Duration `json:"max,omitempty"`
}

// FaultInjectionAbort defines the abort configuration
type FaultInjectionAbort struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Percentage is the percentage of requests to abort
	Percentage int32 `json:"percentage"`

	// +optional
	// +kubebuilder:default={status: 500, body: "Fault injection triggered"}
	// Response is the response when fault injection triggered
	Response *FaultInjectionResponse `json:"response,omitempty"`
}

type FaultInjectionResponse struct {
	// +optional
	// +kubebuilder:default=500
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=600
	// StatusCode is the HTTP status code of the response, default is 500
	StatusCode *int32 `json:"status,omitempty"`

	// +optional
	// Headers is the HTTP headers of response
	Headers map[string]string `json:"headers,omitempty"`

	// +optional
	// +kubebuilder:default="Fault injection triggered"
	// Body is the content of response body, default is "Fault injection triggered"
	Body *string `json:"body,omitempty"`
}

// FaultInjectionStatus defines the observed state of FaultInjection
type FaultInjectionStatus struct {
	// Conditions describe the current conditions of the FaultInjection.
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
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io}

// FaultInjection is the Schema for the FaultInjection API
type FaultInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FaultInjectionSpec   `json:"spec,omitempty"`
	Status FaultInjectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FaultInjectionList contains a list of FaultInjection
type FaultInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FaultInjection `json:"items"`
}
