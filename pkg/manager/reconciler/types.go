// Package v2 contains the reconciler utilities for the FSM manager
package reconciler

import "github.com/flomesh-io/fsm/pkg/logger"

var (
	log = logger.New("fsm-manager/reconciler-v2")
)

type ResourceType string

const (
	MCSCluster                            ResourceType = "MCS(Cluster)"
	MCSServiceExport                      ResourceType = "MCS(ServiceExport)"
	MCSServiceImport                      ResourceType = "MCS(ServiceImport)"
	MCSGlobalTrafficPolicy                ResourceType = "MCS(GlobalTrafficPolicy)"
	GatewayAPIGatewayClass                ResourceType = "GatewayAPI(GatewayClass)"
	GatewayAPIGateway                     ResourceType = "GatewayAPI(Gateway)"
	GatewayAPIHTTPRoute                   ResourceType = "GatewayAPI(HTTPRoute)"
	GatewayAPIGRPCRoute                   ResourceType = "GatewayAPI(GRPCRoute)"
	GatewayAPITCPRoute                    ResourceType = "GatewayAPI(TCPRoute)"
	GatewayAPITLSRoute                    ResourceType = "GatewayAPI(TLSRoute)"
	GatewayAPIUDPRoute                    ResourceType = "GatewayAPI(UDPRoute)"
	GatewayAPIReferenceGrant              ResourceType = "GatewayAPI(ReferenceGrant)"
	GatewayAPIExtensionFilter             ResourceType = "GatewayAPIExtension(Filter)"
	GatewayAPIExtensionListenerFilter     ResourceType = "GatewayAPIExtension(ListenerFilter)"
	GatewayAPIExtensionFilterDefinition   ResourceType = "GatewayAPIExtension(FilterDefinition)"
	GatewayAPIExtensionFilterConfig       ResourceType = "GatewayAPIExtension(FilterConfig)"
	GatewayAPIExtensionCircuitBreaker     ResourceType = "GatewayAPIExtension(CircuitBreaker)"
	GatewayAPIExtensionFaultInjection     ResourceType = "GatewayAPIExtension(FaultInjection)"
	GatewayAPIExtensionRateLimit          ResourceType = "GatewayAPIExtension(RateLimit)"
	GatewayAPIExtensionHTTPLog            ResourceType = "GatewayAPIExtension(HTTPLog)"
	GatewayAPIExtensionMetrics            ResourceType = "GatewayAPIExtension(Metrics)"
	GatewayAPIExtensionZipkin             ResourceType = "GatewayAPIExtension(Zipkin)"
	GatewayAPIExtensionProxyTag           ResourceType = "GatewayAPIExtension(ProxyTag)"
	GatewayAPIExtensionIPRestriction      ResourceType = "GatewayAPIExtension(IPRestriction)"
	GatewayAPIExtensionExternalRateLimit  ResourceType = "GatewayAPIExtension(ExternalRateLimit)"
	GatewayAPIExtensionRequestTermination ResourceType = "GatewayAPIExtension(RequestTermination)"
	GatewayAPIExtensionConcurrencyLimit   ResourceType = "GatewayAPIExtension(ConcurrencyLimit)"
	GatewayAPIExtensionDNSModifier        ResourceType = "GatewayAPIExtension(DNSModifier)"
	PolicyAttachmentHealthCheck           ResourceType = "PolicyAttachment(HealthCheck)"
	PolicyAttachmentBackendLB             ResourceType = "PolicyAttachment(BackendLB)"
	PolicyAttachmentBackendTLS            ResourceType = "PolicyAttachment(BackendTLS)"
	PolicyAttachmentRouteRuleFilter       ResourceType = "PolicyAttachment(RouteRuleFilter)"
	ConnectorConsulConnector              ResourceType = "Connector(ConsulConnector)"
	ConnectorEurekaConnector              ResourceType = "Connector(EurekaConnector)"
	ConnectorNacosConnector               ResourceType = "Connector(NacosConnector)"
	ConnectorZookeeperConnector           ResourceType = "Connector(ZookeeperConnector)"
	ConnectorMachineConnector             ResourceType = "Connector(MachineConnector)"
	ConnectorGatewayConnector             ResourceType = "Connector(GatewayConnector)"
	K8sIngress                            ResourceType = "K8s(Ingress)"
	NamespacedIngress                     ResourceType = "NamespacedIngress"
	ServiceLBService                      ResourceType = "ServiceLB(Service)"
	ServiceLBNode                         ResourceType = "ServiceLB(Node)"
	FLBService                            ResourceType = "FLB(Service)"
	FLBSecret                             ResourceType = "FLB(Secret)"
	FLBTLSSecret                          ResourceType = "FLB(TLSSecret)"
)
