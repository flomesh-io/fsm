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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type LoadBalancerType string

const (
	ActiveActiveLbType LoadBalancerType = "ActiveActive"
	LocalityLbType     LoadBalancerType = "Locality"
	FailOverLbType     LoadBalancerType = "FailOver"
)

type TrafficTarget struct {
	// Format: [region]/[zone]/[group]/[cluster]
	ClusterKey string `json:"clusterKey"`

	// +optional
	Weight *int `json:"weight,omitempty"`
}

// GlobalTrafficPolicySpec defines the desired state of GlobalTrafficPolicy
type GlobalTrafficPolicySpec struct {
	// +kubebuilder:default=Locality
	// +kubebuilder:validation:Enum=Locality;ActiveActive;FailOver
	// Type of global load distribution
	LbType LoadBalancerType `json:"lbType"`

	// +optional
	Targets []TrafficTarget `json:"targets,omitempty"`
}

// GlobalTrafficPolicyStatus defines the observed state of GlobalTrafficPolicy
type GlobalTrafficPolicyStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gtp,scope=Namespaced
// +kubebuilder:printcolumn:name="LB Type",type="string",priority=0,JSONPath=".spec.lbType"
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io

// GlobalTrafficPolicy is the Schema for the GlobalTrafficPolicys API
type GlobalTrafficPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GlobalTrafficPolicySpec   `json:"spec,omitempty"`
	Status GlobalTrafficPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalTrafficPolicyList contains a list of GlobalTrafficPolicy
type GlobalTrafficPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalTrafficPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GlobalTrafficPolicy{}, &GlobalTrafficPolicyList{})
}
