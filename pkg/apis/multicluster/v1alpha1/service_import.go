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
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceImportType designates the type of a ServiceImport
type ServiceImportType string

const (
	// ClusterSetIP are only accessible via the ClusterSet IP.
	ClusterSetIP ServiceImportType = "ClusterSetIP"
	// Headless services allow backend pods to be addressed directly.
	Headless ServiceImportType = "Headless"
)

// ServiceImport is the Schema for the ServiceImports API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceImport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceImportSpec   `json:"spec,omitempty"`
	Status ServiceImportStatus `json:"status,omitempty"`
}

// ServiceImportSpec describes an imported service and the information necessary to consume it.
type ServiceImportSpec struct {
	// +listType=atomic
	Ports []ServicePort `json:"ports"`

	// ip will be used as the VIP for this service when type is ClusterSetIP.
	// +optional
	IPs []string `json:"ips,omitempty"`

	// type defines the type of this service.
	// Must be ClusterSetIP or Headless.
	// +optional
	Type ServiceImportType `json:"type"`

	// Supports "ClientIP" and "None". Used to maintain session affinity.
	// Enable client IP based session affinity.
	// Must be ClientIP or None.
	// Defaults to None.
	// Ignored when type is Headless
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	// +optional
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity,omitempty"`

	// sessionAffinityConfig contains session affinity configuration.
	// +optional
	SessionAffinityConfig *v1.SessionAffinityConfig `json:"sessionAffinityConfig,omitempty"`

	// +optional
	// The ServiceAccount associated with this service
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ServicePort represents the port on which the service is exposed
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
	// +optional
	Protocol v1.Protocol `json:"protocol,omitempty"`

	// The application protocol for this port.
	// This field follows standard Kubernetes label syntax.
	// Un-prefixed names are reserved for IANA standard service names (as per
	// RFC-6335 and http://www.iana.org/assignments/service-names).
	// Non-standard protocols should use prefixed names such as
	// mycompany.com/my-custom-protocol.
	// Field can be enabled with ServiceAppProtocol feature gate.
	// +optional
	AppProtocol *string `json:"appProtocol,omitempty"`

	// The port that will be exposed by this service.
	Port int32 `json:"port"`

	// The address of accessing the service
	Endpoints []Endpoint `json:"endpoints"`
}

func (p *ServicePort) String() string {
	if p == nil {
		return "nil"
	}
	s := strings.Join([]string{`&ServicePort{`,
		`Name:` + fmt.Sprintf("%v", p.Name) + `,`,
		`Protocol:` + fmt.Sprintf("%v", p.Protocol) + `,`,
		`Port:` + fmt.Sprintf("%v", p.Port) + `,`,
		`}`,
	}, "")
	return s
}

// Endpoint imported service's endpoints
type Endpoint struct {
	Target     Target `json:"target"`
	ClusterKey string `json:"clusterKey"`
}

// Target imported service's endpoint target.
type Target struct {
	Host string `json:"host"`
	IP   string `json:"ip"`
	Port int32  `json:"port"`
	Path string `json:"path"`
}

// ServiceImportStatus describes derived state of an imported service.
type ServiceImportStatus struct {
	// clusters is the list of exporting clusters from which this service
	// was derived.
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=cluster
	// +listType=map
	// +listMapKey=cluster
	Clusters []ClusterStatus `json:"clusters,omitempty"`
}

// ClusterStatus contains service configuration mapped to a specific source cluster
type ClusterStatus struct {
	// cluster is the name of the exporting cluster. Must be a valid RFC-1123 DNS
	// label.
	Cluster string `json:"cluster"`

	// in-cluster service, it's the cluster IPs
	// otherwise, it's the url of accessing that service in remote cluster
	// for example, http(s)://[Ingress IP/domain name]:[port]/[path]
	Addresses []string `json:"addresses,omitempty"`
}

// ServiceImportList contains a list of ServiceImport
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceImportList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of endpoint slices
	// +listType=set
	Items []ServiceImport `json:"items"`
}
