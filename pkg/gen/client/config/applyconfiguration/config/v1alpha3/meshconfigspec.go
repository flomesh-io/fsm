/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha3

// MeshConfigSpecApplyConfiguration represents an declarative configuration of the MeshConfigSpec type for use
// with apply.
type MeshConfigSpecApplyConfiguration struct {
	ClusterSet    *ClusterSetSpecApplyConfiguration    `json:"clusterSet,omitempty"`
	Sidecar       *SidecarSpecApplyConfiguration       `json:"sidecar,omitempty"`
	RepoServer    *RepoServerSpecApplyConfiguration    `json:"repoServer,omitempty"`
	Traffic       *TrafficSpecApplyConfiguration       `json:"traffic,omitempty"`
	Observability *ObservabilitySpecApplyConfiguration `json:"observability,omitempty"`
	Certificate   *CertificateSpecApplyConfiguration   `json:"certificate,omitempty"`
	FeatureFlags  *FeatureFlagsApplyConfiguration      `json:"featureFlags,omitempty"`
	PluginChains  *PluginChainsSpecApplyConfiguration  `json:"pluginChains,omitempty"`
	Ingress       *IngressSpecApplyConfiguration       `json:"ingress,omitempty"`
	GatewayAPI    *GatewayAPISpecApplyConfiguration    `json:"gatewayAPI,omitempty"`
	ServiceLB     *ServiceLBSpecApplyConfiguration     `json:"serviceLB,omitempty"`
	FLB           *FLBSpecApplyConfiguration           `json:"flb,omitempty"`
	EgressGateway *EgressGatewaySpecApplyConfiguration `json:"egressGateway,omitempty"`
	Image         *ImageSpecApplyConfiguration         `json:"image,omitempty"`
	Misc          *MiscSpecApplyConfiguration          `json:"misc,omitempty"`
}

// MeshConfigSpecApplyConfiguration constructs an declarative configuration of the MeshConfigSpec type for use with
// apply.
func MeshConfigSpec() *MeshConfigSpecApplyConfiguration {
	return &MeshConfigSpecApplyConfiguration{}
}

// WithClusterSet sets the ClusterSet field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClusterSet field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithClusterSet(value *ClusterSetSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.ClusterSet = value
	return b
}

// WithSidecar sets the Sidecar field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Sidecar field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithSidecar(value *SidecarSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Sidecar = value
	return b
}

// WithRepoServer sets the RepoServer field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RepoServer field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithRepoServer(value *RepoServerSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.RepoServer = value
	return b
}

// WithTraffic sets the Traffic field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Traffic field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithTraffic(value *TrafficSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Traffic = value
	return b
}

// WithObservability sets the Observability field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Observability field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithObservability(value *ObservabilitySpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Observability = value
	return b
}

// WithCertificate sets the Certificate field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Certificate field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithCertificate(value *CertificateSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Certificate = value
	return b
}

// WithFeatureFlags sets the FeatureFlags field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FeatureFlags field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithFeatureFlags(value *FeatureFlagsApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.FeatureFlags = value
	return b
}

// WithPluginChains sets the PluginChains field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PluginChains field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithPluginChains(value *PluginChainsSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.PluginChains = value
	return b
}

// WithIngress sets the Ingress field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ingress field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithIngress(value *IngressSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Ingress = value
	return b
}

// WithGatewayAPI sets the GatewayAPI field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the GatewayAPI field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithGatewayAPI(value *GatewayAPISpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.GatewayAPI = value
	return b
}

// WithServiceLB sets the ServiceLB field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ServiceLB field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithServiceLB(value *ServiceLBSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.ServiceLB = value
	return b
}

// WithFLB sets the FLB field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FLB field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithFLB(value *FLBSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.FLB = value
	return b
}

// WithEgressGateway sets the EgressGateway field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EgressGateway field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithEgressGateway(value *EgressGatewaySpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.EgressGateway = value
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithImage(value *ImageSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Image = value
	return b
}

// WithMisc sets the Misc field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Misc field is set to the value of the last call.
func (b *MeshConfigSpecApplyConfiguration) WithMisc(value *MiscSpecApplyConfiguration) *MeshConfigSpecApplyConfiguration {
	b.Misc = value
	return b
}
