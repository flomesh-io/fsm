package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// HealthCheckPolicySpec defines the desired state of HealthCheckPolicy
type HealthCheckPolicySpec struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// TargetRefs is the references to the target resources to which the policy is applied
	TargetRefs []gwv1alpha2.NamespacedPolicyTargetReference `json:"targetRefs"`

	// +listType=map
	// +listMapKey=port
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// Ports is the health check configuration for ports
	Ports []PortHealthCheck `json:"ports,omitempty"`

	// +optional
	// DefaultHealthCheck is the default health check configuration for all ports
	DefaultHealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

type PortHealthCheck struct {
	// Port is the port number of the target service
	Port gwv1.PortNumber `json:"port"`

	// +optional
	// HealthCheck is the health check configuration for the port
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

type HealthCheckConfig struct {
	// +kubebuilder:default="1s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// Interval is the interval to check the health of the service
	Interval metav1.Duration `json:"interval"`

	// +kubebuilder:validation:Minimum=0
	// MaxFails is the maximum number of consecutive failed health checks before considering the service as unhealthy
	MaxFails int32 `json:"maxFails"`

	// +optional
	// +kubebuilder:default="5s"
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}(h|m|s|ms)){1,4}$`
	// FailTimeout is the time before considering the service as healthy if it's marked as unhealthy, even if it's already healthy
	FailTimeout *metav1.Duration `json:"failTimeout,omitempty"`

	// +optional
	// Path is the path to check the health of the HTTP service, if it's not set, the health check will be TCP based
	Path *string `json:"path,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=16
	// Matches is the list of health check match conditions of HTTP service
	Matches []HealthCheckMatch `json:"matches,omitempty"`
}

type HealthCheckMatch struct {
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=16
	// StatusCodes is the list of status codes to match
	StatusCodes []int32 `json:"statusCodes,omitempty"`

	// +optional
	// Body is the content of response body to match
	Body *string `json:"body,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// Headers is the list of response headers to match
	Headers []gwv1.HTTPHeader `json:"headers,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=gateway-api,shortName=hckpolicy
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels={app.kubernetes.io/name=flomesh.io,gateway.networking.k8s.io/policy=Direct}

// HealthCheckPolicy provides a way to configure how a Gateway
// checks the health state of backend service.
type HealthCheckPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of HealthCheckPolicy.
	Spec HealthCheckPolicySpec `json:"spec,omitempty"`

	// Status defines the current state of HealthCheckPolicy.
	Status gwv1alpha2.PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HealthCheckPolicyList contains a list of HealthCheckPolicy
type HealthCheckPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealthCheckPolicy `json:"items"`
}
