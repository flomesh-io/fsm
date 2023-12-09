package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	gwapi "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/flomesh-io/fsm/pkg/connector"
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

	ctok.WithGateway(Cfg.C2K.FlagWithGateway.Enable)

	if Cfg.C2K.FlagWithGateway.Enable {
		ingressAddr, egressAddr := waitGatewayReady(ctx, kubeClient,
			connector.ViaGateway.IngressIPSelector,
			connector.ViaGateway.EgressIPSelector,
			int32(connector.ViaGateway.Ingress.HTTPPort),
			int32(connector.ViaGateway.Egress.HTTPPort),
			int32(connector.ViaGateway.Ingress.GRPCPort),
			int32(connector.ViaGateway.Egress.GRPCPort))
		connector.ViaGateway.IngressAddr = ingressAddr
		connector.ViaGateway.EgressAddr = egressAddr
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
	ktoc.WithGateway(Cfg.K2C.FlagWithGateway.Enable)

	if Cfg.K2C.FlagWithGateway.Enable {
		ingressAddr, egressAddr := waitGatewayReady(ctx, kubeClient,
			connector.ViaGateway.IngressIPSelector,
			connector.ViaGateway.EgressIPSelector,
			int32(connector.ViaGateway.Ingress.HTTPPort),
			int32(connector.ViaGateway.Egress.HTTPPort),
			int32(connector.ViaGateway.Ingress.GRPCPort),
			int32(connector.ViaGateway.Egress.GRPCPort))
		connector.ViaGateway.IngressAddr = ingressAddr
		connector.ViaGateway.EgressAddr = egressAddr
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
	waitGatewayReady(ctx, kubeClient,
		connector.ViaGateway.IngressIPSelector,
		connector.ViaGateway.EgressIPSelector,
		int32(connector.ViaGateway.Ingress.HTTPPort),
		int32(connector.ViaGateway.Egress.HTTPPort),
		int32(connector.ViaGateway.Ingress.GRPCPort),
		int32(connector.ViaGateway.Egress.GRPCPort))

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

func waitGatewayReady(ctx context.Context, kubeClient kubernetes.Interface, ingressIPSelector, egressIPSelector string, viaPorts ...int32) (ingressAddr, egressAddr string) {
	gatewaySvcName := fmt.Sprintf("fsm-gateway-%s", Cfg.FsmNamespace)
	for {
		if fgwSvc, err := kubeClient.CoreV1().Services(Cfg.FsmNamespace).Get(ctx, gatewaySvcName, metav1.GetOptions{}); err == nil {
			if fgwSvc != nil {
				if foundPorts, uncheckPorts := checkGatewayPorts(viaPorts, fgwSvc); foundPorts {
					if len(ingressAddr) == 0 && strings.EqualFold(ingressIPSelector, VIA_EXTERNAL_IP) &&
						len(fgwSvc.Spec.ExternalIPs) > 0 &&
						len(fgwSvc.Spec.ExternalIPs[0]) > 0 {
						ingressAddr = fgwSvc.Spec.ExternalIPs[0]
					}
					if len(ingressAddr) == 0 && strings.EqualFold(ingressIPSelector, VIA_EXTERNAL_IP) &&
						len(fgwSvc.Status.LoadBalancer.Ingress) > 0 &&
						len(fgwSvc.Status.LoadBalancer.Ingress[0].IP) > 0 {
						ingressAddr = fgwSvc.Status.LoadBalancer.Ingress[0].IP
					}
					if len(ingressAddr) == 0 && strings.EqualFold(ingressIPSelector, VIA_CLUSTER_IP) &&
						len(fgwSvc.Spec.ClusterIPs) > 0 &&
						len(fgwSvc.Spec.ClusterIPs[0]) > 0 {
						ingressAddr = fgwSvc.Spec.ClusterIPs[0]
					}
					if len(egressAddr) == 0 && strings.EqualFold(egressIPSelector, VIA_EXTERNAL_IP) &&
						len(fgwSvc.Spec.ExternalIPs) > 0 &&
						len(fgwSvc.Spec.ExternalIPs[0]) > 0 {
						egressAddr = fgwSvc.Spec.ExternalIPs[0]
					}
					if len(egressAddr) == 0 && strings.EqualFold(egressIPSelector, VIA_EXTERNAL_IP) &&
						len(fgwSvc.Status.LoadBalancer.Ingress) > 0 &&
						len(fgwSvc.Status.LoadBalancer.Ingress[0].IP) > 0 {
						egressAddr = fgwSvc.Status.LoadBalancer.Ingress[0].IP
					}
					if len(egressAddr) == 0 && strings.EqualFold(egressIPSelector, VIA_CLUSTER_IP) &&
						len(fgwSvc.Spec.ClusterIPs) > 0 &&
						len(fgwSvc.Spec.ClusterIPs[0]) > 0 {
						egressAddr = fgwSvc.Spec.ClusterIPs[0]
					}
					if len(ingressAddr) == 0 {
						log.Warn().Msgf("not find %s from fsm gateway: %s", ingressIPSelector, gatewaySvcName)
					} else if len(egressAddr) == 0 {
						log.Warn().Msgf("not find %s from fsm gateway: %s", egressIPSelector, gatewaySvcName)
					} else {
						return
					}
				} else {
					log.Warn().Msgf("not find matched port[HTTP:%v] from fsm gateway: %s", uncheckPorts, gatewaySvcName)
				}
			} else {
				log.Warn().Msgf("not find fsm gateway: %s", gatewaySvcName)
			}
		} else {
			log.Warn().Err(err)
		}
		time.Sleep(time.Second * 5)
	}
}

func checkGatewayPorts(viaPorts []int32, fgwSvc *corev1.Service) (bool, map[int32]bool) {
	foundPorts := false
	uncheckPorts := make(map[int32]bool)
	if len(viaPorts) > 0 {
		for _, viaPort := range viaPorts {
			if viaPort > 0 {
				uncheckPorts[viaPort] = true
			}
		}
	}
	if len(fgwSvc.Spec.Ports) > 0 && len(uncheckPorts) > 0 {
		for _, port := range fgwSvc.Spec.Ports {
			for _, viaPort := range viaPorts {
				if viaPort > 0 && port.Port == viaPort {
					delete(uncheckPorts, viaPort)
					break
				}
			}
			if len(uncheckPorts) == 0 {
				break
			}
		}
		if len(uncheckPorts) == 0 {
			foundPorts = true
		}
	}
	return foundPorts, uncheckPorts
}
