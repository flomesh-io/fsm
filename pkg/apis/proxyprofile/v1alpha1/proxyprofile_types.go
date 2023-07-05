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
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ProxyProfileSpec defines the desired state of ProxyProfile
type ProxyProfileSpec struct {
	// +kubebuilder:default=false
	// +optional

	// If this ProxyProfile is Disabled. A disabled ProxyProfile doesn't participate
	// the sidecar injection process.
	Disabled bool `json:"disabled,omitempty"`

	// +kubebuilder:default=Remote
	// +optional

	// ConfigMode tells where the sidecar loads the scripts/config from, Local means from local files mounted by configmap,
	//  Remote means loads from remote pipy repo. Default value is Remote
	ConfigMode ProxyConfigMode `json:"mode,omitempty"`

	// +optional

	// Selector is a label query over pods that should be injected
	// This's optional, please NOTE a nil or empty Selector match
	// nothing not everything.
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// +optional

	// Namespace ProxyProfile will only match the pods in the namespace
	// otherwise, match pods in all namespaces(in cluster)
	Namespace string `json:"namespace,omitempty"`

	// +optional

	// Config contains the configuration data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// Values with non-UTF-8 byte sequences must use the BinaryData field.
	// This option is mutually exclusive with RepoBaseUrl option, you can only
	// have either one.
	Config map[string]string `json:"config,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge

	// List of environment variables to set in each of the service containers.
	// Cannot be updated.
	ServiceEnv []corev1.EnvVar `json:"serviceEnv,omitempty"`

	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;Always
	// +optional

	// RestartPolicy indicates if ProxyProfile is updated, those already injected PODs
	// should be updated or not. Default value is Never, it only has impact to new created
	// PODs, existing PODs will not be updated.
	RestartPolicy ProxyRestartPolicy `json:"restartPolicy,omitempty"`

	// +kubebuilder:default=Owner
	// +kubebuilder:validation:Enum=Owner
	// +optional

	// RestartScope takes effect when RestartPolicy is Always, it tells if we can restart
	// the entire POD to apply the changes or only the sidecar containers inside the POD.
	// Default value is Owner.
	RestartScope ProxyRestartScope `json:"restartScope,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=5
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name

	// List of sidecars, will be injected into POD. It must have at least ONE sidecar and
	// up to 5 maximum.
	Sidecars []Sidecar `json:"sidecars,omitempty"`
}

type Sidecar struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string `json:"name"`

	// +optional

	// The file name of entrypoint script for starting the PIPY instance.
	// If not provided, the default value is the value of Name field with surfix .js.
	// For example, if the Name of the sidecar is proxy, it looks up proxy.js in config folder.
	// It only works in local config mode, if pulls scripts from remote repo, the repo server
	// returns the name of startup script.
	StartupScriptName string `json:"startupScriptName,omitempty"`

	// +optional

	// Docker image name.
	// This field is optional to allow higher level config management to default or override
	// container images in workload controllers like Deployments and StatefulSets.
	Image string `json:"image,omitempty"`

	// +optional

	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge

	// List of environment variables to set in the sidecar container.
	// Cannot be updated.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +optional

	// Entrypoint array. Not executed within a shell.
	// The docker image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	Command []string `json:"command,omitempty"`

	// +optional

	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	Args []string `json:"args,omitempty"`

	// +optional

	// Compute Resources required by this container.
	// Cannot be updated.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// ProxyProfileStatus defines the observed state of ProxyProfile
type ProxyProfileStatus struct {
	// All associated config maps, key is namespace and value is the name of configmap
	ConfigMaps map[string]string `json:"configMaps"`
}

type ProxyRestartPolicy string

const (
	ProxyRestartPolicyNever  ProxyRestartPolicy = "Never"
	ProxyRestartPolicyAlways ProxyRestartPolicy = "Always"
)

type ProxyRestartScope string

const (
	//ProxyRestartScopePod     ProxyRestartScope = "Pod"
	//ProxyRestartScopeSidecar ProxyRestartScope = "Sidecar"
	ProxyRestartScopeOwner ProxyRestartScope = "Owner"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=pf,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type="string",priority=0,JSONPath=".spec.namespace"
// +kubebuilder:printcolumn:name="Disabled",type="boolean",priority=0,JSONPath=".spec.disabled"
// +kubebuilder:printcolumn:name="Mode",type="string",priority=0,JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="Selector",type="string",priority=1,JSONPath=".spec.selector"
// +kubebuilder:printcolumn:name="Age",type="date",priority=0,JSONPath=".metadata.creationTimestamp"

// ProxyProfile is the Schema for the proxyprofiles API
type ProxyProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProxyProfileSpec   `json:"spec,omitempty"`
	Status ProxyProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProxyProfileList contains a list of ProxyProfile
type ProxyProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProxyProfile{}, &ProxyProfileList{})
}

func (pf *ProxyProfile) ConfigHash() string {
	return util.SimpleHash(pf.Spec.Config)
}

func (pf *ProxyProfile) SpecHash() string {
	return util.SimpleHash(pf.Spec)
}

func (pf *ProxyProfile) ConstructLabelSelector() labels.Selector {
	return labels.SelectorFromSet(pf.ConstructLabels())
}

func (pf *ProxyProfile) ConstructLabels() map[string]string {
	return map[string]string{
		commons.ProxyProfileLabel: pf.Name,
		commons.CRDTypeLabel:      pf.Kind,
		commons.CRDVersionLabel:   pf.GroupVersionKind().Version,
	}
}

func (pf *ProxyProfile) GenerateConfigMapName(namespace string) string {
	return fmt.Sprintf("%s-%s-%s",
		pf.Name+commons.ConfigMapNameSuffix,
		util.HashFNV(fmt.Sprintf("%s/%s", namespace, pf.Name)),
		util.GenerateRandom(4),
	)
}

func (pf *ProxyProfile) GetConfigMode() ProxyConfigMode {
	return pf.Spec.ConfigMode
}

func (pf *ProxyProfile) isRemoteMode() bool {
	return pf.Spec.ConfigMode == ProxyConfigModeRemote
}

func (pf *ProxyProfile) isLocalMode() bool {
	return pf.Spec.ConfigMode == ProxyConfigModeLocal
}
