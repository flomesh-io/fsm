package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HTTPLogSpec defines the desired state of HTTPLog
type HTTPLogSpec struct {
	// +kubebuilder:validation:Required
	// Target is the URL of the HTTPLog service
	Target string `json:"target"`

	// +optional
	// +kubebuilder:default="POST"
	// +kubebuilder:validation:Enum=GET;POST;PUT;DELETE;PATCH;HEAD;OPTIONS
	// Method is the HTTP method of the HTTPLog service, default is POST
	Method *string `json:"method,omitempty"`

	// +optional
	// Headers is the HTTP headers of the log request
	Headers map[string]string `json:"headers,omitempty"`

	// +optional
	// +kubebuilder:default=1048576
	// +kubebuilder:validation:Minimum=1
	// BufferLimit is the maximum size of the buffer in bytes, default is 1048576(1MB)
	BufferLimit *int64 `json:"bufferLimit,omitempty"`

	// +optional
	// +kubebuilder:default={size: 1000, interval: "1s", prefix: "", postfix: "", separator: "\n"}
	// Batch is the batch configuration of the logs
	Batch *HTTPLogBatch `json:"batch,omitempty"`
}

type HTTPLogBatch struct {
	// +optional
	// +kubebuilder:default=1000
	// +kubebuilder:validation:Minimum=1
	// Size is the maximum number of logs in a batch, default is 1000
	Size *int32 `json:"size,omitempty"`

	// +optional
	// +kubebuilder:default="1s"
	// Interval is the interval to send a batch, default is 1s
	Interval *metav1.Duration `json:"interval,omitempty"`

	// +optional
	// +kubebuilder:default=""
	// Prefix is the prefix of the batch, default is ""
	Prefix *string `json:"prefix,omitempty"`

	// +optional
	// +kubebuilder:default=""
	// Postfix is the postfix of the batch, default is ""
	Postfix *string `json:"postfix,omitempty"`

	// +optional
	// +kubebuilder:default="\n"
	// Separator is the separator of the logs in the batch, default is "\n"
	Separator *string `json:"separator,omitempty"`
}

// HTTPLogStatus defines the observed state of HTTPLog
type HTTPLogStatus struct {
	// Conditions describe the current conditions of the HTTPLog.
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

// HTTPLog is the Schema for the HTTPLog API
type HTTPLog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPLogSpec   `json:"spec,omitempty"`
	Status HTTPLogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HTTPLogList contains a list of HTTPLog
type HTTPLogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPLog `json:"items"`
}
