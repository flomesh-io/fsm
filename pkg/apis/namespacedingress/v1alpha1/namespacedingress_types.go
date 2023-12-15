/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespacedIngressSpec defines the desired state of NamespacedIngress
type NamespacedIngressSpec struct {
	// ServiceType determines how the Ingress is exposed. For an Ingress
	//   the most used types are NodePort, and LoadBalancer
	// +kubebuilder:default=LoadBalancer
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// ServiceAnnotations, those annotations are applied to Ingress Service
	// +optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`

	// ServiceLabels, those annotations are applied to Ingress Service
	// +optional
	ServiceLabels map[string]string `json:"serviceLabels,omitempty"`

	// PodAnnotations, those annotations are applied to Ingress POD
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// PodAnnotations, those labels are applied to Ingress POD
	// +optional
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// +kubebuilder:default={enabled: true, port: {name: http, protocol: TCP, port: 80, targetPort: 8000}}
	// +optional
	// The HTTP configuration of this ingress controller.
	HTTP HTTP `json:"http,omitempty"`

	// +kubebuilder:default={enabled: false, port: {name: https, protocol: TCP, port: 443, targetPort: 8443}, sslPassthrough: {enabled: false, upstreamPort: 443}}
	// +optional
	// TLS is the configuration of TLS of this ingress controller
	TLS TLS `json:"tls,omitempty"`

	// List of environment variables to set in the ingress container.
	// Cannot be updated.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Compute Resources required by Ingress container.
	// Cannot be updated.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:default=fsm-namespaced-ingress

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:default=info
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic;disabled
	// +optional
	// LogLevel is the log level of this ingress controller pod.
	LogLevel *string `json:"logLevel,omitempty"`

	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	// Replicas, how many replicas of the ingress controller will be running for this namespace.
	Replicas *int32 `json:"replicas,omitempty"`

	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// PodSecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// HTTP defines the http configuration of this ingress controller.
type HTTP struct {
	// +kubebuilder:default=true
	// +optional
	// Enabled, if HTTP is enabled for the Ingress Controller
	Enabled bool `json:"enabled"`

	// +kubebuilder:default={name: http, protocol: TCP, port: 80, targetPort: 8000}
	// +optional
	// Port, The http port that are exposed by this ingress service.
	Port ServicePort `json:"port,omitempty"`
}

// TLS defines the TLS configuration of this ingress controller.
type TLS struct {
	// +kubebuilder:default=false
	// +optional
	// Enabled, if TLS is enabled for the Ingress Controller
	Enabled bool `json:"enabled"`

	// +kubebuilder:default={name: https, protocol: TCP, port: 443, targetPort: 8443}
	// +optional
	// Port, The https port that are exposed by this ingress service.
	Port ServicePort `json:"port,omitempty"`

	// +kubebuilder:default=false
	// +optional
	// MTLS, if mTLS is enabled for the Ingress Controller
	MTLS bool `json:"mTLS,omitempty"`

	// +kubebuilder:default={enabled: false, upstreamPort: 443}
	// +optional
	// SSLPassthrough configuration
	SSLPassthrough SSLPassthrough `json:"sslPassthrough,omitempty"`
}

// ServicePort contains information on service's port.
type ServicePort struct {
	// The name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. When considering
	// the endpoints for a Service, this must match the 'name' field in the
	// EndpointPort.
	// Optional if only one ServicePort is defined on this service.
	// +optional
	Name string `json:"name,omitempty"`

	// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	// +default="TCP"
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// The port that will be exposed by this service.
	Port int32 `json:"port"`

	// Number or name of the port to access on the pods targeted by the service.
	// Number must be in the range 1 to 65535.
	// +optional
	TargetPort int32 `json:"targetPort,omitempty"`

	// The port on each node on which this service is exposed when type is
	// NodePort or LoadBalancer.  Usually assigned by the system. If a value is
	// specified, in-range, and not in use it will be used, otherwise the
	// operation will fail.  If not specified, a port will be allocated if this
	// Service requires one.  If this field is specified when creating a
	// Service which does not need it, creation will fail. This field will be
	// wiped when updating a Service to no longer need it (e.g. changing type
	// from NodePort to ClusterIP).
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`
}

// SSLPassthrough defines the SSLPassthrough configuration of this ingress controller.
type SSLPassthrough struct {
	// +kubebuilder:default=false
	// +optional
	// Enabled, if SSL passthrough is enabled for the Ingress Controller
	//  It's mutual exclusive with TLS offload/termination within the controller scope.
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=443
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	// UpstreamPort, is the port of upstream services.
	UpstreamPort *int32 `json:"upstreamPort"`
}

// NamespacedIngressStatus defines the observed state of NamespacedIngress
type NamespacedIngressStatus struct {
	Replicas int32  `json:"replicas"`
	Selector string `json:"selector"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:shortName=nsig,scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io

// NamespacedIngress is the Schema for the NamespacedIngresss API
type NamespacedIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespacedIngressSpec   `json:"spec,omitempty"`
	Status NamespacedIngressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedIngressList contains a list of NamespacedIngress
type NamespacedIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespacedIngress `json:"items"`
}
