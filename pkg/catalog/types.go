// Package catalog implements the MeshCataloger interface, which forms the central component in FSM that transforms
// outputs from all other components (SMI policies, Kubernetes services, endpoints etc.) into configuration that is
// consumed by the the proxy control plane component to program sidecar proxies.
// Reference: https://github.com/flomesh-io/fsm/blob/main/DESIGN.md#5-mesh-catalog
package catalog

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/multicluster"
	"github.com/flomesh-io/fsm/pkg/plugin"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/smi"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
	"github.com/flomesh-io/fsm/pkg/utils/cidr"
)

var (
	log = logger.New("mesh-catalog")
)

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	endpointsProviders []endpoint.Provider
	serviceProviders   []service.Provider
	meshSpec           smi.MeshSpec
	configurator       configurator.Configurator
	certManager        *certificate.Manager

	// This is the kubernetes client that operates async caches to avoid issuing synchronous
	// calls through kubeClient and instead relies on background cache synchronization and local
	// lookups
	kubeController k8s.Controller

	// policyController implements the functionality related to the resources part of the policy.flomesh.io
	// API group, such as egress.
	policyController policy.Controller

	// pluginController implements the functionality related to the resources part of the plugin.flomesh.io
	// API group, such as plugin, pluginChain, pluginConfig.
	pluginController plugin.Controller

	// multiclusterController implements the functionality related to the resources part of the flomesh.io
	// API group, such a serviceimport.
	multiclusterController multicluster.Controller
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all sidecar proxies connected to the catalog.
type MeshCataloger interface {
	// ListOutboundServicesForIdentity list the services the given service identity is allowed to initiate outbound connections to
	ListOutboundServicesForIdentity(identity.ServiceIdentity) []service.MeshService

	// ListInboundServiceIdentities lists the downstream service identities that are allowed to connect to the given service identity
	ListInboundServiceIdentities(identity.ServiceIdentity) []identity.ServiceIdentity

	// ListOutboundServiceIdentities lists the upstream service identities the given service identity are allowed to connect to
	ListOutboundServiceIdentities(identity.ServiceIdentity) []identity.ServiceIdentity

	// ListServiceIdentitiesForService lists the service identities associated with the given service
	ListServiceIdentitiesForService(service.MeshService) []identity.ServiceIdentity

	// ListAllowedUpstreamEndpointsForService returns the list of endpoints over which the downstream client identity
	// is allowed access the upstream service
	ListAllowedUpstreamEndpointsForService(identity.ServiceIdentity, service.MeshService) []endpoint.Endpoint

	// GetIngressTrafficPolicy returns the ingress traffic policy for the given mesh service
	GetIngressTrafficPolicy(service.MeshService) (*trafficpolicy.IngressTrafficPolicy, error)

	// GetAccessControlTrafficPolicy returns the access control traffic policy for the given mesh service
	GetAccessControlTrafficPolicy(service.MeshService) (*trafficpolicy.AccessControlTrafficPolicy, error)

	// ListInboundTrafficTargetsWithRoutes returns a list traffic target objects composed of its routes for the given destination service identity
	ListInboundTrafficTargetsWithRoutes(identity.ServiceIdentity) ([]trafficpolicy.TrafficTargetWithRoutes, error)

	// GetEgressGatewayPolicy returns the Egress gateway policy.
	GetEgressGatewayPolicy() (*trafficpolicy.EgressGatewayPolicy, error)

	// GetEgressTrafficPolicy returns the Egress traffic policy associated with the given service identity.
	GetEgressTrafficPolicy(identity.ServiceIdentity) (*trafficpolicy.EgressTrafficPolicy, error)

	// GetEgressSourceSecret returns the secret resource that matches the given options
	GetEgressSourceSecret(corev1.SecretReference) (*corev1.Secret, error)

	// GetKubeController returns the kube controller instance handling the current cluster
	GetKubeController() k8s.Controller

	// GetOutboundMeshTrafficPolicy returns the outbound mesh traffic policy for the given downstream identity
	GetOutboundMeshTrafficPolicy(identity.ServiceIdentity) *trafficpolicy.OutboundMeshTrafficPolicy

	// GetInboundMeshTrafficPolicy returns the inbound mesh traffic policy for the given upstream identity and services
	GetInboundMeshTrafficPolicy(identity.ServiceIdentity, []service.MeshService) *trafficpolicy.InboundMeshTrafficPolicy

	// GetRetryPolicy returns the RetryPolicySpec for the given downstream identity and upstream service
	GetRetryPolicy(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) *v1alpha1.RetryPolicySpec

	// GetExportTrafficPolicy returns the export policy for the given mesh service
	GetExportTrafficPolicy(svc service.MeshService) (*trafficpolicy.ServiceExportTrafficPolicy, error)

	GetIsolationCidrs() []*cidr.CIDR

	// GetPlugins returns the plugin policies
	GetPlugins() []*trafficpolicy.Plugin

	// GetPluginConfigs lists plugin configs
	GetPluginConfigs() []*trafficpolicy.PluginConfig

	// GetPluginChains lists plugin chains
	GetPluginChains() []*trafficpolicy.PluginChain
}

type trafficDirection string

const (
	inbound  trafficDirection = "inbound"
	outbound trafficDirection = "outbound"
)
