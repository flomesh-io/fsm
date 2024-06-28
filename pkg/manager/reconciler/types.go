// Package v2 contains the reconciler utilities for the FSM manager
package reconciler

import "github.com/flomesh-io/fsm/pkg/logger"

var (
	log = logger.New("fsm-manager/reconciler-v2")
)

type ResourceType string

const (
	MCSCluster                  ResourceType = "MCS(Cluster)"
	MCSServiceExport            ResourceType = "MCS(ServiceExport)"
	MCSServiceImport            ResourceType = "MCS(ServiceImport)"
	MCSGlobalTrafficPolicy      ResourceType = "MCS(GlobalTrafficPolicy)"
	GatewayAPIGatewayClass      ResourceType = "GatewayAPI(GatewayClass)"
	GatewayAPIGateway           ResourceType = "GatewayAPI(Gateway)"
	GatewayAPIHTTPRoute         ResourceType = "GatewayAPI(HTTPRoute)"
	GatewayAPIGRPCRoute         ResourceType = "GatewayAPI(GRPCRoute)"
	GatewayAPITCPRoute          ResourceType = "GatewayAPI(TCPRoute)"
	GatewayAPITLSRoute          ResourceType = "GatewayAPI(TLSRoute)"
	GatewayAPIUDPRoute          ResourceType = "GatewayAPI(UDPRoute)"
	PolicyAttachmentHealthCheck ResourceType = "PolicyAttachment(HealthCheck)"
	PolicyAttachmentRetry       ResourceType = "PolicyAttachment(Retry)"
	PolicyAttachmentBackendLB   ResourceType = "PolicyAttachment(BackendLB)"
	PolicyAttachmentBackendTLS  ResourceType = "PolicyAttachment(BackendTLS)"
	ConnectorConsulConnector    ResourceType = "Connector(ConsulConnector)"
	ConnectorEurekaConnector    ResourceType = "Connector(EurekaConnector)"
	ConnectorNacosConnector     ResourceType = "Connector(NacosConnector)"
	ConnectorMachineConnector   ResourceType = "Connector(MachineConnector)"
	ConnectorGatewayConnector   ResourceType = "Connector(GatewayConnector)"
	K8sIngress                  ResourceType = "K8s(Ingress)"
	NamespacedIngress           ResourceType = "NamespacedIngress"
	ServiceLBService            ResourceType = "ServiceLB(Service)"
	ServiceLBNode               ResourceType = "ServiceLB(Node)"
	FLBService                  ResourceType = "FLB(Service)"
	FLBSecret                   ResourceType = "FLB(Secret)"
	FLBTLSSecret                ResourceType = "FLB(TLSSecret)"
)
