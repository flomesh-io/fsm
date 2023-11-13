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
	// +optional
	// +kubebuilder:default=LoadBalancer
	// +kubebuilder:validation:Enum=NodePort;LoadBalancer
	// ServiceType determines how the Ingress is exposed. For an Ingress
	//   the most used types are NodePort, and LoadBalancer
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// +optional
	// ServiceAnnotations, those annotations are applied to Ingress Service
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`

	// +optional
	// ServiceLabels, those annotations are applied to Ingress Service
	ServiceLabels map[string]string `json:"serviceLabels,omitempty"`

	// +optional
	// PodAnnotations, those annotations are applied to Ingress POD
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// +optional
	// PodAnnotations, those labels are applied to Ingress POD
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// +optional
	// +kubebuilder:default={enabled: true, port: {name: http, protocol: TCP, port: 80, targetPort: 8000}}
	// HTTP defines the http configuration of this ingress controller.
	HTTP *HTTP `json:"http,omitempty"`

	// +optional
	// +kubebuilder:default={enabled: false, port: {name: https, protocol: TCP, port: 443, targetPort: 8443}, sslPassthrough: {enabled: false, upstreamPort: 443}}
	// TLS is the configuration of TLS of this ingress controller
	TLS *TLS `json:"tls,omitempty"`

	// +optional
	// Env defines the list of environment variables to set in the ingress container.
	// Cannot be updated.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +optional
	// Compute Resources required by Ingress container.
	// Cannot be updated.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	// +mapType=atomic
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	// +kubebuilder:default=fsm-namespaced-ingress
	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`

	// +optional
	// If specified, the pod's scheduling constraints
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +optional
	// If specified, the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +optional
	// +kubebuilder:default=info
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic;disabled
	// LogLevel is the log level of this ingress controller pod.
	LogLevel *string `json:"logLevel,omitempty"`

	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// Replicas, how many replicas of the ingress controller will be running for this namespace.
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// +optional
	// PodSecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// HTTP defines the http configuration of this ingress controller.
type HTTP struct {
	// +optional
	// +kubebuilder:default=true
	// Enabled, if HTTP is enabled for the Ingress Controller
	Enabled *bool `json:"enabled"`

	// Port, The http port that are exposed by this ingress service.
	Port corev1.ServicePort `json:"port,omitempty"`
}

// TLS defines the TLS configuration of this ingress controller.
type TLS struct {
	// +optional
	// +kubebuilder:default=false
	// Enabled, if TLS is enabled for the Ingress Controller
	Enabled *bool `json:"enabled"`

	// +optional
	// +kubebuilder:default=false
	// MTLS defines the mTLS configuration of this ingress controller.
	MTLS *bool `json:"mTLS"`

	// Port, The https port that are exposed by this ingress service.
	Port corev1.ServicePort `json:"port,omitempty"`

	// +optional
	// +kubebuilder:default={enabled: false, upstreamPort: 443}
	// SSLPassthrough defines the SSLPassthrough configuration of this ingress controller.
	SSLPassthrough *SSLPassthrough `json:"sslPassthrough,omitempty"`
}

// SSLPassthrough defines the SSLPassthrough configuration of this ingress controller.
type SSLPassthrough struct {
	// +optional
	// +kubebuilder:default=false
	// Enabled, if SSL passthrough is enabled for the Ingress Controller
	//  It's mutual exclusive with TLS offload/termination within the controller scope.
	Enabled *bool `json:"enabled"`

	// +optional
	// +kubebuilder:default=443
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
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
