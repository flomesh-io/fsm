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

	// The http configuration of this ingress controller.
	// +optional
	HTTP HTTP `json:"http,omitempty"`

	// +kubebuilder:default={enabled: false, port: {name: https, protocol: TCP, port: 443, targetPort: 8443}, sslPassthrough: {enabled: false, upstreamPort: 443}}

	// TLS is the configuration of TLS of this ingress controller
	// +optional
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

	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10

	// LogLevel is the log level of this ingress controller pod.
	// +optional
	LogLevel *int `json:"logLevel,omitempty"`

	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1

	// Replicas, how many replicas of the ingress controller will be running for this namespace.
	// +optional
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

type HTTP struct {
	// +kubebuilder:default=true

	// Enabled, if HTTP is enabled for the Ingress Controller
	// +optional
	Enabled bool `json:"enabled"`

	// Port, The http port that are exposed by this ingress service.
	Port corev1.ServicePort `json:"port,omitempty"`
}

type TLS struct {
	// +kubebuilder:default=false

	// Enabled, if TLS is enabled for the Ingress Controller
	// +optional
	Enabled bool `json:"enabled"`

	// Port, The https port that are exposed by this ingress service.
	Port corev1.ServicePort `json:"port,omitempty"`

	// +kubebuilder:default={enabled: false, upstreamPort: 443}

	// SSLPassthrough configuration
	// +optional
	SSLPassthrough SSLPassthrough `json:"sslPassthrough,omitempty"`
}

type SSLPassthrough struct {
	// +kubebuilder:default=false

	// Enabled, if SSL passthrough is enabled for the Ingress Controller
	//  It's mutual exclusive with TLS offload/termination within the controller scope.
	// +optional
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=443
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535

	// UpstreamPort, is the port of upstream services.
	// +optional
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

func init() {
	SchemeBuilder.Register(&NamespacedIngress{}, &NamespacedIngressList{})
}
