package repo

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
	"github.com/flomesh-io/fsm/pkg/utils"
)

func generatePipyInboundTrafficPolicy(meshCatalog catalog.MeshCataloger, pipyConf *PipyConf, inboundPolicy *trafficpolicy.InboundMeshTrafficPolicy, trustDomain string, proxy *pipy.Proxy) {
	itp := pipyConf.newInboundTrafficPolicy()

	for _, trafficMatch := range inboundPolicy.TrafficMatches {
		destinationProtocol := strings.ToLower(trafficMatch.DestinationProtocol)
		upstreamSvc := trafficMatchToMeshSvc(trafficMatch)
		clusterConfigs := getMeshClusterConfigs(inboundPolicy.ClustersConfigs, service.ClusterName(upstreamSvc.SidecarLocalClusterName()))
		if len(clusterConfigs) == 0 {
			continue
		}
		for _, clusterConfig := range clusterConfigs {
			tm := itp.newTrafficMatch(Port(clusterConfig.Service.Port))
			tm.setProtocol(Protocol(destinationProtocol))
			tm.setPort(Port(trafficMatch.DestinationPort))
			tm.setTCPServiceRateLimit(trafficMatch.RateLimit)

			if destinationProtocol == constants.ProtocolHTTP ||
				trafficMatch.DestinationProtocol == constants.ProtocolGRPC {
				upstreamPolicyName := upstreamSvc.PolicyName(false)

				httpRouteConfig := getInboundHTTPRouteConfigs(inboundPolicy.HTTPRouteConfigsPerPort,
					int(upstreamSvc.TargetPort), upstreamPolicyName)
				if httpRouteConfig == nil {
					continue
				}

				ruleRef := &HTTPRouteRuleRef{RuleName: HTTPRouteRuleName(httpRouteConfig.Name)}
				hsrrs := tm.newHTTPServiceRouteRules(ruleRef)
				hsrrs.setHTTPServiceRateLimit(trafficMatch.RateLimit)
				hsrrs.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
				for _, hostname := range httpRouteConfig.Hostnames {
					tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleRef)
				}

				for _, rule := range httpRouteConfig.Rules {
					if len(rule.Route.HTTPRouteMatch.Path) == 0 {
						continue
					}

					route := rule.Route
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					hsrr, duplicate := hsrrs.newHTTPServiceRouteRule(httpMatch)
					if !duplicate {
						hsrr.setRateLimit(rule.Route.RateLimit)
						for routeCluster := range rule.Route.WeightedClusters.Iter() {
							weightedCluster := routeCluster.(service.WeightedCluster)
							hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName),
								Weight(weightedCluster.Weight))
						}
					}

					for allowedPrincipal := range rule.AllowedPrincipals.Iter() {
						servicePrincipal := allowedPrincipal.(string)
						serviceIdentity := identity.FromPrincipal(servicePrincipal, trustDomain)
						hsrr.addAllowedService(ServiceName(serviceIdentity))
						if identity.WildcardPrincipal == servicePrincipal || pipyConf.isPermissiveTrafficPolicyMode() {
							continue
						}
						allowedServiceEndpoints := getEndpointsForProxyIdentity(meshCatalog, serviceIdentity)
						if len(allowedServiceEndpoints) > 0 {
							for _, allowedEndpoint := range allowedServiceEndpoints {
								hsrrs.addAllowedEndpoint(Address(allowedEndpoint.IP.String()), ServiceName(serviceIdentity))
							}
						}
					}
				}
			} else if destinationProtocol == constants.ProtocolTCP ||
				destinationProtocol == constants.ProtocolTCPServerFirst {
				tsrr := tm.newTCPServiceRouteRules()
				tsrr.addWeightedCluster(ClusterName(clusterConfig.Name), Weight(constants.ClusterWeightAcceptAll))
				tsrr.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
			}
		}
	}

	for _, cluster := range inboundPolicy.ClustersConfigs {
		clusterConfigs := itp.newClusterConfigs(ClusterName(cluster.Name))
		port := Port(cluster.Port)
		weight := Weight(constants.ClusterWeightAcceptAll)
		if proxy.VM {
			address := Address(proxy.MachineIP.String())
			clusterConfigs.addWeightedEndpoint(address, port, weight)
		} else {
			address := Address(cluster.Address)
			clusterConfigs.addWeightedEndpoint(address, port, weight)
		}
	}
}

func generatePipyOutboundTrafficRoutePolicy(_ catalog.MeshCataloger, pipyConf *PipyConf, cfg configurator.Configurator,
	outboundPolicy *trafficpolicy.OutboundMeshTrafficPolicy,
	desiredSuffix string) map[service.ClusterName]*WeightedCluster {
	if len(outboundPolicy.TrafficMatches) == 0 {
		return nil
	}

	otp := pipyConf.newOutboundTrafficPolicy()
	dependClusters := make(map[service.ClusterName]*WeightedCluster)

	wildcardIPRange := generatePipyWildcardIPRanges(cfg)

	for _, trafficMatch := range outboundPolicy.TrafficMatches {
		destinationProtocol := strings.ToLower(trafficMatch.DestinationProtocol)
		trafficMatchName := trafficMatch.Name
		if destinationProtocol == constants.ProtocolHTTP || destinationProtocol == constants.ProtocolGRPC {
			trafficMatchName = constants.ProtocolHTTP
		}
		tm, exist := otp.newTrafficMatch(Port(trafficMatch.DestinationPort), trafficMatchName)
		if !exist {
			tm.setProtocol(Protocol(destinationProtocol))
			tm.setPort(Port(trafficMatch.DestinationPort))
		}

		generatePipyOutboundDestinationIPRanges(trafficMatch, tm, wildcardIPRange)

		if destinationProtocol == constants.ProtocolHTTP ||
			destinationProtocol == constants.ProtocolGRPC {
			upstreamSvc := trafficMatchToMeshSvc(trafficMatch)

			upstreamPolicyName := upstreamSvc.PolicyName(true)
			httpRouteConfigs := getOutboundHTTPRouteConfigs(outboundPolicy.HTTPRouteConfigsPerPort,
				int(upstreamSvc.TargetPort), upstreamPolicyName, trafficMatch.WeightedClusters)
			if len(httpRouteConfigs) == 0 {
				continue
			}

			for _, httpRouteConfig := range httpRouteConfigs {
				ruleRef := &HTTPRouteRuleRef{RuleName: HTTPRouteRuleName(httpRouteConfig.Name)}
				hsrrs := tm.newHTTPServiceRouteRules(ruleRef)
				hsrrs.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
				for _, hostname := range httpRouteConfig.Hostnames {
					tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleRef, desiredSuffix)
				}

				for _, route := range httpRouteConfig.Routes {
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					hsrr, _ := hsrrs.newHTTPServiceRouteRule(httpMatch)
					for cluster := range route.WeightedClusters.Iter() {
						serviceCluster := cluster.(service.WeightedCluster)
						weightedCluster := &WeightedCluster{
							WeightedCluster: serviceCluster,
							RetryPolicy:     route.RetryPolicy,
						}
						if _, exists := dependClusters[weightedCluster.ClusterName]; !exists {
							dependClusters[weightedCluster.ClusterName] = weightedCluster
						}
						hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
					}
				}
			}
		} else if destinationProtocol == constants.ProtocolTCP ||
			destinationProtocol == constants.ProtocolTCPServerFirst {
			tsrr := tm.newTCPServiceRouteRules()
			tsrr.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
			for _, serviceCluster := range trafficMatch.WeightedClusters {
				weightedCluster := &WeightedCluster{
					WeightedCluster: serviceCluster,
				}
				if _, exists := dependClusters[weightedCluster.ClusterName]; !exists {
					dependClusters[weightedCluster.ClusterName] = weightedCluster
				}
				tsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
			}
		} else if destinationProtocol == constants.ProtocolHTTPS {
			upstreamSvc := trafficMatchToMeshSvc(trafficMatch)

			upstreamPolicyName := upstreamSvc.PolicyName(true)
			httpRouteConfigs := getOutboundHTTPRouteConfigs(outboundPolicy.HTTPRouteConfigsPerPort,
				int(upstreamSvc.TargetPort), upstreamPolicyName, trafficMatch.WeightedClusters)
			if len(httpRouteConfigs) == 0 {
				continue
			}

			tsrr := tm.newTCPServiceRouteRules()
			tsrr.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
			for _, httpRouteConfig := range httpRouteConfigs {
				for _, route := range httpRouteConfig.Routes {
					for cluster := range route.WeightedClusters.Iter() {
						serviceCluster := cluster.(service.WeightedCluster)
						weightedCluster := &WeightedCluster{
							WeightedCluster: serviceCluster,
							RetryPolicy:     route.RetryPolicy,
						}
						if _, exists := dependClusters[weightedCluster.ClusterName]; !exists {
							dependClusters[weightedCluster.ClusterName] = weightedCluster
						}
						tsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
					}
				}
			}
		}
	}

	return dependClusters
}

func generatePipyOutboundDestinationIPRanges(trafficMatch *trafficpolicy.TrafficMatch, tm *OutboundTrafficMatch, wildcardIPRange []string) {
	for _, ipRange := range trafficMatch.DestinationIPRanges {
		tm.addDestinationIPRange(DestinationIPRange(ipRange), nil)
	}

	if len(wildcardIPRange) > 0 {
		for _, ipRange := range wildcardIPRange {
			tm.addDestinationIPRange(DestinationIPRange(ipRange), nil)
		}
	}
}

func generatePipyWildcardIPRanges(cfg configurator.Configurator) []string {
	var wildcardIPv4 []string
	if cfg.IsLocalDNSProxyEnabled() {
		dnsProxy := cfg.GetMeshConfig().Spec.Sidecar.LocalDNSProxy
		if cfg.IsWildcardDNSProxyEnabled() {
			for _, ipAddr := range dnsProxy.Wildcard.IPs {
				if len(ipAddr.IPv4) > 0 {
					wildcardIPv4 = append(wildcardIPv4, fmt.Sprintf("%s/32", ipAddr.IPv4))
				}
			}
		}
	}
	return wildcardIPv4
}

func generatePipyEgressTrafficRoutePolicy(meshCatalog catalog.MeshCataloger, pipyConf *PipyConf, egressPolicy *trafficpolicy.EgressTrafficPolicy, desiredSuffix string) map[service.ClusterName]*WeightedCluster {
	if len(egressPolicy.TrafficMatches) == 0 {
		return nil
	}

	otp := pipyConf.newOutboundTrafficPolicy()
	dependClusters := make(map[service.ClusterName]*WeightedCluster)
	for _, trafficMatch := range egressPolicy.TrafficMatches {
		destinationProtocol := strings.ToLower(trafficMatch.DestinationProtocol)
		tm, exists := otp.newTrafficMatch(Port(trafficMatch.DestinationPort), trafficMatch.Name)
		if !exists {
			tm.setProtocol(Protocol(destinationProtocol))
			tm.setPort(Port(trafficMatch.DestinationPort))
		}

		var destinationSpec = getEgressClusterDestinationSpec(meshCatalog, egressPolicy, trafficMatch)

		for _, ipRange := range trafficMatch.DestinationIPRanges {
			tm.addDestinationIPRange(DestinationIPRange(ipRange), destinationSpec)
		}

		if destinationProtocol == constants.ProtocolHTTP || destinationProtocol == constants.ProtocolGRPC {
			httpRouteConfigs := getEgressHTTPRouteConfigs(egressPolicy.HTTPRouteConfigsPerPort, trafficMatch.DestinationPort)
			if len(httpRouteConfigs) == 0 {
				continue
			}

			for _, httpRouteConfig := range httpRouteConfigs {
				ruleRef := &HTTPRouteRuleRef{RuleName: HTTPRouteRuleName(httpRouteConfig.Name)}
				hsrrs := tm.newHTTPServiceRouteRules(ruleRef)
				hsrrs.setEgressForwardGateway(trafficMatch.EgressGateWay)
				for _, hostname := range httpRouteConfig.Hostnames {
					tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleRef, desiredSuffix)
				}
				for _, rule := range httpRouteConfig.RoutingRules {
					route := rule.Route
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					hsrr, _ := hsrrs.newHTTPServiceRouteRule(httpMatch)
					for cluster := range route.WeightedClusters.Iter() {
						serviceCluster := cluster.(service.WeightedCluster)
						weightedCluster := new(WeightedCluster)
						weightedCluster.WeightedCluster = serviceCluster
						weightedCluster.RetryPolicy = route.RetryPolicy
						if _, exist := dependClusters[weightedCluster.ClusterName]; !exist {
							dependClusters[weightedCluster.ClusterName] = weightedCluster
						}
						hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
					}

					for _, allowedIPRange := range rule.AllowedDestinationIPRanges {
						tm.addDestinationIPRange(DestinationIPRange(allowedIPRange), destinationSpec)
					}
				}
			}
		} else if destinationProtocol == constants.ProtocolHTTPS {
			weightedCluster := new(WeightedCluster)
			weightedCluster.ClusterName = service.ClusterName(trafficMatch.Cluster)
			weightedCluster.Weight = constants.ClusterWeightAcceptAll
			tsrr := tm.newTCPServiceRouteRules()
			tsrr.setEgressForwardGateway(trafficMatch.EgressGateWay)
			tsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName), Weight(weightedCluster.Weight))
			clusterConfigs := otp.newClusterConfigs(ClusterName(weightedCluster.ClusterName.String()))
			for _, serverName := range trafficMatch.ServerNames {
				address := Address(serverName)
				port := Port(trafficMatch.DestinationPort)
				weight := Weight(constants.ClusterWeightAcceptAll)
				clusterConfigs.addWeightedEndpoint(address, port, weight)
			}
		} else if destinationProtocol == constants.ProtocolTCP ||
			destinationProtocol == constants.ProtocolTCPServerFirst {
			tsrr := tm.newTCPServiceRouteRules()
			tsrr.setAllowedEgressTraffic(true)
			tsrr.setEgressForwardGateway(trafficMatch.EgressGateWay)
		}
	}

	return dependClusters
}

func getEgressClusterDestinationSpec(meshCatalog catalog.MeshCataloger, egressPolicy *trafficpolicy.EgressTrafficPolicy, trafficMatch *trafficpolicy.TrafficMatch) *DestinationSecuritySpec {
	var destinationSpec *DestinationSecuritySpec
	if clusterConfig := getEgressClusterConfigs(egressPolicy.ClustersConfigs, service.ClusterName(trafficMatch.Cluster)); clusterConfig != nil {
		if clusterConfig.SourceMTLS != nil {
			destinationSpec = new(DestinationSecuritySpec)
			destinationSpec.SourceCert = new(Certificate)
			fsmIssued := strings.EqualFold(`fsm`, clusterConfig.SourceMTLS.Issuer)
			destinationSpec.SourceCert.FsmIssued = &fsmIssued
			if !fsmIssued && clusterConfig.SourceMTLS.Cert != nil {
				secretReference := corev1.SecretReference{
					Name:      clusterConfig.SourceMTLS.Cert.Secret.Name,
					Namespace: clusterConfig.SourceMTLS.Cert.Secret.Namespace,
				}
				if secret, err := meshCatalog.GetEgressSourceSecret(secretReference); err == nil {
					destinationSpec.SourceCert.SubjectAltNames = clusterConfig.SourceMTLS.Cert.SubjectAltNames
					destinationSpec.SourceCert.Expiration = clusterConfig.SourceMTLS.Cert.Expiration
					if caCrt, ok := secret.Data["ca.crt"]; ok {
						destinationSpec.SourceCert.IssuingCA = string(caCrt)
					}
					if tlsCrt, ok := secret.Data["tls.crt"]; ok {
						destinationSpec.SourceCert.CertChain = string(tlsCrt)
					}
					if tlsKey, ok := secret.Data["tls.key"]; ok {
						destinationSpec.SourceCert.PrivateKey = string(tlsKey)
					}
				} else {
					log.Error().Err(err)
				}
			}
		}
	}
	return destinationSpec
}

func generatePipyOutboundTrafficBalancePolicy(meshCatalog catalog.MeshCataloger, cfg configurator.Configurator,
	proxy *pipy.Proxy, pipyConf *PipyConf, outboundPolicy *trafficpolicy.OutboundMeshTrafficPolicy,
	dependClusters map[service.ClusterName]*WeightedCluster) bool {
	ready := true
	viaGateway := cfg.GetMeshConfig().Spec.Connector.ViaGateway
	otp := pipyConf.newOutboundTrafficPolicy()
	clustersConfigsMap := make(map[string][]*trafficpolicy.MeshClusterConfig)
	if len(outboundPolicy.ClustersConfigs) > 0 {
		for _, clustersConfig := range outboundPolicy.ClustersConfigs {
			items := clustersConfigsMap[clustersConfig.Service.SidecarClusterName()]
			items = append(items, clustersConfig)
			clustersConfigsMap[clustersConfig.Service.SidecarClusterName()] = items

			items = clustersConfigsMap[clustersConfig.Service.SidecarLocalClusterName()]
			items = append(items, clustersConfig)
			clustersConfigsMap[clustersConfig.Service.SidecarLocalClusterName()] = items
		}
	}
	for _, cluster := range dependClusters {
		meshClusterConfigs := clustersConfigsMap[string(cluster.ClusterName)]
		if len(meshClusterConfigs) == 0 {
			ready = false
			continue
		}
		for _, clusterConfig := range meshClusterConfigs {
			clusterConfigs := otp.newClusterConfigs(ClusterName(cluster.ClusterName.String()))
			upstreamEndpoints := getUpstreamEndpoints(meshCatalog, proxy.Identity, service.ClusterName(clusterConfig.Service.ClusterName()))
			if len(upstreamEndpoints) == 0 {
				ready = false
				continue
			}
			for _, upstreamEndpoint := range upstreamEndpoints {
				address := Address(upstreamEndpoint.IP.String())
				port := Port(clusterConfig.Service.Port)
				if len(upstreamEndpoint.ClusterKey) > 0 {
					if targetPort := Port(clusterConfig.Service.TargetPort); targetPort > 0 {
						port = targetPort
					}
				}
				weight := Weight(upstreamEndpoint.Weight)
				viaGw := ""
				if strings.EqualFold(constants.ProtocolHTTP, upstreamEndpoint.AppProtocol) {
					viaGw = upstreamEndpoint.ViaGatewayHTTP
				} else if strings.EqualFold(constants.ProtocolGRPC, upstreamEndpoint.AppProtocol) {
					viaGw = upstreamEndpoint.ViaGatewayGRPC
				}
				if len(upstreamEndpoint.ViaGatewayMode) > 0 {
					if upstreamEndpoint.WithGateway {
						if upstreamEndpoint.WithMultiGateways {
							viaGw = generatePipyViaGateway(upstreamEndpoint.AppProtocol, upstreamEndpoint.ClusterID, proxy, &viaGateway)
						}
					} else {
						port = Port(upstreamEndpoint.Port)
					}
				}
				clusterConfigs.addWeightedZoneEndpoint(address, port, weight, upstreamEndpoint.ClusterKey, upstreamEndpoint.LBType, upstreamEndpoint.Path, viaGw)
			}
			if clusterConfig.UpstreamTrafficSetting != nil {
				if clusterConfig.UpstreamTrafficSetting.Spec.ConnectionSettings != nil {
					clusterConfigs.setConnectionSettings(clusterConfig.UpstreamTrafficSetting.Spec.ConnectionSettings)
				}
			}
			if cluster.RetryPolicy != nil {
				clusterConfigs.setRetryPolicy(cluster.RetryPolicy)
			}
		}
	}
	return ready
}

func generatePipyViaGateway(appProtocol, clusterID string, proxy *pipy.Proxy, viaGateway *configv1alpha3.ConnectorGatewaySpec) string {
	viaGw := ""
	if len(appProtocol) > 0 && !strings.EqualFold(proxy.ClusterID, clusterID) {
		if len(proxy.ClusterID) == 0 { // k8s -> fgw(EgressIP:EgressPort) -> others
			if len(viaGateway.ClusterIP) > 0 && viaGateway.EgressHTTPPort > 0 &&
				strings.EqualFold(constants.ProtocolHTTP, appProtocol) {
				viaGw = fmt.Sprintf("%s:%d", viaGateway.ClusterIP, viaGateway.EgressHTTPPort)
			}
			if len(viaGateway.ClusterIP) > 0 && viaGateway.EgressGRPCPort > 0 &&
				strings.EqualFold(constants.ProtocolGRPC, appProtocol) {
				viaGw = fmt.Sprintf("%s:%d", viaGateway.ClusterIP, viaGateway.EgressGRPCPort)
			}
		} else {
			if len(clusterID) == 0 { // others -> fgw(IngressIP:IngressPort) -> k8s
				if len(viaGateway.IngressAddr) > 0 && viaGateway.IngressHTTPPort > 0 &&
					strings.EqualFold(constants.ProtocolHTTP, appProtocol) {
					viaGw = fmt.Sprintf("%s:%d", viaGateway.IngressAddr, viaGateway.IngressHTTPPort)
				}
				if len(viaGateway.IngressAddr) > 0 && viaGateway.IngressGRPCPort > 0 &&
					strings.EqualFold(constants.ProtocolGRPC, appProtocol) {
					viaGw = fmt.Sprintf("%s:%d", viaGateway.IngressAddr, viaGateway.IngressGRPCPort)
				}
			} else { // others -> fgw(IngressIP:EgressPort) -> others
				if len(viaGateway.IngressAddr) > 0 && viaGateway.EgressHTTPPort > 0 &&
					strings.EqualFold(constants.ProtocolHTTP, appProtocol) {
					viaGw = fmt.Sprintf("%s:%d", viaGateway.IngressAddr, viaGateway.EgressHTTPPort)
				}
				if len(viaGateway.IngressAddr) > 0 && viaGateway.EgressGRPCPort > 0 &&
					strings.EqualFold(constants.ProtocolGRPC, appProtocol) {
					viaGw = fmt.Sprintf("%s:%d", viaGateway.IngressAddr, viaGateway.EgressGRPCPort)
				}
			}
		}
	}
	return viaGw
}

func generatePipyIngressTrafficRoutePolicy(_ catalog.MeshCataloger, pipyConf *PipyConf, ingressPolicy *trafficpolicy.IngressTrafficPolicy) {
	if len(ingressPolicy.TrafficMatches) == 0 {
		return
	}

	if pipyConf.Inbound == nil {
		return
	}

	if len(pipyConf.Inbound.ClustersConfigs) == 0 {
		return
	}

	itp := pipyConf.newInboundTrafficPolicy()

	for _, trafficMatch := range ingressPolicy.TrafficMatches {
		tm := itp.getTrafficMatch(Port(trafficMatch.Port))
		if tm == nil {
			tm = itp.newTrafficMatch(Port(trafficMatch.Port))
			protocol := strings.ToLower(trafficMatch.Protocol)
			tm.setProtocol(Protocol(protocol))
			tm.setPort(Port(trafficMatch.Port))
			tm.setTCPServiceRateLimit(trafficMatch.RateLimit)
		}

		var securitySpec *SourceSecuritySpec
		if trafficMatch.TLS != nil {
			securitySpec = &SourceSecuritySpec{
				MTLS:                     true,
				SkipClientCertValidation: trafficMatch.TLS.SkipClientCertValidation,
			}
		}

		for _, ipRange := range trafficMatch.SourceIPRanges {
			tm.addSourceIPRange(SourceIPRange(ipRange), securitySpec)
		}

		var authenticatedPrincipals []string
		protocol := strings.ToLower(trafficMatch.Protocol)
		if protocol != constants.ProtocolHTTP && protocol != constants.ProtocolHTTPS && protocol != constants.ProtocolGRPC {
			continue
		}
		for _, httpRouteConfig := range ingressPolicy.HTTPRoutePolicies {
			if len(httpRouteConfig.Rules) == 0 {
				continue
			}
			for _, hostname := range httpRouteConfig.Hostnames {
				ruleRef := &HTTPRouteRuleRef{RuleName: HTTPRouteRuleName(hostname)}
				tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleRef)

				hsrrs := tm.newHTTPServiceRouteRules(ruleRef)
				hsrrs.setHTTPServiceRateLimit(trafficMatch.RateLimit)
				hsrrs.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
				for _, rule := range httpRouteConfig.Rules {
					if len(rule.Route.HTTPRouteMatch.Path) == 0 {
						continue
					}

					route := rule.Route
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					if hsrr, duplicate := hsrrs.newHTTPServiceRouteRule(httpMatch); !duplicate {
						hsrr.setRateLimit(rule.Route.RateLimit)
						for routeCluster := range rule.Route.WeightedClusters.Iter() {
							weightedCluster := routeCluster.(service.WeightedCluster)
							hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName),
								Weight(weightedCluster.Weight))
						}
					}

					for allowedPrincipal := range rule.AllowedPrincipals.Iter() {
						servicePrincipal := allowedPrincipal.(string)
						authenticatedPrincipals = append(authenticatedPrincipals, servicePrincipal)
					}
				}
			}
		}

		if securitySpec != nil {
			securitySpec.AuthenticatedPrincipals = authenticatedPrincipals
		}
	}
}

func generatePipyEgressTrafficForwardPolicy(_ catalog.MeshCataloger, pipyConf *PipyConf, egressGatewayPolicy *trafficpolicy.EgressGatewayPolicy) bool {
	if egressGatewayPolicy == nil || (egressGatewayPolicy.Global == nil && (egressGatewayPolicy.Rules == nil || len(egressGatewayPolicy.Rules) == 0)) {
		return true
	}

	success := true
	ftp := pipyConf.newForwardTrafficPolicy()
	if egressGatewayPolicy.Global != nil {
		forwardMatch := ftp.newForwardMatch("*")
		for _, gateway := range egressGatewayPolicy.Global {
			clusterName := fmt.Sprintf("%s.%s", gateway.Service, gateway.Namespace)
			if gateway.Weight != nil {
				forwardMatch[ClusterName(clusterName)] = Weight(*gateway.Weight)
			} else {
				forwardMatch[ClusterName(clusterName)] = Weight(0)
			}
			if len(gateway.Endpoints) > 0 {
				clusterConfigs := ftp.newEgressGateway(ClusterName(clusterName), gateway.Mode)
				for _, endPeer := range gateway.Endpoints {
					address := Address(endPeer.IP.String())
					port := Port(endPeer.Port)
					weight := Weight(0)
					clusterConfigs.addWeightedEndpoint(address, port, weight)
				}
			}
		}
	}
	if egressGatewayPolicy.Rules != nil {
		for index, rule := range egressGatewayPolicy.Rules {
			ruleName := fmt.Sprintf("%s.%s.%d", rule.Namespace, rule.Name, index)
			forwardMatch := ftp.newForwardMatch(ruleName)
			for _, gateway := range rule.EgressGateways {
				clusterName := fmt.Sprintf("%s.%s", gateway.Service, gateway.Namespace)
				if gateway.Weight != nil {
					forwardMatch[ClusterName(clusterName)] = Weight(*gateway.Weight)
				} else {
					forwardMatch[ClusterName(clusterName)] = constants.ClusterWeightAcceptAll
				}
				if len(gateway.Endpoints) > 0 {
					clusterConfigs := ftp.newEgressGateway(ClusterName(clusterName), gateway.Mode)
					for _, endPeer := range gateway.Endpoints {
						address := Address(endPeer.IP.String())
						port := Port(endPeer.Port)
						weight := Weight(0)
						clusterConfigs.addWeightedEndpoint(address, port, weight)
					}
				} else {
					success = false
				}
			}
		}
	}

	return success
}

func generatePipyAccessControlTrafficRoutePolicy(_ catalog.MeshCataloger, pipyConf *PipyConf, aclPolicy *trafficpolicy.AccessControlTrafficPolicy) {
	if len(aclPolicy.TrafficMatches) == 0 {
		return
	}

	if pipyConf.Inbound == nil {
		return
	}

	if len(pipyConf.Inbound.ClustersConfigs) == 0 {
		return
	}

	itp := pipyConf.newInboundTrafficPolicy()

	for _, trafficMatch := range aclPolicy.TrafficMatches {
		tm := itp.getTrafficMatch(Port(trafficMatch.Port))
		if tm == nil {
			tm = itp.newTrafficMatch(Port(trafficMatch.Port))
			protocol := strings.ToLower(trafficMatch.Protocol)
			tm.setProtocol(Protocol(protocol))
			tm.setPort(Port(trafficMatch.Port))
			tm.setTCPServiceRateLimit(trafficMatch.RateLimit)
		}

		var securitySpec *SourceSecuritySpec
		if trafficMatch.TLS != nil {
			securitySpec = &SourceSecuritySpec{
				MTLS:                     true,
				SkipClientCertValidation: trafficMatch.TLS.SkipClientCertValidation,
			}
		}

		for _, ipRange := range trafficMatch.SourceIPRanges {
			tm.addSourceIPRange(SourceIPRange(ipRange), securitySpec)
		}

		var authenticatedPrincipals []string
		protocol := strings.ToLower(trafficMatch.Protocol)
		if protocol != constants.ProtocolHTTP && protocol != constants.ProtocolHTTPS && protocol != constants.ProtocolGRPC {
			continue
		}
		for _, httpRouteConfig := range aclPolicy.HTTPRoutePolicies {
			if len(httpRouteConfig.Rules) == 0 {
				continue
			}
			for _, hostname := range httpRouteConfig.Hostnames {
				ruleRef := &HTTPRouteRuleRef{RuleName: HTTPRouteRuleName(hostname)}
				tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleRef)

				hsrrs := tm.newHTTPServiceRouteRules(ruleRef)
				hsrrs.setHTTPServiceRateLimit(trafficMatch.RateLimit)
				hsrrs.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
				for _, rule := range httpRouteConfig.Rules {
					if len(rule.Route.HTTPRouteMatch.Path) == 0 {
						continue
					}

					route := rule.Route
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					if hsrr, duplicate := hsrrs.newHTTPServiceRouteRule(httpMatch); !duplicate {
						hsrr.setRateLimit(rule.Route.RateLimit)
						for routeCluster := range rule.Route.WeightedClusters.Iter() {
							weightedCluster := routeCluster.(service.WeightedCluster)
							hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName),
								Weight(weightedCluster.Weight))
						}
					}

					for allowedPrincipal := range rule.AllowedPrincipals.Iter() {
						servicePrincipal := allowedPrincipal.(string)
						authenticatedPrincipals = append(authenticatedPrincipals, servicePrincipal)
					}
				}
			}
		}

		if securitySpec != nil {
			securitySpec.AuthenticatedPrincipals = authenticatedPrincipals
		}
	}
}

func generatePipyServiceExportTrafficRoutePolicy(_ catalog.MeshCataloger, pipyConf *PipyConf, expPolicy *trafficpolicy.ServiceExportTrafficPolicy) {
	if len(expPolicy.TrafficMatches) == 0 {
		return
	}

	if pipyConf.Inbound == nil {
		return
	}

	if len(pipyConf.Inbound.ClustersConfigs) == 0 {
		return
	}

	itp := pipyConf.newInboundTrafficPolicy()

	for _, trafficMatch := range expPolicy.TrafficMatches {
		tm := itp.getTrafficMatch(Port(trafficMatch.Port))
		if tm == nil {
			tm = itp.newTrafficMatch(Port(trafficMatch.Port))
			protocol := strings.ToLower(trafficMatch.Protocol)
			tm.setProtocol(Protocol(protocol))
			tm.setPort(Port(trafficMatch.Port))
		}

		var securitySpec *SourceSecuritySpec
		if trafficMatch.TLS != nil {
			securitySpec = &SourceSecuritySpec{
				MTLS:                     true,
				SkipClientCertValidation: trafficMatch.TLS.SkipClientCertValidation,
			}
		}

		for _, ipRange := range trafficMatch.SourceIPRanges {
			tm.addSourceIPRange(SourceIPRange(ipRange), securitySpec)
		}

		var authenticatedPrincipals []string
		protocol := strings.ToLower(trafficMatch.Protocol)
		if protocol != constants.ProtocolHTTP && protocol != constants.ProtocolHTTPS && protocol != constants.ProtocolGRPC {
			continue
		}
		for _, httpRouteConfig := range expPolicy.HTTPRoutePolicies {
			if len(httpRouteConfig.Rules) == 0 {
				continue
			}
			for _, hostname := range httpRouteConfig.Hostnames {
				ruleRef := &HTTPRouteRuleRef{RuleName: HTTPRouteRuleName(hostname)}
				tm.addHTTPHostPort2Service(HTTPHostPort(hostname), ruleRef)

				hsrrs := tm.newHTTPServiceRouteRules(ruleRef)
				hsrrs.setPlugins(pipyConf.getTrafficMatchPluginConfigs(trafficMatch.Name))
				for _, rule := range httpRouteConfig.Rules {
					if len(rule.Route.HTTPRouteMatch.Path) == 0 {
						continue
					}

					route := rule.Route
					httpMatch := new(HTTPMatchRule)
					httpMatch.Path = URIPathValue(route.HTTPRouteMatch.Path)
					httpMatch.Type = matchType(route.HTTPRouteMatch.PathMatchType)
					if len(httpMatch.Type) == 0 {
						httpMatch.Type = PathMatchRegex
					}
					if len(httpMatch.Path) == 0 {
						httpMatch.Path = constants.RegexMatchAll
					}
					for k, v := range route.HTTPRouteMatch.Headers {
						httpMatch.addHeaderMatch(Header(k), HeaderRegexp(v))
					}
					if len(route.HTTPRouteMatch.Methods) == 0 {
						httpMatch.addMethodMatch("*")
					} else {
						for _, method := range route.HTTPRouteMatch.Methods {
							httpMatch.addMethodMatch(Method(method))
						}
					}

					if hsrr, duplicate := hsrrs.newHTTPServiceRouteRule(httpMatch); !duplicate {
						for routeCluster := range rule.Route.WeightedClusters.Iter() {
							weightedCluster := routeCluster.(service.WeightedCluster)
							hsrr.addWeightedCluster(ClusterName(weightedCluster.ClusterName),
								Weight(weightedCluster.Weight))
						}
					}

					for allowedPrincipal := range rule.AllowedPrincipals.Iter() {
						servicePrincipal := allowedPrincipal.(string)
						authenticatedPrincipals = append(authenticatedPrincipals, servicePrincipal)
					}
				}
			}
		}

		if securitySpec != nil {
			securitySpec.AuthenticatedPrincipals = authenticatedPrincipals
		}
	}
}

func generatePipyEgressTrafficBalancePolicy(meshCatalog catalog.MeshCataloger, proxy *pipy.Proxy, pipyConf *PipyConf, egressPolicy *trafficpolicy.EgressTrafficPolicy, dependClusters map[service.ClusterName]*WeightedCluster) bool {
	ready := true
	otp := pipyConf.newOutboundTrafficPolicy()
	for _, cluster := range dependClusters {
		clusterConfig := getEgressClusterConfigs(egressPolicy.ClustersConfigs, cluster.ClusterName)
		if clusterConfig == nil {
			ready = false
			continue
		}
		clusterConfigs := otp.newClusterConfigs(ClusterName(cluster.ClusterName.String()))
		address := Address(clusterConfig.Name)
		port := Port(clusterConfig.Port)
		weight := Weight(constants.ClusterWeightAcceptAll)
		clusterConfigs.addWeightedEndpoint(address, port, weight)
		if clusterConfig.UpstreamTrafficSetting != nil {
			clusterConfigs.setConnectionSettings(clusterConfig.UpstreamTrafficSetting.Spec.ConnectionSettings)
		}
		if clusterConfig.SourceMTLS != nil {
			clusterConfigs.SourceCert = new(Certificate)
			fsmIssued := strings.EqualFold(`fsm`, clusterConfig.SourceMTLS.Issuer)
			clusterConfigs.SourceCert.FsmIssued = &fsmIssued
			if !fsmIssued && clusterConfig.SourceMTLS.Cert != nil {
				secretReference := corev1.SecretReference{
					Name:      clusterConfig.SourceMTLS.Cert.Secret.Name,
					Namespace: clusterConfig.SourceMTLS.Cert.Secret.Namespace,
				}
				if secret, err := meshCatalog.GetEgressSourceSecret(secretReference); err == nil {
					clusterConfigs.SourceCert.SubjectAltNames = clusterConfig.SourceMTLS.Cert.SubjectAltNames
					clusterConfigs.SourceCert.Expiration = clusterConfig.SourceMTLS.Cert.Expiration
					if caCrt, ok := secret.Data["ca.crt"]; ok {
						clusterConfigs.SourceCert.IssuingCA = string(caCrt)
					}
					if tlsCrt, ok := secret.Data["tls.crt"]; ok {
						clusterConfigs.SourceCert.CertChain = string(tlsCrt)
					}
					if tlsKey, ok := secret.Data["tls.key"]; ok {
						clusterConfigs.SourceCert.PrivateKey = string(tlsKey)
					}
				} else {
					log.Error().Err(err)
				}
			}
		}
		if cluster.RetryPolicy != nil {
			clusterConfigs.setRetryPolicy(cluster.RetryPolicy)
		} else if upstreamSvc, err := hostToMeshSvc(cluster.ClusterName.String()); err == nil {
			if retryPolicy := meshCatalog.GetRetryPolicy(proxy.Identity, upstreamSvc); retryPolicy != nil {
				clusterConfigs.setRetryPolicy(retryPolicy)
			}
		}
	}
	return ready
}

func getInboundHTTPRouteConfigs(httpRouteConfigsPerPort map[int][]*trafficpolicy.InboundTrafficPolicy,
	targetPort int, upstreamPolicyName string) *trafficpolicy.InboundTrafficPolicy {
	if httpRouteConfigs, ok := httpRouteConfigsPerPort[targetPort]; ok {
		for _, httpRouteConfig := range httpRouteConfigs {
			if httpRouteConfig.Name == upstreamPolicyName {
				return httpRouteConfig
			}
		}
	}
	return nil
}

func getOutboundHTTPRouteConfigs(httpRouteConfigsPerPort map[int][]*trafficpolicy.OutboundTrafficPolicy,
	targetPort int, upstreamPolicyName string, weightedClusters []service.WeightedCluster) []*trafficpolicy.OutboundTrafficPolicy {
	var outboundTrafficPolicies []*trafficpolicy.OutboundTrafficPolicy
	if trafficPolicies, ok := httpRouteConfigsPerPort[targetPort]; ok {
		for _, trafficPolicy := range trafficPolicies {
			if trafficPolicy.Name == upstreamPolicyName {
				for _, route := range trafficPolicy.Routes {
					if arrayEqual(weightedClusters, route.WeightedClusters) {
						outboundTrafficPolicies = append(outboundTrafficPolicies, trafficPolicy)
						break
					}
				}
			}
		}
	}
	return outboundTrafficPolicies
}

func getEgressHTTPRouteConfigs(httpRouteConfigsPerPort map[int][]*trafficpolicy.EgressHTTPRouteConfig,
	targetPort int) []*trafficpolicy.EgressHTTPRouteConfig {
	if httpRouteConfigs, ok := httpRouteConfigsPerPort[targetPort]; ok {
		return httpRouteConfigs
	}
	return nil
}

func trafficMatchToMeshSvc(trafficMatch *trafficpolicy.TrafficMatch) *service.MeshService {
	chunks := strings.Split(trafficMatch.Name, "_")
	if len(chunks) != 5 {
		log.Error().Msgf("Invalid traffic match name. Expected: xxx_<namespace>/<name>_<port>_<protocol>_<attached>, got: %s",
			trafficMatch.Name)
		return nil
	}

	namespacedName, err := k8s.NamespacedNameFrom(chunks[1])
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving NamespacedName from TrafficMatch")
		return nil
	}

	attachedNamespace := chunks[4]
	return &service.MeshService{
		Namespace:              namespacedName.Namespace,
		Name:                   namespacedName.Name,
		Protocol:               strings.ToLower(trafficMatch.DestinationProtocol),
		TargetPort:             uint16(trafficMatch.DestinationPort),
		CloudAttachedNamespace: attachedNamespace,
	}
}

func getMeshClusterConfigs(clustersConfigs []*trafficpolicy.MeshClusterConfig,
	clusterName service.ClusterName) []*trafficpolicy.MeshClusterConfig {
	if len(clustersConfigs) == 0 {
		return nil
	}

	var items []*trafficpolicy.MeshClusterConfig

	for _, clustersConfig := range clustersConfigs {
		if clusterName.String() == clustersConfig.Service.SidecarClusterName() {
			items = append(items, clustersConfig)
		}
		if clusterName.String() == clustersConfig.Service.SidecarLocalClusterName() {
			items = append(items, clustersConfig)
		}
	}

	return items
}

func getEgressClusterConfigs(clustersConfigs []*trafficpolicy.EgressClusterConfig,
	clusterName service.ClusterName) *trafficpolicy.EgressClusterConfig {
	if len(clustersConfigs) == 0 {
		return nil
	}

	for _, clustersConfig := range clustersConfigs {
		if clusterName.String() == clustersConfig.Name {
			return clustersConfig
		}
	}

	return nil
}

func getUpstreamEndpoints(meshCatalog catalog.MeshCataloger, proxyIdentity identity.ServiceIdentity,
	clusterName service.ClusterName) []endpoint.Endpoint {
	if dstSvc, err := clusterToMeshSvc(clusterName.String()); err == nil {
		return meshCatalog.ListAllowedUpstreamEndpointsForService(proxyIdentity, dstSvc)
	}
	return nil
}

// clusterToMeshSvc returns the MeshService associated with the given cluster name
func clusterToMeshSvc(cluster string) (service.MeshService, error) {
	splitFunc := func(r rune) bool {
		return r == '/' || r == '|'
	}

	chunks := strings.FieldsFunc(cluster, splitFunc)
	if len(chunks) != 3 {
		return service.MeshService{},
			errors.Errorf("Invalid cluster name. Expected: <namespace>/<name>|<port>, got: %s", cluster)
	}

	port, err := strconv.ParseUint(chunks[2], 10, 16)
	if err != nil {
		return service.MeshService{}, errors.Errorf("Invalid cluster port %s, expected int value: %s", chunks[2], err)
	}

	return service.MeshService{
		Namespace: chunks[0],
		Name:      chunks[1],
		// The port always maps to MeshServer.TargetPort and not MeshService.Port because
		// endpoints of a service are derived from it's TargetPort and not Port.
		TargetPort: uint16(port),
	}, nil
}

// hostToMeshSvc returns the MeshService associated with the given host name
func hostToMeshSvc(cluster string) (service.MeshService, error) {
	splitFunc := func(r rune) bool {
		return r == '.' || r == ':'
	}

	chunks := strings.FieldsFunc(cluster, splitFunc)
	if len(chunks) > 4 && strings.EqualFold("svc", chunks[3]) {
		return service.MeshService{},
			errors.Errorf("Invalid host. Expected: <name>.<namespace>.svc.trustdomain:<port>, got: %s", cluster)
	}

	port, err := strconv.ParseUint(chunks[len(chunks)-1], 10, 16)
	if err != nil {
		return service.MeshService{}, errors.Errorf("Invalid cluster port %s, expected int value: %s", chunks[len(chunks)-1], err)
	}

	return service.MeshService{
		Namespace: chunks[1],
		Name:      chunks[0],
		// The port always maps to MeshServer.TargetPort and not MeshService.Port because
		// endpoints of a service are derived from it's TargetPort and not Port.
		TargetPort: uint16(port),
	}, nil
}

func getEndpointsForProxyIdentity(meshCatalog catalog.MeshCataloger, proxyIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	if mc, ok := meshCatalog.(*catalog.MeshCatalog); ok {
		return mc.ListEndpointsForServiceIdentity(proxyIdentity)
	}
	return nil
}

func arrayEqual(a []service.WeightedCluster, set mapset.Set) bool {
	var b []service.WeightedCluster
	for e := range set.Iter() {
		if o, ok := e.(service.WeightedCluster); ok {
			b = append(b, o)
		}
	}
	if len(a) == len(b) {
		for _, ca := range a {
			caEqualb := false
			for _, cb := range b {
				if ca.ClusterName == cb.ClusterName && ca.Weight == cb.Weight {
					caEqualb = true
					break
				}
			}
			if !caEqualb {
				return false
			}
		}
		for _, cb := range b {
			cbEquala := false
			for _, ca := range a {
				if cb.ClusterName == ca.ClusterName && cb.Weight == ca.Weight {
					cbEquala = true
					break
				}
			}
			if !cbEquala {
				return false
			}
		}
		return true
	}
	return false
}

func Hash(bytes []byte) uint64 {
	if hashCode, err := utils.HashFromString(string(bytes)); err == nil {
		return hashCode
	}
	return uint64(time.Now().Nanosecond())
}

func matchType(matchType trafficpolicy.PathMatchType) URIMatchType {
	switch matchType {
	case trafficpolicy.PathMatchExact:
		return PathMatchExact
	case trafficpolicy.PathMatchPrefix:
		return PathMatchPrefix
	default:
		return PathMatchRegex
	}
}
