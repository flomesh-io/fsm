package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/connector/ctok"
	"github.com/flomesh-io/fsm/pkg/connector/ktoc"
	"github.com/flomesh-io/fsm/pkg/connector/ktog"
	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

func (c *client) syncCtoK() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelFuncs = append(c.cancelFuncs, cancelFunc)

	if c.GetC2KWithGateway() {
		c.waitViaGatewayReady(ctx, c.configClient)
	}

	syncer := ctok.NewCtoKSyncer(c, c.discClient, c.kubeClient, ctx, Cfg.FsmNamespace, Cfg.Workers)
	source := ctok.NewCtoKSource(c, syncer, c.discClient, Cfg.TrustDomain)

	syncer.SetMicroAggregator(source)
	syncer.Ready()

	go source.Run(ctx)

	// Build the controller and start it
	ctl := &connector.CacheController{Resource: syncer}
	go ctl.Run(ctx.Done())
}

func (c *client) syncKtoC() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelFuncs = append(c.cancelFuncs, cancelFunc)

	if c.GetK2CWithGateway() {
		c.waitViaGatewayReady(ctx, c.configClient)
	}

	syncer := ktoc.NewKtoCSyncer(c, c.discClient)
	go syncer.Run(ctx)

	msgBroker := messaging.NewBroker(ctx.Done())
	serviceSource := ktoc.NewKtoCSource(c, syncer, ctx, msgBroker, c.kubeClient, c.discClient)
	cacheCtl := &connector.CacheController{Resource: serviceSource}

	go serviceSource.BroadcastListener(ctx.Done(), c.GetSyncPeriod())
	go cacheCtl.Run(ctx.Done())
}

func (c *client) syncKtoG() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelFuncs = append(c.cancelFuncs, cancelFunc)

	ingressAddr, egressAddr, clusterIP, externalIP := waitGatewayReady(ctx, c.kubeClient,
		c.GetViaFgwName(),
		c.GetViaIngressIPSelector(),
		c.GetViaEgressIPSelector(),
		int32(c.GetViaIngressHTTPPort()),
		int32(c.GetViaEgressHTTPPort()),
		int32(c.GetViaIngressGRPCPort()),
		int32(c.GetViaEgressGRPCPort()))

	meshConfigClient := c.configClient.ConfigV1alpha3().MeshConfigs(Cfg.FsmNamespace)
	meshConfig, err := meshConfigClient.Get(ctx, Cfg.FsmMeshConfigName, metav1.GetOptions{})
	if err != nil {
		log.Fatal().Err(err)
	}

	meshConfigChanged := false

	viaGateway := &meshConfig.Spec.Connector.ViaGateway
	if !strings.EqualFold(viaGateway.IngressAddr, ingressAddr) ||
		!strings.EqualFold(viaGateway.EgressAddr, egressAddr) ||
		!strings.EqualFold(viaGateway.ClusterIP, clusterIP) ||
		!strings.EqualFold(viaGateway.ExternalIP, externalIP) ||
		viaGateway.IngressHTTPPort != c.GetViaIngressHTTPPort() ||
		viaGateway.IngressGRPCPort != c.GetViaIngressGRPCPort() ||
		viaGateway.EgressHTTPPort != c.GetViaEgressHTTPPort() ||
		viaGateway.EgressGRPCPort != c.GetViaEgressGRPCPort() {
		viaGateway.ClusterIP = clusterIP
		viaGateway.ExternalIP = externalIP
		viaGateway.IngressAddr = ingressAddr
		viaGateway.IngressHTTPPort = c.GetViaIngressHTTPPort()
		viaGateway.IngressGRPCPort = c.GetViaIngressGRPCPort()
		viaGateway.EgressAddr = egressAddr
		viaGateway.EgressHTTPPort = c.GetViaEgressHTTPPort()
		viaGateway.EgressGRPCPort = c.GetViaEgressGRPCPort()
		meshConfigChanged = true
	}

	if meshConfigChanged {
		_, err = meshConfigClient.Update(ctx, meshConfig, metav1.UpdateOptions{})
		if err != nil {
			log.Fatal().Err(err)
		}
	}

	gatewaySource := &ktog.GatewaySource{InterceptionMode: meshConfig.Spec.Traffic.InterceptionMode}

	syncer := ktog.NewKtoGSyncer(c, gatewaySource)

	serviceResource := ktog.NewKtoGSource(
		c, syncer, gatewaySource,
		Cfg.FsmNamespace,
		c.kubeClient, c.gatewayClient,
		ctx)

	gatewaySource.SetServiceResource(serviceResource)
	gatewaySource.SetInformers(c.informers)

	// Build the controller and start it
	gwCtl := &connector.CacheController{Resource: gatewaySource}

	// Build the controller and start it
	svcCtl := &connector.CacheController{Resource: serviceResource}

	go syncer.Run(ctx, gwCtl, svcCtl)
	go gwCtl.Run(ctx.Done())
	go svcCtl.Run(ctx.Done())
}

func (c *client) waitViaGatewayReady(ctx context.Context, configClient configClientset.Interface) {
	meshConfigClient := configClient.ConfigV1alpha3().MeshConfigs(Cfg.FsmNamespace)
	for {
		meshConfig, err := meshConfigClient.Get(ctx, Cfg.FsmMeshConfigName, metav1.GetOptions{})
		if err != nil {
			log.Warn().Err(err)
		} else {
			viaGateway := &meshConfig.Spec.Connector.ViaGateway
			if len(viaGateway.IngressAddr) > 0 && len(viaGateway.EgressAddr) > 0 {
				c.SetViaIngressAddr(viaGateway.IngressAddr)
				c.SetViaIngressHTTPPort(viaGateway.IngressHTTPPort)
				c.SetViaIngressGRPCPort(viaGateway.IngressGRPCPort)

				c.SetViaEgressAddr(viaGateway.EgressAddr)
				c.SetViaEgressHTTPPort(viaGateway.EgressHTTPPort)
				c.SetViaEgressGRPCPort(viaGateway.EgressGRPCPort)
				break
			}
		}
		time.Sleep(time.Second * 5)
	}
}

func waitGatewayReady(ctx context.Context, kubeClient kubernetes.Interface, viaFgwName string, ingressIPSelector, egressIPSelector ctv1.AddrSelector, viaPorts ...int32) (ingressAddr, egressAddr, clusterIP, externalIP string) {
	gatewaySvcName := fmt.Sprintf("%s-%s-%s-%s", constants.FSMGatewayName, Cfg.FsmNamespace, viaFgwName, constants.ProtocolTCP)
	for {
		if fgwSvc, err := kubeClient.CoreV1().Services(Cfg.FsmNamespace).Get(ctx, gatewaySvcName, metav1.GetOptions{}); err == nil {
			if fgwSvc != nil {
				if foundPorts, uncheckPorts := checkGatewayPorts(viaPorts, fgwSvc); foundPorts {
					ingressAddr, egressAddr, clusterIP, externalIP = checkGatewayIPs(fgwSvc, ingressIPSelector, egressIPSelector)
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

func checkGatewayIPs(fgwSvc *corev1.Service, ingressIPSelector, egressIPSelector ctv1.AddrSelector) (ingressAddr, egressAddr, clusterIP, externalIP string) {
	if len(externalIP) == 0 && len(fgwSvc.Spec.ExternalIPs) > 0 && len(fgwSvc.Spec.ExternalIPs[0]) > 0 {
		externalIP = fgwSvc.Spec.ExternalIPs[0]
	}
	if len(externalIP) == 0 && len(fgwSvc.Status.LoadBalancer.Ingress) > 0 && len(fgwSvc.Status.LoadBalancer.Ingress[0].IP) > 0 {
		externalIP = fgwSvc.Status.LoadBalancer.Ingress[0].IP
	}
	if len(clusterIP) == 0 && len(fgwSvc.Spec.ClusterIPs) > 0 && len(fgwSvc.Spec.ClusterIPs[0]) > 0 {
		clusterIP = fgwSvc.Spec.ClusterIPs[0]
	}

	ingressAddr = selectIP(ingressAddr, ingressIPSelector, ctv1.ExternalIP, fgwSvc.Spec.ExternalIPs)
	ingressAddr = selectIngressIP(ingressAddr, ingressIPSelector, ctv1.ExternalIP, fgwSvc.Status.LoadBalancer.Ingress)
	ingressAddr = selectIP(ingressAddr, ingressIPSelector, ctv1.ClusterIP, fgwSvc.Spec.ClusterIPs)
	egressAddr = selectIP(egressAddr, egressIPSelector, ctv1.ExternalIP, fgwSvc.Spec.ExternalIPs)
	egressAddr = selectIngressIP(egressAddr, egressIPSelector, ctv1.ExternalIP, fgwSvc.Status.LoadBalancer.Ingress)
	egressAddr = selectIP(egressAddr, egressIPSelector, ctv1.ClusterIP, fgwSvc.Spec.ClusterIPs)
	return
}

func selectIP(ip string, ipSelector, ipVia ctv1.AddrSelector, ips []string) string {
	if len(ip) == 0 && strings.EqualFold(string(ipSelector), string(ipVia)) && len(ips) > 0 && len(ips[0]) > 0 {
		return ips[0]
	}
	return ip
}

func selectIngressIP(ip string, ipSelector, ipVia ctv1.AddrSelector, ingress []corev1.LoadBalancerIngress) string {
	if len(ip) == 0 && strings.EqualFold(string(ipSelector), string(ipVia)) && len(ingress) > 0 && len(ingress[0].IP) > 0 {
		return ingress[0].IP
	}
	return ip
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
