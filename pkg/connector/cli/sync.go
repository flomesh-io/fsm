package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/connector/ktoc"
	"github.com/flomesh-io/fsm/pkg/connector/ktog"
	"github.com/flomesh-io/fsm/pkg/connector/provider"
)

func SyncCtoK(ctx context.Context, kubeClient kubernetes.Interface, discClient provider.ServiceDiscoveryClient, gatewayClient gwapi.Interface) {
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
	ktoc.WithGatewayAPI(Cfg.K2C.FlagWithGatewayAPI.Enable)

	if Cfg.K2C.FlagWithGatewayAPI.Enable {
		for {
			if fgwSvc, err := kubeClient.CoreV1().Services(Cfg.FsmNamespace).Get(ctx, fmt.Sprintf("fsm-gateway-%s", Cfg.FsmNamespace), metav1.GetOptions{}); err == nil {
				if fgwSvc != nil {
					if strings.EqualFold(Cfg.K2C.FlagWithGatewayAPI.Via, "ExternalIP") &&
						len(fgwSvc.Spec.ExternalIPs) > 0 &&
						len(fgwSvc.Spec.ExternalIPs[0]) > 0 &&
						len(fgwSvc.Spec.Ports) > 0 &&
						fgwSvc.Spec.Ports[0].Port > 0 {
						ktoc.WithGatewayViaAddr(fgwSvc.Spec.ExternalIPs[0])
						ktoc.WithGatewayViaPort(fgwSvc.Spec.Ports[0].Port)
						break
					}
					if strings.EqualFold(Cfg.K2C.FlagWithGatewayAPI.Via, "ExternalIP") &&
						len(fgwSvc.Status.LoadBalancer.Ingress) > 0 &&
						len(fgwSvc.Status.LoadBalancer.Ingress[0].IP) > 0 &&
						len(fgwSvc.Spec.Ports) > 0 &&
						fgwSvc.Spec.Ports[0].Port > 0 {
						ktoc.WithGatewayViaAddr(fgwSvc.Status.LoadBalancer.Ingress[0].IP)
						ktoc.WithGatewayViaPort(fgwSvc.Spec.Ports[0].Port)
						break
					}
					if strings.EqualFold(Cfg.K2C.FlagWithGatewayAPI.Via, "ClusterIP") &&
						len(fgwSvc.Spec.ClusterIPs) > 0 &&
						len(fgwSvc.Spec.ClusterIPs[0]) > 0 &&
						len(fgwSvc.Spec.Ports) > 0 &&
						fgwSvc.Spec.Ports[0].Port > 0 {
						ktoc.WithGatewayViaAddr(fgwSvc.Spec.ClusterIPs[0])
						ktoc.WithGatewayViaPort(fgwSvc.Spec.Ports[0].Port)
						break
					}
				}
			} else {
				fmt.Println(err.Error())
			}
			time.Sleep(time.Second * 5)
		}
	}

	ktoc.SetSyncCloudNamespace(Cfg.DeriveNamespace)

	allowSet := ToSet(Cfg.K2C.FlagAllowK8SNamespaces)
	denySet := ToSet(Cfg.K2C.FlagDenyK8SNamespaces)

	syncer := &ktoc.CloudSyncer{
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

func SyncKtoG(ctx context.Context, kubeClient kubernetes.Interface, gatewayClient gwapi.Interface) {
	allowSet := ToSet(Cfg.K2G.FlagAllowK8SNamespaces)
	denySet := ToSet(Cfg.K2G.FlagDenyK8SNamespaces)

	gatewayResource := &ktog.GatewayResource{}

	syncer := &ktog.GatewayRouteSyncer{
		SyncPeriod:        Cfg.K2G.FlagSyncPeriod,
		ServicePollPeriod: Cfg.K2G.FlagSyncPeriod * 2,
		GatewayResource:   gatewayResource,
	}

	serviceResource := &ktog.ServiceResource{
		FsmNamespace:          Cfg.FsmNamespace,
		Client:                kubeClient,
		GatewayClient:         gatewayClient,
		GatewayResource:       gatewayResource,
		Ctx:                   ctx,
		Syncer:                syncer,
		AllowK8sNamespacesSet: allowSet,
		DenyK8sNamespacesSet:  denySet,
		ExplicitEnable:        !Cfg.K2G.FlagDefaultSync,
	}

	gatewayResource.Service = serviceResource

	// Build the controller and start it
	gwCtl := &ktog.Controller{
		Resource: gatewayResource,
	}

	// Build the controller and start it
	ctl := &ktog.Controller{
		Resource: serviceResource,
	}

	go syncer.Run(ctx, gwCtl, ctl)
	go gwCtl.Run(ctx.Done())
	go ctl.Run(ctx.Done())
}
