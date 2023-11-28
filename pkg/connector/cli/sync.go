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
	allowSet := ToSet(Cfg.K2C.FlagAllowK8sNamespacesList)
	denySet := ToSet(Cfg.K2C.FlagDenyK8sNamespacesList)

	syncer := &ktoc.ConsulSyncer{
		DiscClient:              discClient,
		EnableNamespaces:        Cfg.K2C.FlagEnableNamespaces,
		CrossNamespaceACLPolicy: Cfg.K2C.FlagCrossNamespaceACLPolicy,
		SyncPeriod:              Cfg.K2C.FlagConsulWritePeriod,
		ServicePollPeriod:       Cfg.K2C.FlagConsulWritePeriod * 2,
		ConsulK8STag:            Cfg.K2C.FlagConsulK8STag,
		ConsulNodeName:          Cfg.K2C.FlagConsulNodeName,
	}
	go syncer.Run(ctx)

	serviceResource := ktoc.ServiceResource{
		Client:                     kubeClient,
		Syncer:                     syncer,
		Ctx:                        ctx,
		AllowK8sNamespacesSet:      allowSet,
		DenyK8sNamespacesSet:       denySet,
		ExplicitEnable:             !Cfg.K2C.FlagK8SDefault,
		ClusterIPSync:              Cfg.K2C.FlagSyncClusterIPServices,
		LoadBalancerEndpointsSync:  Cfg.K2C.FlagSyncLBEndpoints,
		NodePortSync:               ktoc.NodePortSyncType(Cfg.K2C.FlagNodePortSyncType),
		ConsulK8STag:               Cfg.K2C.FlagConsulK8STag,
		ConsulServicePrefix:        Cfg.K2C.FlagConsulServicePrefix,
		AddK8SNamespaceSuffix:      Cfg.K2C.FlagAddK8SNamespaceSuffix,
		EnableNamespaces:           Cfg.K2C.FlagEnableNamespaces,
		ConsulDestinationNamespace: Cfg.K2C.FlagConsulDestinationNamespace,
		EnableK8SNSMirroring:       Cfg.K2C.FlagEnableK8SNSMirroring,
		K8SNSMirroringPrefix:       Cfg.K2C.FlagK8SNSMirroringPrefix,
		ConsulNodeName:             Cfg.K2C.FlagConsulNodeName,
		EnableIngress:              Cfg.K2C.FlagEnableIngress,
		SyncLoadBalancerIPs:        Cfg.K2C.FlagLoadBalancerIPs,
	}

	// Build the controller and start it
	ctl := &ktoc.Controller{
		Resource: &serviceResource,
	}
	go ctl.Run(ctx.Done())
}
