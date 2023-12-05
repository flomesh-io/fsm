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

const (
	VIA_EXTERNAL_IP = "ExternalIP"
	VIA_CLUSTER_IP  = "ClusterIP"
)

func SyncCtoK(ctx context.Context, kubeClient kubernetes.Interface, discClient provider.ServiceDiscoveryClient) {
	ctok.SetSyncCloudNamespace(Cfg.DeriveNamespace)

	ctok.WithGatewayEgress(Cfg.C2K.FlagWithGatewayEgress.Enable)

	if Cfg.C2K.FlagWithGatewayEgress.Enable {
		viaAddr := waitGatewayReady(ctx, kubeClient, VIA_CLUSTER_IP, int32(Cfg.C2K.FlagWithGatewayEgress.ViaEgressPort))
		ctok.WithGatewayEgressAddr(viaAddr)
		ctok.WithGatewayEgressPort(int32(Cfg.C2K.FlagWithGatewayEgress.ViaEgressPort))
	}

	sink := ctok.NewSink(ctx, kubeClient, discClient, Cfg.FsmNamespace)
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
	ktoc.WithGatewayIngress(Cfg.K2C.FlagWithGatewayIngress.Enable)

	if Cfg.K2C.FlagWithGatewayIngress.Enable {
		viaAddr := waitGatewayReady(ctx, kubeClient, Cfg.K2C.FlagWithGatewayIngress.ViaIngressType, int32(Cfg.K2C.FlagWithGatewayIngress.ViaIngressPort))
		ktoc.WithGatewayIngressAddr(viaAddr)
		ktoc.WithGatewayIngressPort(int32(Cfg.K2C.FlagWithGatewayIngress.ViaIngressPort))
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
	ktog.WithGatewayIngressHTTPPort(int32(Cfg.K2G.FlagIngress.HTTPPort))
	ktog.WithGatewayIngressGRPCPort(int32(Cfg.K2G.FlagIngress.GRPCPort))
	ktog.WithGatewayEgressHTTPPort(int32(Cfg.K2G.FlagEgress.HTTPPort))
	ktog.WithGatewayEgressGRPCPort(int32(Cfg.K2G.FlagEgress.GRPCPort))

	if Cfg.K2G.FlagIngress.HTTPPort > 0 {
		waitGatewayReady(ctx, kubeClient, VIA_CLUSTER_IP, int32(Cfg.K2G.FlagIngress.HTTPPort))
	}

	if Cfg.K2G.FlagEgress.HTTPPort > 0 {
		waitGatewayReady(ctx, kubeClient, VIA_CLUSTER_IP, int32(Cfg.K2G.FlagEgress.HTTPPort))
	}

	if Cfg.K2G.FlagIngress.GRPCPort > 0 {
		waitGatewayReady(ctx, kubeClient, VIA_CLUSTER_IP, int32(Cfg.K2G.FlagIngress.GRPCPort))
	}

	if Cfg.K2G.FlagEgress.GRPCPort > 0 {
		waitGatewayReady(ctx, kubeClient, VIA_CLUSTER_IP, int32(Cfg.K2G.FlagEgress.GRPCPort))
	}

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

func waitGatewayReady(ctx context.Context, kubeClient kubernetes.Interface, viaAddrType string, viaPort int32) (viaAddr string) {
	gatewaySvcName := fmt.Sprintf("fsm-gateway-%s", Cfg.FsmNamespace)
	for {
		if fgwSvc, err := kubeClient.CoreV1().Services(Cfg.FsmNamespace).Get(ctx, gatewaySvcName, metav1.GetOptions{}); err == nil {
			if fgwSvc != nil {
				if strings.EqualFold(viaAddrType, VIA_EXTERNAL_IP) &&
					len(fgwSvc.Spec.ExternalIPs) > 0 &&
					len(fgwSvc.Spec.ExternalIPs[0]) > 0 {
					if len(fgwSvc.Spec.Ports) > 0 {
						for _, port := range fgwSvc.Spec.Ports {
							if port.Port == viaPort {
								viaAddr = fgwSvc.Spec.ExternalIPs[0]
								return
							}
						}
					}
					log.Warn().Msgf("not find matched port[HTTP:%s] from fsm gateway: %s", viaPort, gatewaySvcName)
				}
				if strings.EqualFold(viaAddrType, VIA_EXTERNAL_IP) &&
					len(fgwSvc.Status.LoadBalancer.Ingress) > 0 &&
					len(fgwSvc.Status.LoadBalancer.Ingress[0].IP) > 0 {
					if len(fgwSvc.Spec.Ports) > 0 {
						for _, port := range fgwSvc.Spec.Ports {
							if port.Port == viaPort {
								viaAddr = fgwSvc.Status.LoadBalancer.Ingress[0].IP
								return
							}
						}
					}
					log.Warn().Msgf("not find matched port[HTTP:%d] from fsm gateway: %s", viaPort, gatewaySvcName)
				}
				if strings.EqualFold(viaAddrType, VIA_CLUSTER_IP) &&
					len(fgwSvc.Spec.ClusterIPs) > 0 &&
					len(fgwSvc.Spec.ClusterIPs[0]) > 0 {
					if len(fgwSvc.Spec.Ports) > 0 {
						for _, port := range fgwSvc.Spec.Ports {
							if port.Port == viaPort {
								viaAddr = fgwSvc.Spec.ClusterIPs[0]
								return
							}
						}
					}
					log.Warn().Msgf("not find matched port[HTTP:%d] from fsm gateway: %s", viaPort, gatewaySvcName)
				}
			}
			log.Warn().Msgf("not find %s from fsm gateway: %s", viaAddrType, gatewaySvcName)
		} else {
			log.Warn().Err(err)
		}
		time.Sleep(time.Second * 5)
	}
}
