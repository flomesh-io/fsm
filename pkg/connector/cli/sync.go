package cli

import (
	"context"

	"k8s.io/client-go/kubernetes"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/connector/ktoc"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
)

func SyncCtoK(ctx context.Context, kubeClient kubernetes.Interface, discClient provider.ServiceDiscoveryClient, gatewayClient gwapi.Interface) {
	ctok.EnabledGatewayAPI(Cfg.C2K.FlagWithGatewayAPI)
	ctok.SetSyncCloudNamespace(Cfg.DeriveNamespace)

	sink := ctok.NewSink(ctx, kubeClient, gatewayClient, Cfg.FsmNamespace)
	source := &ctok.Source{
		DiscClient:  discClient,
		Domain:      Cfg.TrustDomain,
		Sink:        sink,
		Prefix:      "",
		FilterTag:   Cfg.C2K.FlagFilterTag,
		PrefixTag:   Cfg.C2K.FlagPrefixTag,
		SuffixTag:   Cfg.C2K.FlagSuffixTag,
		PassingOnly: Cfg.C2K.FlagPassingOnly,
	}
	sink.MicroAggregator = source
	go source.Run(ctx)

	// Build the controller and start it
	ctl := &ctok.Controller{
		Resource: sink,
	}
	go ctl.Run(ctx.Done())
}

func SyncKtoC(ctx context.Context, kubeClient kubernetes.Interface, discClient provider.ServiceDiscoveryClient) {
	allowSet := ToSet(Cfg.K2C.FlagAllowK8SNamespaces)
	denySet := ToSet(Cfg.K2C.FlagDenyK8SNamespaces)

	syncer := &ktoc.ConsulSyncer{
		DiscClient:              discClient,
		EnableNamespaces:        Cfg.K2C.Consul.FlagConsulEnableNamespaces,
		CrossNamespaceACLPolicy: Cfg.K2C.Consul.FlagConsulCrossNamespaceACLPolicy,
		SyncPeriod:              Cfg.K2C.FlagSyncPeriod,
		ServicePollPeriod:       Cfg.K2C.FlagSyncPeriod * 2,
		ConsulK8STag:            Cfg.K2C.Consul.FlagConsulK8STag,
		ConsulNodeName:          Cfg.K2C.Consul.FlagConsulNodeName,
	}
	go syncer.Run(ctx)

	serviceResource := ktoc.ServiceResource{
		Client:                         kubeClient,
		Syncer:                         syncer,
		Ctx:                            ctx,
		AllowK8sNamespacesSet:          allowSet,
		DenyK8sNamespacesSet:           denySet,
		ExplicitEnable:                 !Cfg.K2C.FlagDefaultSync,
		ClusterIPSync:                  Cfg.K2C.FlagSyncClusterIPServices,
		LoadBalancerEndpointsSync:      Cfg.K2C.FlagSyncLoadBalancerEndpoints,
		NodePortSync:                   ktoc.NodePortSyncType(Cfg.K2C.FlagNodePortSyncType),
		ConsulK8STag:                   Cfg.K2C.Consul.FlagConsulK8STag,
		AddServicePrefix:               Cfg.K2C.FlagAddServicePrefix,
		AddK8SNamespaceAsServiceSuffix: Cfg.K2C.FlagAddK8SNamespaceAsServiceSuffix,
		EnableNamespaces:               Cfg.K2C.Consul.FlagConsulEnableNamespaces,
		ConsulDestinationNamespace:     Cfg.K2C.Consul.FlagConsulDestinationNamespace,
		EnableK8SNSMirroring:           Cfg.K2C.Consul.FlagConsulEnableK8SNSMirroring,
		K8SNSMirroringPrefix:           Cfg.K2C.Consul.FlagConsulK8SNSMirroringPrefix,
		ConsulNodeName:                 Cfg.K2C.Consul.FlagConsulNodeName,
		SyncIngress:                    Cfg.K2C.FlagSyncIngress,
		SyncIngressLoadBalancerIPs:     Cfg.K2C.FlagSyncIngressLoadBalancerIPs,
	}

	// Build the controller and start it
	ctl := &ktoc.Controller{
		Resource: &serviceResource,
	}
	go ctl.Run(ctx.Done())
}
