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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// +kubebuilder:default=default
	// +optional

	// Region, the locality information of this cluster
	Region string `json:"region,omitempty"`

	// +kubebuilder:default=default
	// +optional

	// Zone, the locality information of this cluster
	Zone string `json:"zone,omitempty"`

	// +kubebuilder:default=default
	// +optional

	// Group, the locality information of this cluster
	Group string `json:"group,omitempty"`

	// GatewayHost, the Full Qualified Domain Name or IP of the gateway/ingress of this cluster
	// If it's an IP address, only IPv4 is supported
	GatewayHost string `json:"gatewayHost,omitempty"`

	// +kubebuilder:default=80
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional

	// The port number of the gateway
	GatewayPort int32 `json:"gatewayPort,omitempty"`

	// FIXME: temp solution, should NOT store this as plain text.
	//  consider use cli to add cluster to control plane, import kubeconfig
	//  and create a Secret with proper SA to store it as bytes

	// Kubeconfig, The kubeconfig of the cluster you want to connnect to
	// This's not needed if ClusterMode is InCluster, it will use InCluster
	// config
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// +kubebuilder:default=fsm-mesh-config
	// +optional
	// FsmMeshConfigName, defines the name of the MeshConfig of managed cluster
	FsmMeshConfigName string `json:"fsmMeshConfigName,omitempty"`

	// FsmNamespace, defines the namespace of managed cluster in which fsm is installed
	FsmNamespace string `json:"fsmNamespace"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ClusterConditionType identifies a specific condition.
type ClusterConditionType string

const (
	// ClusterManaged means that the cluster has joined the CLusterSet successfully
	//  and is managed by Control Plane.
	ClusterManaged ClusterConditionType = "Managed"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Region",type="string",priority=0,JSONPath=".spec.region"
// +kubebuilder:printcolumn:name="Zone",type="string",priority=0,JSONPath=".spec.zone"
// +kubebuilder:printcolumn:name="Group",type="string",priority=0,JSONPath=".spec.group"
// +kubebuilder:printcolumn:name="Gateway Host",type="string",priority=0,JSONPath=".spec.gatewayHost"
// +kubebuilder:printcolumn:name="Gateway Port",type="integer",priority=0,JSONPath=".spec.gatewayPort"
// +kubebuilder:printcolumn:name="Managed",type="string",priority=0,JSONPath=".status.conditions[?(@.type=='Managed')].status"
// +kubebuilder:printcolumn:name="Managed Age",type="date",priority=0,JSONPath=".status.conditions[?(@.type=='Managed')].lastTransitionTime"
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:metadata:labels=app.kubernetes.io/name=flomesh.io

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

// Key returns the key of the cluster
func (c *Cluster) Key() string {
	return utils.EvaluateTemplate(constants.ClusterIDTemplate, struct {
		Region  string
		Zone    string
		Group   string
		Cluster string
	}{
		Region:  c.Spec.Region,
		Zone:    c.Spec.Zone,
		Group:   c.Spec.Group,
		Cluster: c.Name,
	})
}
