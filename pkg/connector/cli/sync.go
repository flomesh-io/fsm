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
	ctok.EnabledGatewayAPI(Cfg.c2k.FlagWithGatewayAPI)
	ctok.SetSyncCloudNamespace(Cfg.DeriveNamespace)

	sink := ctok.NewSink(ctx, kubeClient, gatewayClient, Cfg.FsmNamespace)
	source := &ctok.Source{
		DiscClient:  discClient,
		Domain:      Cfg.TrustDomain,
		Sink:        sink,
		Prefix:      "",
		FilterTag:   Cfg.c2k.FlagFilterTag,
		PrefixTag:   Cfg.c2k.FlagPrefixTag,
		SuffixTag:   Cfg.c2k.FlagSuffixTag,
		PassingOnly: Cfg.c2k.FlagPassingOnly,
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
	allowSet := ToSet(Cfg.k2c.FlagAllowK8sNamespacesList)
	denySet := ToSet(Cfg.k2c.FlagDenyK8sNamespacesList)

	syncer := &ktoc.ConsulSyncer{
		DiscClient:              discClient,
		EnableNamespaces:        Cfg.k2c.FlagEnableNamespaces,
		CrossNamespaceACLPolicy: Cfg.k2c.FlagCrossNamespaceACLPolicy,
		SyncPeriod:              Cfg.k2c.FlagConsulWritePeriod,
		ServicePollPeriod:       Cfg.k2c.FlagConsulWritePeriod * 2,
		ConsulK8STag:            Cfg.k2c.FlagConsulK8STag,
		ConsulNodeName:          Cfg.k2c.FlagConsulNodeName,
	}
	go syncer.Run(ctx)

	serviceResource := ktoc.ServiceResource{
		Client:                     kubeClient,
		Syncer:                     syncer,
		Ctx:                        ctx,
		AllowK8sNamespacesSet:      allowSet,
		DenyK8sNamespacesSet:       denySet,
		ExplicitEnable:             !Cfg.k2c.FlagK8SDefault,
		ClusterIPSync:              Cfg.k2c.FlagSyncClusterIPServices,
		LoadBalancerEndpointsSync:  Cfg.k2c.FlagSyncLBEndpoints,
		NodePortSync:               ktoc.NodePortSyncType(Cfg.k2c.FlagNodePortSyncType),
		ConsulK8STag:               Cfg.k2c.FlagConsulK8STag,
		ConsulServicePrefix:        Cfg.k2c.FlagConsulServicePrefix,
		AddK8SNamespaceSuffix:      Cfg.k2c.FlagAddK8SNamespaceSuffix,
		EnableNamespaces:           Cfg.k2c.FlagEnableNamespaces,
		ConsulDestinationNamespace: Cfg.k2c.FlagConsulDestinationNamespace,
		EnableK8SNSMirroring:       Cfg.k2c.FlagEnableK8SNSMirroring,
		K8SNSMirroringPrefix:       Cfg.k2c.FlagK8SNSMirroringPrefix,
		ConsulNodeName:             Cfg.k2c.FlagConsulNodeName,
		EnableIngress:              Cfg.k2c.FlagEnableIngress,
		SyncLoadBalancerIPs:        Cfg.k2c.FlagLoadBalancerIPs,
	}

	// Build the controller and start it
	ctl := &ktoc.Controller{
		Resource: &serviceResource,
	}
	go ctl.Run(ctx.Done())
}
