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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlgoBalancer defines Balancer Algo
type AlgoBalancer string

// ServiceExport is the Schema for the ServiceExports API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceExport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceExportSpec   `json:"spec,omitempty"`
	Status ServiceExportStatus `json:"status,omitempty"`
}

// ServiceExportRule defines service export rule.
type ServiceExportRule struct {
	// The port number of service
	PortNumber int32 `json:"portNumber,omitempty"`

	// Path is matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path" part of a URL
	// as defined by RFC 3986. Paths must begin with a '/' and must be present
	// when using PathType with value "Exact" or "Prefix".
	Path string `json:"path,omitempty"`

	// PathType determines the interpretation of the Path matching. PathType can
	// be one of the following values:
	// * Exact: Matches the URL path exactly.
	// * Prefix: Matches based on a URL path prefix split by '/'. Matching is
	//   done on a path element by element basis. A path element refers is the
	//   list of labels in the path split by the '/' separator. A request is a
	//   match for path p if every p is an element-wise prefix of p of the
	//   request path. Note that if the last element of the path is a substring
	//   of the last element in request path, it is not a match (e.g. /foo/bar
	//   matches /foo/bar/baz, but does not match /foo/barbaz).
	PathType *networkingv1.PathType `json:"pathType"`
}

// PathRewrite defines path rewrite rule.
type PathRewrite struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// ServiceExportSpec defines the desired state of ServiceExport
type ServiceExportSpec struct {
	// +optional
	// PathRewrite, it shares ONE rewrite rule for the same ServiceExport
	PathRewrite *PathRewrite `json:"pathRewrite,omitempty"`

	// +optional
	// Indicates if session sticky is  enabled
	SessionSticky bool `json:"sessionSticky,omitempty"`

	// +optional
	// The LoadBalancer Type applied to the Ingress Rules those created by the ServiceExport
	LoadBalancer AlgoBalancer `json:"loadBalancer,omitempty"`

	// The paths for accessing the service via Ingress controller
	Rules []ServiceExportRule `json:"rules,omitempty"`

	// +optional
	// If empty, service is exported to all managed clusters.
	// If not empty, service is exported to specified clusters,
	//  must be in format [region]/[zone]/[group]/[cluster]
	TargetClusters []string `json:"targetClusters,omitempty"`

	// +optional
	// The ServiceAccount associated with this service
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ServiceExportStatus defines the observed state of ServiceExport
type ServiceExportStatus struct {
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ServiceExportConditionType identifies a specific condition.
type ServiceExportConditionType string

const (
	// ServiceExportValid means that the service referenced by this
	// service export has been recognized as valid by controller.
	// This will be false if the service is found to be unexportable
	// (ExternalName, not found).
	ServiceExportValid ServiceExportConditionType = "Valid"
	// ServiceExportConflict means that there is a conflict between two
	// exports for the same Service. When "True", the condition message
	// should contain enough information to diagnose the conflict:
	// field(s) under contention, which cluster won, and why.
	// Users should not expect detailed per-cluster information in the
	// conflict message.
	ServiceExportConflict ServiceExportConditionType = "Conflict"
)

// ServiceExportList contains a list of ServiceExport
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceExportList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of endpoint slices
	// +listType=set
	Items []ServiceExport `json:"items"`
}
