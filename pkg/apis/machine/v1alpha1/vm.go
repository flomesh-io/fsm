package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachine is the type used to represent a VirtualMachine resource.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io
// +kubebuilder:resource:shortName=vm,scope=Namespaced
type VirtualMachine struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the VirtualMachine specification
	// +optional
	Spec VMSpec `json:"spec,omitempty"`

	// Status is the status of the VirtualMachine configuration.
	// +optional
	Status VMStatus `json:"status,omitempty"`
}

// VMSpec is the type used to represent the VirtualMachine specification.
type VMSpec struct {
	// SidecarIP is the IP address of the vm
	SidecarIP string `json:"sidecarIP"`

	// MachineIP is the IP address of the vm
	MachineIP string `json:"machineIP"`

	// IPFamily is one of IP families (e.g. IPv4, IPv6) assigned to this vm
	// +optional
	IPFamily corev1.IPFamily `json:"ipFamily,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this VM.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// The list of services that are exposed by this vm.
	Services []ServiceSpec `json:"services"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.
	// This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,
	// when it might take a long time to load data or warm a cache, than during steady-state operation.
	// This cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`
}

// ServiceSpec describes the attributes that a user creates on a service.
type ServiceSpec struct {
	// Name defines the service's name
	ServiceName string `json:"serviceName"`

	// The name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. When considering
	// the endpoints for a Service, this must match the 'name' field in the
	// EndpointPort.
	// Optional if only one ServicePort is defined on this service.
	// +optional
	PortName string `json:"portName,omitempty"`

	// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	// +default="TCP"
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// The application protocol for this port.
	// This field follows standard Kubernetes label syntax.
	// Un-prefixed names are reserved for IANA standard service names (as per
	// RFC-6335 and https://www.iana.org/assignments/service-names).
	// Non-standard protocols should use prefixed names such as
	// mycompany.com/my-custom-protocol.
	// +optional
	AppProtocol *string `json:"appProtocol,omitempty"`

	// The port that will be exposed by this service.
	Port int32 `json:"port"`
}

// VMStatus is the type used to represent the status of a VirtualMachine resource.
type VMStatus struct {
	// CurrentStatus defines the current status of a VirtualMachine resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of a VirtualMachine resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// VirtualMachineList defines the list of VirtualMachine objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachine `json:"items"`
}
