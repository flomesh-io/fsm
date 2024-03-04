package cli

import (
	"time"

	connectorv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
)

func (c *client) initGatewayConnectorConfig(spec connectorv1alpha1.GatewaySpec) {
	//-sync-k8s-to-fgw
	Cfg.SyncK8sToGateway = spec.SyncToFgw.Enable
	//-sync-k8s-to-fgw-default-sync
	Cfg.K2G.FlagDefaultSync = spec.SyncToFgw.DefaultSync
	Cfg.K2G.FlagSyncPeriod = 5 * time.Second
	//-sync-k8s-to-fgw-allow-k8s-namespaces
	Cfg.K2G.FlagAllowK8SNamespaces = spec.SyncToFgw.AllowK8sNamespaces
	//-sync-k8s-to-fgw-deny-k8s-namespaces
	Cfg.K2G.FlagDenyK8SNamespaces = spec.SyncToFgw.DenyK8sNamespaces

	//-via-gateway-ingress-ip-selector
	Cfg.Via.IngressIPSelector = string(spec.Ingress.IPSelector)
	//-via-gateway-egress-ip-selector
	Cfg.Via.EgressIPSelector = string(spec.Egress.IPSelector)
	//-via-gateway-ingress-http-port
	Cfg.Via.Ingress.HTTPPort = uint(spec.Ingress.HTTPPort)
	//-via-gateway-egress-http-port
	Cfg.Via.Ingress.GRPCPort = uint(spec.Ingress.GRPCPort)
	//-via-gateway-ingress-grpc-port
	Cfg.Via.Egress.HTTPPort = uint(spec.Egress.HTTPPort)
	//-via-gateway-egress-grpc-port
	Cfg.Via.Egress.GRPCPort = uint(spec.Egress.GRPCPort)
}

func (c *client) initMachineConnectorConfig(spec connectorv1alpha1.MachineSpec) {
	//-derive-namespace
	Cfg.DeriveNamespace = spec.DeriveNamespace
	//-as-internal-services
	Cfg.AsInternalServices = spec.AsInternalServices

	//-sync-cloud-to-k8s
	Cfg.SyncCloudToK8s = spec.SyncToK8S.Enable
	//-sync-cloud-to-k8s-cluster-id
	Cfg.C2K.FlagClusterId = spec.SyncToK8S.ClusterId
	//-sync-cloud-to-k8s-passing-only
	Cfg.C2K.FlagPassingOnly = spec.SyncToK8S.PassingOnly
	//-sync-cloud-to-k8s-filter-tag
	Cfg.C2K.FlagFilterTag = spec.SyncToK8S.FilterLabel
	//-sync-cloud-to-k8s-prefix-tag
	Cfg.C2K.FlagPrefixTag = spec.SyncToK8S.PrefixLabel
	//-sync-cloud-to-k8s-suffix-tag
	Cfg.C2K.FlagSuffixTag = spec.SyncToK8S.SuffixLabel
	//-sync-cloud-to-k8s-with-gateway
	Cfg.C2K.FlagWithGateway.Enable = spec.SyncToK8S.WithGateway
}

func (c *client) initNacosConnectorConfig(spec connectorv1alpha1.NacosSpec) {
	//-derive-namespace
	Cfg.DeriveNamespace = spec.DeriveNamespace
	//-as-internal-services
	Cfg.AsInternalServices = spec.AsInternalServices
	//-sdr-http-addr
	Cfg.HttpAddr = spec.HTTPAddr

	//-nacos-username
	Cfg.Nacos.FlagUsername = spec.Username
	//-nacos-password
	Cfg.Nacos.FlagPassword = spec.Password
	//-nacos-namespace-id
	Cfg.Nacos.FlagNamespaceId = spec.NamespaceId

	//-sync-cloud-to-k8s
	Cfg.SyncCloudToK8s = spec.SyncToK8S.Enable
	//-sync-cloud-to-k8s-cluster-id
	Cfg.C2K.FlagClusterId = spec.SyncToK8S.ClusterId
	//-sync-cloud-to-k8s-passing-only
	Cfg.C2K.FlagPassingOnly = spec.SyncToK8S.PassingOnly
	//-sync-cloud-to-k8s-filter-tag
	Cfg.C2K.FlagFilterTag = spec.SyncToK8S.FilterMetadata
	//-sync-cloud-to-k8s-prefix-tag
	Cfg.C2K.FlagPrefixTag = spec.SyncToK8S.PrefixMetadata
	//-sync-cloud-to-k8s-suffix-tag
	Cfg.C2K.FlagSuffixTag = spec.SyncToK8S.SuffixMetadata
	//-sync-cloud-to-k8s-with-gateway
	Cfg.C2K.FlagWithGateway.Enable = spec.SyncToK8S.WithGateway

	//-sync-cloud-to-k8s-nacos-cluster-set
	Cfg.C2K.Nacos.FlagClusterSet = spec.SyncToK8S.ClusterSet
	//-sync-cloud-to-k8s-nacos-group-set
	Cfg.C2K.Nacos.FlagGroupSet = spec.SyncToK8S.GroupSet

	//-sync-k8s-to-cloud
	Cfg.SyncK8sToCloud = spec.SyncFromK8S.Enable
	//-sync-k8s-to-cloud-cluster-id
	Cfg.K2C.Nacos.FlagClusterId = spec.SyncFromK8S.ClusterId
	//-sync-k8s-to-cloud-group-id
	Cfg.K2C.Nacos.FlagGroupId = spec.SyncFromK8S.GroupId
	//-sync-k8s-to-cloud-default-sync
	Cfg.K2C.FlagDefaultSync = spec.SyncFromK8S.DefaultSync
	Cfg.K2C.FlagSyncPeriod = 5 * time.Second
	//-sync-k8s-to-cloud-sync-cluster-ip-services
	Cfg.K2C.FlagSyncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	//-sync-k8s-to-cloud-sync-load-balancer-services-endpoints
	Cfg.K2C.FlagSyncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	//-sync-k8s-to-cloud-node-port-sync-type
	Cfg.K2C.FlagNodePortSyncType = string(spec.SyncFromK8S.NodePortSyncType)
	//-sync-k8s-to-cloud-sync-ingress
	Cfg.K2C.FlagSyncIngress = spec.SyncFromK8S.SyncIngress
	//sync-k8s-to-cloud-sync-ingress-load-balancer-ips
	Cfg.K2C.FlagSyncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	//-sync-k8s-to-cloud-add-service-prefix
	Cfg.K2C.FlagAddServicePrefix = spec.SyncFromK8S.AddServicePrefix
	//-sync-k8s-to-cloud-add-k8s-namespace-as-service-suffix
	Cfg.K2C.FlagAddK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix

	Cfg.K2C.FlagAppendMetadataKeys = nil
	Cfg.K2C.FlagAppendMetadataValues = nil
	if len(spec.SyncFromK8S.AppendMetadatas) > 0 {
		for _, metadata := range spec.SyncFromK8S.AppendMetadatas {
			//-sync-k8s-to-cloud-append-metadata-key
			Cfg.K2C.FlagAppendMetadataKeys = append(Cfg.K2C.FlagAppendMetadataKeys, metadata.Key)
			//-sync-k8s-to-cloud-append-metadata-value
			Cfg.K2C.FlagAppendMetadataValues = append(Cfg.K2C.FlagAppendMetadataValues, metadata.Value)
		}
	}

	//-sync-k8s-to-cloud-allow-k8s-namespaces
	Cfg.K2C.FlagAllowK8SNamespaces = spec.SyncFromK8S.AllowK8sNamespaces
	//-sync-k8s-to-cloud-deny-k8s-namespaces
	Cfg.K2C.FlagDenyK8SNamespaces = spec.SyncFromK8S.DenyK8sNamespaces

	//-sync-k8s-to-cloud-with-gateway
	Cfg.K2C.FlagWithGateway.Enable = spec.SyncFromK8S.WithGateway
}

func (c *client) initEurekaConnectorConfig(spec connectorv1alpha1.EurekaSpec) {
	//-derive-namespace
	Cfg.DeriveNamespace = spec.DeriveNamespace
	//-as-internal-services
	Cfg.AsInternalServices = spec.AsInternalServices
	//-sdr-http-addr
	Cfg.HttpAddr = spec.HTTPAddr

	//-sync-cloud-to-k8s
	Cfg.SyncCloudToK8s = spec.SyncToK8S.Enable
	//-sync-cloud-to-k8s-cluster-id
	Cfg.C2K.FlagClusterId = spec.SyncToK8S.ClusterId
	//-sync-cloud-to-k8s-passing-only
	Cfg.C2K.FlagPassingOnly = spec.SyncToK8S.PassingOnly
	//-sync-cloud-to-k8s-filter-tag
	Cfg.C2K.FlagFilterTag = spec.SyncToK8S.FilterMetadata
	//-sync-cloud-to-k8s-prefix-tag
	Cfg.C2K.FlagPrefixTag = spec.SyncToK8S.PrefixMetadata
	//-sync-cloud-to-k8s-suffix-tag
	Cfg.C2K.FlagSuffixTag = spec.SyncToK8S.SuffixMetadata
	//-sync-cloud-to-k8s-with-gateway
	Cfg.C2K.FlagWithGateway.Enable = spec.SyncToK8S.WithGateway

	//-sync-k8s-to-cloud
	Cfg.SyncK8sToCloud = spec.SyncFromK8S.Enable
	//-sync-k8s-to-cloud-default-sync
	Cfg.K2C.FlagDefaultSync = spec.SyncFromK8S.DefaultSync
	Cfg.K2C.FlagSyncPeriod = 5 * time.Second
	//-sync-k8s-to-cloud-sync-cluster-ip-services
	Cfg.K2C.FlagSyncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	//-sync-k8s-to-cloud-sync-load-balancer-services-endpoints
	Cfg.K2C.FlagSyncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	//-sync-k8s-to-cloud-node-port-sync-type
	Cfg.K2C.FlagNodePortSyncType = string(spec.SyncFromK8S.NodePortSyncType)
	//-sync-k8s-to-cloud-sync-ingress
	Cfg.K2C.FlagSyncIngress = spec.SyncFromK8S.SyncIngress
	//sync-k8s-to-cloud-sync-ingress-load-balancer-ips
	Cfg.K2C.FlagSyncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	//-sync-k8s-to-cloud-add-service-prefix
	Cfg.K2C.FlagAddServicePrefix = spec.SyncFromK8S.AddServicePrefix
	//-sync-k8s-to-cloud-add-k8s-namespace-as-service-suffix
	Cfg.K2C.FlagAddK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix

	Cfg.K2C.FlagAppendMetadataKeys = nil
	Cfg.K2C.FlagAppendMetadataValues = nil
	if len(spec.SyncFromK8S.AppendMetadatas) > 0 {
		for _, metadata := range spec.SyncFromK8S.AppendMetadatas {
			//-sync-k8s-to-cloud-append-metadata-key
			Cfg.K2C.FlagAppendMetadataKeys = append(Cfg.K2C.FlagAppendMetadataKeys, metadata.Key)
			//-sync-k8s-to-cloud-append-metadata-value
			Cfg.K2C.FlagAppendMetadataValues = append(Cfg.K2C.FlagAppendMetadataValues, metadata.Value)
		}
	}

	//-sync-k8s-to-cloud-allow-k8s-namespaces
	Cfg.K2C.FlagAllowK8SNamespaces = spec.SyncFromK8S.AllowK8sNamespaces
	//-sync-k8s-to-cloud-deny-k8s-namespaces
	Cfg.K2C.FlagDenyK8SNamespaces = spec.SyncFromK8S.DenyK8sNamespaces

	//-sync-k8s-to-cloud-with-gateway
	Cfg.K2C.FlagWithGateway.Enable = spec.SyncFromK8S.WithGateway
}

func (c *client) initConsulConnectorConfig(spec connectorv1alpha1.ConsulSpec) {
	//-derive-namespace
	Cfg.DeriveNamespace = spec.DeriveNamespace
	//-as-internal-services
	Cfg.AsInternalServices = spec.AsInternalServices
	//-sdr-http-addr
	Cfg.HttpAddr = spec.HTTPAddr

	//-sync-cloud-to-k8s
	Cfg.SyncCloudToK8s = spec.SyncToK8S.Enable
	//-sync-cloud-to-k8s-cluster-id
	Cfg.C2K.FlagClusterId = spec.SyncToK8S.ClusterId
	//-sync-cloud-to-k8s-passing-only
	Cfg.C2K.FlagPassingOnly = spec.SyncToK8S.PassingOnly
	//-sync-cloud-to-k8s-filter-tag
	Cfg.C2K.FlagFilterTag = spec.SyncToK8S.FilterTag
	//-sync-cloud-to-k8s-prefix-tag
	Cfg.C2K.FlagPrefixTag = spec.SyncToK8S.PrefixTag
	//-sync-cloud-to-k8s-suffix-tag
	Cfg.C2K.FlagSuffixTag = spec.SyncToK8S.SuffixTag
	//-sync-cloud-to-k8s-with-gateway
	Cfg.C2K.FlagWithGateway.Enable = spec.SyncToK8S.WithGateway

	//-sync-k8s-to-cloud
	Cfg.SyncK8sToCloud = spec.SyncFromK8S.Enable
	//-sync-k8s-to-cloud-default-sync
	Cfg.K2C.FlagDefaultSync = spec.SyncFromK8S.DefaultSync
	Cfg.K2C.FlagSyncPeriod = 5 * time.Second
	//-sync-k8s-to-cloud-sync-cluster-ip-services
	Cfg.K2C.FlagSyncClusterIPServices = spec.SyncFromK8S.SyncClusterIPServices
	//-sync-k8s-to-cloud-sync-load-balancer-services-endpoints
	Cfg.K2C.FlagSyncLoadBalancerEndpoints = spec.SyncFromK8S.SyncLoadBalancerEndpoints
	//-sync-k8s-to-cloud-node-port-sync-type
	Cfg.K2C.FlagNodePortSyncType = string(spec.SyncFromK8S.NodePortSyncType)
	//-sync-k8s-to-cloud-sync-ingress
	Cfg.K2C.FlagSyncIngress = spec.SyncFromK8S.SyncIngress
	//sync-k8s-to-cloud-sync-ingress-load-balancer-ips
	Cfg.K2C.FlagSyncIngressLoadBalancerIPs = spec.SyncFromK8S.SyncIngressLoadBalancerIPs
	//-sync-k8s-to-cloud-add-service-prefix
	Cfg.K2C.FlagAddServicePrefix = spec.SyncFromK8S.AddServicePrefix
	//-sync-k8s-to-cloud-add-k8s-namespace-as-service-suffix
	Cfg.K2C.FlagAddK8SNamespaceAsServiceSuffix = spec.SyncFromK8S.AddK8SNamespaceAsServiceSuffix

	//-sync-k8s-to-cloud-append-tag
	Cfg.K2C.FlagAppendTags = spec.SyncFromK8S.AppendTags

	//-sync-k8s-to-cloud-allow-k8s-namespaces
	Cfg.K2C.FlagAllowK8SNamespaces = spec.SyncFromK8S.AllowK8sNamespaces
	//-sync-k8s-to-cloud-deny-k8s-namespaces
	Cfg.K2C.FlagDenyK8SNamespaces = spec.SyncFromK8S.DenyK8sNamespaces

	//-sync-k8s-to-cloud-with-gateway
	Cfg.K2C.FlagWithGateway.Enable = spec.SyncFromK8S.WithGateway

	//-sync-k8s-to-cloud-consul-node-name
	Cfg.K2C.Consul.FlagConsulNodeName = spec.SyncFromK8S.ConsulNodeName
	//-sync-k8s-to-cloud-consul-k8s-tag
	Cfg.K2C.Consul.FlagConsulK8STag = spec.SyncFromK8S.ConsulK8STag
	//-sync-k8s-to-cloud-consul-enable-namespaces
	Cfg.K2C.Consul.FlagConsulEnableNamespaces = spec.SyncFromK8S.ConsulEnableNamespaces
	//-sync-k8s-to-cloud-consul-destination-namespace
	Cfg.K2C.Consul.FlagConsulDestinationNamespace = spec.SyncFromK8S.ConsulDestinationNamespace
	//-sync-k8s-to-cloud-consul-enable-k8s-namespace-mirroring
	Cfg.K2C.Consul.FlagConsulEnableK8SNSMirroring = spec.SyncFromK8S.ConsulEnableK8SNSMirroring
	//-sync-k8s-to-cloud-consul-k8s-namespace-mirroring-prefix
	Cfg.K2C.Consul.FlagConsulK8SNSMirroringPrefix = spec.SyncFromK8S.ConsulK8SNSMirroringPrefix
	//-sync-k8s-to-cloud-consul-cross-namespace-acl-policy
	Cfg.K2C.Consul.FlagConsulCrossNamespaceACLPolicy = spec.SyncFromK8S.ConsulCrossNamespaceACLPolicy
}
