package repo

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	multiclusterv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/registry"
	"github.com/flomesh-io/fsm/pkg/utils/cidr"
)

var (
	addrWithPort, _ = regexp.Compile(`:\d+$`)
	prettyConfig    func() bool
)

func (plugin *Pluggable) setPlugins(plugins map[string]*runtime.RawExtension) {
	plugin.Plugins = plugins
}

func (p *PipyConf) setSidecarLogLevel(sidecarLogLevel string) (update bool) {
	if update = !strings.EqualFold(p.Spec.SidecarLogLevel, sidecarLogLevel); update {
		p.Spec.SidecarLogLevel = sidecarLogLevel
	}
	return
}

func (p *PipyConf) setSidecarTimeout(sidecarTimeout int) (update bool) {
	if update = p.Spec.SidecarTimeout != sidecarTimeout; update {
		p.Spec.SidecarTimeout = sidecarTimeout
	}
	return
}

func (p *PipyConf) setLocalDNSProxy(enable bool, conf *configurator.Configurator) {
	if enable {
		p.Spec.LocalDNSProxy = new(LocalDNSProxy)
		primary := (*conf).GetLocalDNSProxyPrimaryUpstream()
		secondary := (*conf).GetLocalDNSProxySecondaryUpstream()
		primary = strings.TrimSpace(primary)
		secondary = strings.TrimSpace(secondary)
		if len(primary) > 0 || len(secondary) > 0 {
			p.Spec.LocalDNSProxy.UpstreamDNSServers = new(UpstreamDNSServers)
			if len(primary) > 0 {
				p.Spec.LocalDNSProxy.UpstreamDNSServers.Primary = &primary
			}
			if len(secondary) > 0 {
				p.Spec.LocalDNSProxy.UpstreamDNSServers.Secondary = &secondary
			}
		}
	} else {
		p.Spec.LocalDNSProxy = nil
	}
}

func (p *PipyConf) setObservabilityTracing(enable bool, conf *configurator.Configurator) {
	if enable {
		p.Spec.Observability.Tracing = &TracingSpec{
			Address:         fmt.Sprintf("%s:%d", (*conf).GetTracingHost(), (*conf).GetTracingPort()),
			Endpoint:        (*conf).GetTracingEndpoint(),
			SampledFraction: fmt.Sprintf("%0.2f", (*conf).GetTracingSampledFraction()),
		}
	} else {
		p.Spec.Observability.Tracing = nil
	}
}

func (p *PipyConf) setObservabilityRemoteLogging(enable bool, conf *configurator.Configurator) {
	if enable {
		p.Spec.Observability.RemoteLogging = &RemoteLoggingSpec{
			Level:           (*conf).GetRemoteLoggingLevel(),
			Address:         fmt.Sprintf("%s:%d", (*conf).GetRemoteLoggingHost(), (*conf).GetRemoteLoggingPort()),
			Endpoint:        (*conf).GetRemoteLoggingEndpoint(),
			Authorization:   (*conf).GetRemoteLoggingAuthorization(),
			SampledFraction: fmt.Sprintf("%0.2f", (*conf).GetRemoteLoggingSampledFraction()),
		}
	} else {
		p.Spec.Observability.RemoteLogging = nil
	}
}

func (p *PipyConf) setEnableSidecarActiveHealthChecks(enableSidecarActiveHealthChecks bool) (update bool) {
	if update = p.Spec.FeatureFlags.EnableSidecarActiveHealthChecks != enableSidecarActiveHealthChecks; update {
		p.Spec.FeatureFlags.EnableSidecarActiveHealthChecks = enableSidecarActiveHealthChecks
	}
	return
}

func (p *PipyConf) setEnableAutoDefaultRoute(enableAutoDefaultRoute bool) (update bool) {
	if update = p.Spec.FeatureFlags.EnableAutoDefaultRoute != enableAutoDefaultRoute; update {
		p.Spec.FeatureFlags.EnableAutoDefaultRoute = enableAutoDefaultRoute
	}
	return
}

func (p *PipyConf) setEnableEgress(enableEgress bool) (update bool) {
	if update = p.Spec.Traffic.EnableEgress != enableEgress; update {
		p.Spec.Traffic.EnableEgress = enableEgress
	}
	return
}

func (p *PipyConf) setHTTP1PerRequestLoadBalancing(http1PerRequestLoadBalancing bool) (update bool) {
	if update = p.Spec.Traffic.HTTP1PerRequestLoadBalancing != http1PerRequestLoadBalancing; update {
		p.Spec.Traffic.HTTP1PerRequestLoadBalancing = http1PerRequestLoadBalancing
	}
	return
}

func (p *PipyConf) setHTTP2PerRequestLoadBalancing(http2PerRequestLoadBalancing bool) (update bool) {
	if update = p.Spec.Traffic.HTTP2PerRequestLoadBalancing != http2PerRequestLoadBalancing; update {
		p.Spec.Traffic.HTTP2PerRequestLoadBalancing = http2PerRequestLoadBalancing
	}
	return
}

func (p *PipyConf) setEnablePermissiveTrafficPolicyMode(enablePermissiveTrafficPolicyMode bool) (update bool) {
	if update = p.Spec.Traffic.enablePermissiveTrafficPolicyMode != enablePermissiveTrafficPolicyMode; update {
		p.Spec.Traffic.enablePermissiveTrafficPolicyMode = enablePermissiveTrafficPolicyMode
	}
	return
}

func (p *PipyConf) isPermissiveTrafficPolicyMode() bool {
	return p.Spec.Traffic.enablePermissiveTrafficPolicyMode
}

func (p *PipyConf) newInboundTrafficPolicy() *InboundTrafficPolicy {
	if p.Inbound == nil {
		p.Inbound = new(InboundTrafficPolicy)
	}
	return p.Inbound
}

func (p *PipyConf) newOutboundTrafficPolicy() *OutboundTrafficPolicy {
	if p.Outbound == nil {
		p.Outbound = new(OutboundTrafficPolicy)
	}
	return p.Outbound
}

func (p *PipyConf) newForwardTrafficPolicy() *ForwardTrafficPolicy {
	if p.Forward == nil {
		p.Forward = new(ForwardTrafficPolicy)
	}
	return p.Forward
}

func (p *PipyConf) rebalancedOutboundClusters() {
	if p.Outbound == nil {
		return
	}
	if p.Outbound.ClustersConfigs == nil || len(p.Outbound.ClustersConfigs) == 0 {
		return
	}
	for _, clusterConfigs := range p.Outbound.ClustersConfigs {
		weightedEndpoints := clusterConfigs.Endpoints
		if weightedEndpoints == nil || len(*weightedEndpoints) == 0 {
			continue
		}
		hasLocalEndpoints := false
		for _, wze := range *weightedEndpoints {
			if len(wze.Cluster) == 0 {
				hasLocalEndpoints = true
				break
			}
		}
		for _, wze := range *weightedEndpoints {
			if len(wze.Cluster) > 0 {
				if multiclusterv1alpha1.FailOverLbType == multiclusterv1alpha1.LoadBalancerType(wze.LBType) {
					if hasLocalEndpoints {
						wze.Weight = constants.ClusterWeightFailOver
					} else {
						wze.Weight = constants.ClusterWeightAcceptAll
					}
				} else if multiclusterv1alpha1.ActiveActiveLbType == multiclusterv1alpha1.LoadBalancerType(wze.LBType) {
					if wze.Weight == 0 {
						wze.Weight = constants.ClusterWeightAcceptAll
					}
				}
			} else {
				if wze.Weight == 0 {
					wze.Weight = constants.ClusterWeightAcceptAll
				}
			}
		}
	}
}

func (p *PipyConf) rebalancedForwardClusters() {
	if p.Forward == nil {
		return
	}
	if p.Forward.ForwardMatches != nil && len(p.Forward.ForwardMatches) > 0 {
		for _, weightedEndpoints := range p.Forward.ForwardMatches {
			if len(weightedEndpoints) == 0 {
				continue
			}
			for upstreamEndpoint, weight := range weightedEndpoints {
				if weight == 0 {
					(weightedEndpoints)[upstreamEndpoint] = constants.ClusterWeightAcceptAll
				}
			}
		}
	}
	if p.Forward.EgressGateways != nil && len(p.Forward.EgressGateways) > 0 {
		for _, clusterConfigs := range p.Forward.EgressGateways {
			weightedEndpoints := clusterConfigs.Endpoints
			if weightedEndpoints == nil || len(*weightedEndpoints) == 0 {
				continue
			}
			for _, wze := range *weightedEndpoints {
				if wze.Weight == 0 {
					wze.Weight = constants.ClusterWeightAcceptAll
				}
			}
		}
	}
}

func (p *PipyConf) copyAllowedEndpoints(kubeController k8s.Controller, proxyRegistry *registry.ProxyRegistry) bool {
	ready := true
	p.AllowedEndpoints = make(map[string]string)
	allPods := kubeController.ListPods()
	for _, pod := range allPods {
		proxyUUID, err := GetProxyUUIDFromPod(pod)
		if err != nil {
			continue
		}
		proxy := proxyRegistry.GetConnectedProxy(proxyUUID)
		if proxy == nil {
			ready = false
			continue
		}
		if prettyConfig() {
			p.AllowedEndpoints[proxy.GetAddr()] = fmt.Sprintf("%s.%s", pod.Namespace, pod.Name)
		} else {
			p.AllowedEndpoints[proxy.GetAddr()] = ""
		}
		if len(proxy.GetAddr()) == 0 {
			ready = false
		}
	}
	allVms := kubeController.ListVms()
	for _, vm := range allVms {
		proxyUUID, err := GetProxyUUIDFromVm(vm)
		if err != nil {
			continue
		}
		proxy := proxyRegistry.GetConnectedProxy(proxyUUID)
		if proxy == nil {
			ready = false
			continue
		}
		if prettyConfig() {
			p.AllowedEndpoints[proxy.GetAddr()] = fmt.Sprintf("%s.%s", vm.Namespace, vm.Name)
		} else {
			p.AllowedEndpoints[proxy.GetAddr()] = ""
		}
		if len(proxy.GetAddr()) == 0 {
			ready = false
		}
	}
	if p.Inbound == nil {
		return ready
	}
	if len(p.Inbound.TrafficMatches) == 0 {
		return ready
	}
	for _, trafficMatch := range p.Inbound.TrafficMatches {
		if len(trafficMatch.SourceIPRanges) == 0 {
			continue
		}
		for ipRange := range trafficMatch.SourceIPRanges {
			ingressIP := strings.TrimSuffix(string(ipRange), "/32")
			if prettyConfig() {
				p.AllowedEndpoints[ingressIP] = "Ingress/Accessor"
			} else {
				p.AllowedEndpoints[ingressIP] = ""
			}
		}
	}
	return ready
}

func (itm *InboundTrafficMatch) addSourceIPRange(ipRange SourceIPRange, sourceSpec *SourceSecuritySpec) {
	if itm.SourceIPRanges == nil {
		itm.SourceIPRanges = make(map[SourceIPRange]*SourceSecuritySpec)
	}
	if _, exists := itm.SourceIPRanges[ipRange]; !exists {
		itm.SourceIPRanges[ipRange] = sourceSpec
	}
}

func (otm *OutboundTrafficMatch) addDestinationIPRange(ipRange DestinationIPRange, destinationSpec *DestinationSecuritySpec) {
	if otm.DestinationIPRanges == nil {
		otm.DestinationIPRanges = make(map[DestinationIPRange]*DestinationSecuritySpec)
	}
	if _, exists := otm.DestinationIPRanges[ipRange]; !exists {
		otm.DestinationIPRanges[ipRange] = destinationSpec
	}
}

func (itm *InboundTrafficMatch) setPort(port Port) {
	itm.Port = port
}

func (otm *OutboundTrafficMatch) setPort(port Port) {
	otm.Port = port
}

func (itm *InboundTrafficMatch) setProtocol(protocol Protocol) {
	protocol = Protocol(strings.ToLower(string(protocol)))
	if constants.ProtocolTCPServerFirst == protocol {
		itm.Protocol = constants.ProtocolTCP
	} else {
		itm.Protocol = protocol
	}
}

func (otm *OutboundTrafficMatch) setProtocol(protocol Protocol) {
	protocol = Protocol(strings.ToLower(string(protocol)))
	if constants.ProtocolTCPServerFirst == protocol {
		otm.Protocol = constants.ProtocolTCP
	} else {
		otm.Protocol = protocol
	}
}

func (itm *InboundTrafficMatch) setTCPServiceRateLimit(rateLimit *policyv1alpha1.RateLimitSpec) {
	if rateLimit == nil || rateLimit.Local == nil {
		itm.TCPRateLimit = nil
	} else {
		itm.TCPRateLimit = newTCPRateLimit(rateLimit.Local)
	}
}

func (itm *InboundTrafficMatch) newTCPServiceRouteRules() *InboundTCPServiceRouteRules {
	if itm.TCPServiceRouteRules == nil {
		itm.TCPServiceRouteRules = new(InboundTCPServiceRouteRules)
	}
	return itm.TCPServiceRouteRules
}

func (otm *OutboundTrafficMatch) newTCPServiceRouteRules() *OutboundTCPServiceRouteRules {
	if otm.TCPServiceRouteRules == nil {
		otm.TCPServiceRouteRules = new(OutboundTCPServiceRouteRules)
	}
	return otm.TCPServiceRouteRules
}

func (srr *InboundTCPServiceRouteRules) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if srr.TargetClusters == nil {
		srr.TargetClusters = make(WeightedClusters)
	}
	fnv32aClusterName := fnv32a(string(clusterName))
	srr.TargetClusters[ClusterName(fnv32aClusterName)] = weight
}

func (srr *OutboundTCPServiceRouteRules) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if srr.TargetClusters == nil {
		srr.TargetClusters = make(WeightedClusters)
	}
	fnv32aClusterName := fnv32a(string(clusterName))
	srr.TargetClusters[ClusterName(fnv32aClusterName)] = weight
}

func (srr *OutboundTCPServiceRouteRules) setAllowedEgressTraffic(allowedEgressTraffic bool) {
	srr.AllowedEgressTraffic = allowedEgressTraffic
}

func (srr *OutboundTCPServiceRouteRules) setEgressForwardGateway(egresssGateway *string) {
	srr.EgressForwardGateway = egresssGateway
}

func (itm *InboundTrafficMatch) addHTTPHostPort2Service(hostPort HTTPHostPort, ruleName HTTPRouteRuleName) {
	if itm.HTTPHostPort2Service == nil {
		itm.HTTPHostPort2Service = make(HTTPHostPort2Service)
	}

	fnv32aRuleName := fnv32a(string(ruleName))
	if preRuleName, exist := itm.HTTPHostPort2Service[hostPort]; exist {
		clen := len(ruleName)
		plen := len(preRuleName)
		if idx := strings.Index(string(preRuleName), "|"); idx > 0 {
			plen, _ = strconv.Atoi(string(preRuleName)[idx+1:])
		}
		if clen < plen {
			itm.HTTPHostPort2Service[hostPort] = HTTPRouteRuleName(fnv32aRuleName)
		}
	} else {
		itm.HTTPHostPort2Service[hostPort] = HTTPRouteRuleName(fnv32aRuleName)
	}
}

func (otm *OutboundTrafficMatch) addHTTPHostPort2Service(hostPort HTTPHostPort, ruleName HTTPRouteRuleName, desiredSuffix string) {
	if otm.HTTPHostPort2Service == nil {
		otm.HTTPHostPort2Service = make(HTTPHostPort2Service)
	}

	fnv32aRuleName := fnv32a(string(ruleName))
	if preRuleName, exist := otm.HTTPHostPort2Service[hostPort]; exist {
		clen := len(ruleName)
		plen := len(preRuleName)
		if idx := strings.Index(string(preRuleName), "|"); idx >= 0 {
			plen, _ = strconv.Atoi(string(preRuleName)[idx+1:])
		}
		if len(desiredSuffix) > 0 &&
			strings.HasSuffix(string(ruleName), desiredSuffix) &&
			strings.HasSuffix(string(preRuleName), desiredSuffix) {
			if clen < plen {
				otm.HTTPHostPort2Service[hostPort] = HTTPRouteRuleName(fnv32aRuleName)
			}
		} else if clen < plen {
			otm.HTTPHostPort2Service[hostPort] = HTTPRouteRuleName(fnv32aRuleName)
		}
	} else {
		otm.HTTPHostPort2Service[hostPort] = HTTPRouteRuleName(fnv32aRuleName)
	}
}

func (itm *InboundTrafficMatch) newHTTPServiceRouteRules(httpRouteRuleName HTTPRouteRuleName) *InboundHTTPRouteRules {
	if itm.HTTPServiceRouteRules == nil {
		itm.HTTPServiceRouteRules = make(InboundHTTPServiceRouteRules)
	}
	if len(httpRouteRuleName) == 0 {
		return nil
	}

	fnv32aRuleName := fnv32a(string(httpRouteRuleName))

	rules, exist := itm.HTTPServiceRouteRules[HTTPRouteRuleName(fnv32aRuleName)]
	if !exist || rules == nil {
		newCluster := new(InboundHTTPRouteRules)
		itm.HTTPServiceRouteRules[HTTPRouteRuleName(fnv32aRuleName)] = newCluster
		return newCluster
	}
	return rules
}

func (otm *OutboundTrafficMatch) newHTTPServiceRouteRules(httpRouteRuleName HTTPRouteRuleName) *OutboundHTTPRouteRules {
	if otm.HTTPServiceRouteRules == nil {
		otm.HTTPServiceRouteRules = make(OutboundHTTPServiceRouteRules)
	}
	if len(httpRouteRuleName) == 0 {
		return nil
	}

	fnv32aRuleName := fnv32a(string(httpRouteRuleName))

	rules, exist := otm.HTTPServiceRouteRules[HTTPRouteRuleName(fnv32aRuleName)]
	if !exist || rules == nil {
		newCluster := new(OutboundHTTPRouteRules)
		otm.HTTPServiceRouteRules[HTTPRouteRuleName(fnv32aRuleName)] = newCluster
		return newCluster
	}
	return rules
}

func (itp *InboundTrafficPolicy) newTrafficMatch(port Port) *InboundTrafficMatch {
	if itp.TrafficMatches == nil {
		itp.TrafficMatches = make(InboundTrafficMatches)
	}
	trafficMatch, exist := itp.TrafficMatches[port]
	if !exist || trafficMatch == nil {
		trafficMatch = new(InboundTrafficMatch)
		itp.TrafficMatches[port] = trafficMatch
		return trafficMatch
	}
	return trafficMatch
}

func (itp *InboundTrafficPolicy) getTrafficMatch(port Port) *InboundTrafficMatch {
	if itp.TrafficMatches == nil {
		return nil
	}
	if trafficMatch, exist := itp.TrafficMatches[port]; exist {
		return trafficMatch
	}
	return nil
}

func (otp *OutboundTrafficPolicy) newTrafficMatch(port Port, name string) (*OutboundTrafficMatch, bool) {
	namedPort := fmt.Sprintf(`%d=%s`, port, name)
	if otp.namedTrafficMatches == nil {
		otp.namedTrafficMatches = make(namedOutboundTrafficMatches)
	}
	trafficMatch, exists := otp.namedTrafficMatches[namedPort]
	if exists {
		return trafficMatch, true
	}

	trafficMatch = new(OutboundTrafficMatch)
	otp.namedTrafficMatches[namedPort] = trafficMatch

	if otp.TrafficMatches == nil {
		otp.TrafficMatches = make(OutboundTrafficMatches)
	}
	trafficMatches := otp.TrafficMatches[port]
	trafficMatches = append(trafficMatches, trafficMatch)
	otp.TrafficMatches[port] = trafficMatches
	return trafficMatch, false
}

func (hrrs *InboundHTTPRouteRules) setHTTPServiceRateLimit(rateLimit *policyv1alpha1.RateLimitSpec) {
	if rateLimit == nil || rateLimit.Local == nil {
		hrrs.HTTPRateLimit = nil
	} else {
		hrrs.HTTPRateLimit = newHTTPRateLimit(rateLimit.Local)
	}
}

func (hrrs *InboundHTTPRouteRules) addAllowedEndpoint(address Address, serviceName ServiceName) {
	if hrrs.AllowedEndpoints == nil {
		hrrs.AllowedEndpoints = make(AllowedEndpoints)
	}
	if _, exists := hrrs.AllowedEndpoints[address]; !exists {
		hrrs.AllowedEndpoints[address] = serviceName
	}
}

func (hrrs *InboundHTTPRouteRules) newHTTPServiceRouteRule(matchRule *HTTPMatchRule) (route *InboundHTTPRouteRule, duplicate bool) {
	for _, routeRule := range hrrs.RouteRules {
		if reflect.DeepEqual(*matchRule, routeRule.HTTPMatchRule) {
			return routeRule, true
		}
	}

	routeRule := new(InboundHTTPRouteRule)
	routeRule.HTTPMatchRule = *matchRule
	hrrs.RouteRules = append(hrrs.RouteRules, routeRule)
	return routeRule, false
}

func (hrrs *OutboundHTTPRouteRules) setServiceIdentity(serviceIdentity identity.ServiceIdentity) {
	hrrs.ServiceIdentity = serviceIdentity
}

func (hrrs *OutboundHTTPRouteRules) newHTTPServiceRouteRule(matchRule *HTTPMatchRule) (route *OutboundHTTPRouteRule, duplicate bool) {
	for _, routeRule := range hrrs.RouteRules {
		if reflect.DeepEqual(*matchRule, routeRule.HTTPMatchRule) {
			return routeRule, true
		}
	}

	routeRule := new(OutboundHTTPRouteRule)
	routeRule.HTTPMatchRule = *matchRule
	hrrs.RouteRules = append(hrrs.RouteRules, routeRule)
	return routeRule, false
}

func (hrrs *OutboundHTTPRouteRules) setEgressForwardGateway(egresssGateway *string) {
	hrrs.EgressForwardGateway = egresssGateway
}

func (hmr *HTTPMatchRule) addHeaderMatch(header Header, headerRegexp HeaderRegexp) {
	if hmr.Headers == nil {
		hmr.Headers = make(Headers)
	}
	hmr.Headers[header] = headerRegexp
}

func (hmr *HTTPMatchRule) addMethodMatch(method Method) {
	if hmr.allowedAnyMethod {
		return
	}
	if "*" == method {
		hmr.allowedAnyMethod = true
	}
	if hmr.allowedAnyMethod {
		hmr.Methods = nil
	} else {
		hmr.Methods = append(hmr.Methods, method)
	}
}

func (hrr *HTTPRouteRule) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if hrr.TargetClusters == nil {
		hrr.TargetClusters = make(WeightedClusters)
	}
	fnv32aClusterName := fnv32a(string(clusterName))
	hrr.TargetClusters[ClusterName(fnv32aClusterName)] = weight
}

func (hrr *HTTPRouteRule) addAllowedService(serviceName ServiceName) {
	if hrr.allowedAnyService {
		return
	}
	if "*" == serviceName {
		hrr.allowedAnyService = true
	}
	if hrr.allowedAnyService {
		hrr.AllowedServices = nil
	} else {
		hrr.AllowedServices = append(hrr.AllowedServices, serviceName)
	}
}

func (ihrr *InboundHTTPRouteRule) setRateLimit(rateLimit *policyv1alpha1.HTTPPerRouteRateLimitSpec) {
	ihrr.RateLimit = newHTTPPerRouteRateLimit(rateLimit)
}

func (itp *InboundTrafficPolicy) newClusterConfigs(clusterName ClusterName) *WeightedEndpoint {
	if itp.ClustersConfigs == nil {
		itp.ClustersConfigs = make(map[ClusterName]*WeightedEndpoint)
	}
	fnv32aClusterName := fnv32a(string(clusterName))
	cluster, exist := itp.ClustersConfigs[ClusterName(fnv32aClusterName)]
	if !exist || cluster == nil {
		newCluster := make(WeightedEndpoint, 0)
		itp.ClustersConfigs[ClusterName(fnv32aClusterName)] = &newCluster
		return &newCluster
	}
	return cluster
}

func (otp *OutboundTrafficPolicy) newClusterConfigs(clusterName ClusterName) *ClusterConfigs {
	if otp.ClustersConfigs == nil {
		otp.ClustersConfigs = make(map[ClusterName]*ClusterConfigs)
	}
	fnv32aClusterName := fnv32a(string(clusterName))
	cluster, exist := otp.ClustersConfigs[ClusterName(fnv32aClusterName)]
	if !exist || cluster == nil {
		newCluster := new(ClusterConfigs)
		otp.ClustersConfigs[ClusterName(fnv32aClusterName)] = newCluster
		return newCluster
	}
	return cluster
}

func (otp *ClusterConfigs) addWeightedEndpoint(address Address, port Port, weight Weight) {
	if otp.Endpoints == nil {
		weightedEndpoints := make(WeightedEndpoints)
		otp.Endpoints = &weightedEndpoints
	}
	otp.Endpoints.addWeightedEndpoint(address, port, weight)
}

func (otp *ClusterConfigs) addWeightedZoneEndpoint(address Address, port Port, weight Weight, cluster, lbType, contextPath, viaGw string) {
	if otp.Endpoints == nil {
		weightedEndpoints := make(WeightedEndpoints)
		otp.Endpoints = &weightedEndpoints
	}
	otp.Endpoints.addWeightedZoneEndpoint(address, port, weight, cluster, lbType, contextPath, viaGw)
}

func (wes *WeightedEndpoints) addWeightedEndpoint(address Address, port Port, weight Weight) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight: weight,
		}
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight: weight,
		}
	}
}

func (wes *WeightedEndpoints) addWeightedZoneEndpoint(address Address, port Port, weight Weight, cluster, lbType, contextPath, viaGw string) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight:      weight,
			Cluster:     cluster,
			LBType:      lbType,
			ContextPath: contextPath,
			ViaGateway:  viaGw,
		}
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight:      weight,
			Cluster:     cluster,
			LBType:      lbType,
			ContextPath: contextPath,
			ViaGateway:  viaGw,
		}
	}
}

func (we *WeightedEndpoint) addWeightedEndpoint(address Address, port Port, weight Weight) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*we)[httpHostPort] = weight
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*we)[httpHostPort] = weight
	}
}

func (otp *ClusterConfigs) setConnectionSettings(connectionSettings *policyv1alpha1.ConnectionSettingsSpec) {
	if connectionSettings == nil {
		otp.ConnectionSettings = nil
		return
	}
	otp.ConnectionSettings = new(ConnectionSettings)
	if connectionSettings.TCP != nil {
		otp.ConnectionSettings.TCP = new(TCPConnectionSettings)
		otp.ConnectionSettings.TCP.MaxConnections = connectionSettings.TCP.MaxConnections
		if connectionSettings.TCP.ConnectTimeout != nil {
			duration := connectionSettings.TCP.ConnectTimeout.Seconds()
			otp.ConnectionSettings.TCP.ConnectTimeout = &duration
		}
	}
	if connectionSettings.HTTP != nil {
		otp.ConnectionSettings.HTTP = new(HTTPConnectionSettings)
		otp.ConnectionSettings.HTTP.MaxRequests = connectionSettings.HTTP.MaxRequests
		otp.ConnectionSettings.HTTP.MaxRequestsPerConnection = connectionSettings.HTTP.MaxRequestsPerConnection
		otp.ConnectionSettings.HTTP.MaxPendingRequests = connectionSettings.HTTP.MaxPendingRequests
		otp.ConnectionSettings.HTTP.MaxRetries = connectionSettings.HTTP.MaxRetries
		if connectionSettings.HTTP.CircuitBreaking != nil {
			otp.ConnectionSettings.HTTP.CircuitBreaking = new(HTTPCircuitBreaking)
			if connectionSettings.HTTP.CircuitBreaking.StatTimeWindow != nil {
				duration := connectionSettings.HTTP.CircuitBreaking.StatTimeWindow.Seconds()
				otp.ConnectionSettings.HTTP.CircuitBreaking.StatTimeWindow = &duration
			}
			otp.ConnectionSettings.HTTP.CircuitBreaking.MinRequestAmount = connectionSettings.HTTP.CircuitBreaking.MinRequestAmount
			if connectionSettings.HTTP.CircuitBreaking.DegradedTimeWindow != nil {
				duration := connectionSettings.HTTP.CircuitBreaking.DegradedTimeWindow.Seconds()
				otp.ConnectionSettings.HTTP.CircuitBreaking.DegradedTimeWindow = &duration
			}
			if connectionSettings.HTTP.CircuitBreaking.SlowTimeThreshold != nil {
				duration := connectionSettings.HTTP.CircuitBreaking.SlowTimeThreshold.Seconds()
				otp.ConnectionSettings.HTTP.CircuitBreaking.SlowTimeThreshold = &duration
			}
			otp.ConnectionSettings.HTTP.CircuitBreaking.SlowAmountThreshold = connectionSettings.HTTP.CircuitBreaking.SlowAmountThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.SlowRatioThreshold = connectionSettings.HTTP.CircuitBreaking.SlowRatioThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.ErrorAmountThreshold = connectionSettings.HTTP.CircuitBreaking.ErrorAmountThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.ErrorRatioThreshold = connectionSettings.HTTP.CircuitBreaking.ErrorRatioThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.DegradedStatusCode = connectionSettings.HTTP.CircuitBreaking.DegradedStatusCode
			otp.ConnectionSettings.HTTP.CircuitBreaking.DegradedResponseContent = connectionSettings.HTTP.CircuitBreaking.DegradedResponseContent
		}
	}
}

func (otp *ClusterConfigs) setRetryPolicy(retryPolicy *policyv1alpha1.RetryPolicySpec) {
	if retryPolicy == nil {
		otp.RetryPolicy = nil
		return
	}
	otp.RetryPolicy = new(RetryPolicy)
	otp.RetryPolicy.RetryOn = retryPolicy.RetryOn
	otp.RetryPolicy.NumRetries = retryPolicy.NumRetries
	perTryTimeout := retryPolicy.PerTryTimeout.Seconds()
	otp.RetryPolicy.PerTryTimeout = &perTryTimeout
	retryBackoffBaseInterval := retryPolicy.RetryBackoffBaseInterval.Seconds()
	otp.RetryPolicy.RetryBackoffBaseInterval = &retryBackoffBaseInterval
}

func (ftp *ForwardTrafficPolicy) newForwardMatch(rule string) WeightedClusters {
	if ftp.ForwardMatches == nil {
		ftp.ForwardMatches = make(ForwardTrafficMatches)
	}
	forwardMatch, exist := ftp.ForwardMatches[rule]
	if !exist || forwardMatch == nil {
		forwardMatch = make(WeightedClusters)
		ftp.ForwardMatches[rule] = forwardMatch
		return forwardMatch
	}
	return forwardMatch
}

func (ftp *ForwardTrafficPolicy) newEgressGateway(clusterName ClusterName, mode string) *EgressGatewayClusterConfigs {
	if ftp.EgressGateways == nil {
		ftp.EgressGateways = make(map[ClusterName]*EgressGatewayClusterConfigs)
	}
	fnv32aClusterName := fnv32a(string(clusterName))
	cluster, exist := ftp.EgressGateways[ClusterName(fnv32aClusterName)]
	if !exist || cluster == nil {
		newCluster := new(EgressGatewayClusterConfigs)
		newCluster.Mode = mode
		ftp.EgressGateways[ClusterName(fnv32aClusterName)] = newCluster
		return newCluster
	}
	return cluster
}

// Len is the number of elements in the collection.
func (otms OutboundTrafficMatchSlice) Len() int {
	return len(otms)
}

// Less reports whether the element with index i
// must sort before the element with index j.
func (otms OutboundTrafficMatchSlice) Less(i, j int) bool {
	a, b := otms[i], otms[j]

	aLen, bLen := len(a.DestinationIPRanges), len(b.DestinationIPRanges)
	if aLen == 0 && bLen == 0 {
		return false
	}
	if aLen > 0 && bLen == 0 {
		return false
	}
	if aLen == 0 && bLen > 0 {
		return true
	}

	var aCidrs, bCidrs []*cidr.CIDR
	for ipRangea := range a.DestinationIPRanges {
		cidra, _ := cidr.ParseCIDR(string(ipRangea))
		aCidrs = append(aCidrs, cidra)
	}
	for ipRangeb := range b.DestinationIPRanges {
		cidrb, _ := cidr.ParseCIDR(string(ipRangeb))
		bCidrs = append(bCidrs, cidrb)
	}

	cidr.DescSortCIDRs(aCidrs)
	cidr.DescSortCIDRs(bCidrs)

	minLen := aLen
	if aLen > bLen {
		minLen = bLen
	}

	for n := 0; n < minLen; n++ {
		if cidr.CompareCIDR(aCidrs[n], bCidrs[n]) == 1 {
			return true
		}
	}

	return aLen > bLen
}

// Swap swaps the elements with indexes i and j.
func (otms OutboundTrafficMatchSlice) Swap(i, j int) {
	otms[j], otms[i] = otms[i], otms[j]
}

// Sort sorts data.
// It makes one call to data.Len to determine n and O(n*log(n)) calls to
// data.Less and data.Swap. The sort is not guaranteed to be stable.
func (otms *OutboundTrafficMatches) Sort() {
	for _, trafficMatches := range *otms {
		if len(trafficMatches) > 1 {
			sort.Sort(trafficMatches)
		}
	}
}

func (hrrs *OutboundHTTPRouteRuleSlice) sort() {
	if len(*hrrs) > 1 {
		sort.Sort(hrrs)
	}
}

func (hrrs *OutboundHTTPRouteRuleSlice) Len() int {
	return len(*hrrs)
}

func (hrrs *OutboundHTTPRouteRuleSlice) Swap(i, j int) {
	(*hrrs)[j], (*hrrs)[i] = (*hrrs)[i], (*hrrs)[j]
}

func (hrrs *OutboundHTTPRouteRuleSlice) Less(i, j int) bool {
	a, b := (*hrrs)[i], (*hrrs)[j]
	if strings.EqualFold(string(a.Path), string(b.Path)) {
		if len(a.Headers) > len(b.Headers) {
			return true
		}
		if len(a.Methods) > len(b.Methods) {
			return true
		}
		return false
	}

	if a.Path == constants.RegexMatchAll {
		return false
	}
	if b.Path == constants.RegexMatchAll {
		return true
	}
	return strings.Compare(string(a.Path), string(b.Path)) == -1
}

func (hrrs *InboundHTTPRouteRules) sort() {
	if len(hrrs.RouteRules) > 1 {
		sort.Sort(hrrs.RouteRules)
	}
}

func (irrs InboundHTTPRouteRuleSlice) Len() int {
	return len(irrs)
}

func (irrs InboundHTTPRouteRuleSlice) Swap(i, j int) {
	irrs[j], irrs[i] = irrs[i], irrs[j]
}

func (irrs InboundHTTPRouteRuleSlice) Less(i, j int) bool {
	a, b := irrs[i], irrs[j]
	if strings.EqualFold(string(a.Path), string(b.Path)) {
		if len(a.Headers) > len(b.Headers) {
			return true
		}
		if len(a.Methods) > len(b.Methods) {
			return true
		}
		return false
	}

	if a.Path == constants.RegexMatchAll {
		return false
	}
	if b.Path == constants.RegexMatchAll {
		return true
	}
	return strings.Compare(string(a.Path), string(b.Path)) == -1
}

func (ps *PluginSlice) Len() int {
	return len(*ps)
}

func (ps *PluginSlice) Swap(i, j int) {
	(*ps)[j], (*ps)[i] = (*ps)[i], (*ps)[j]
}

func (ps *PluginSlice) Less(i, j int) bool {
	a, b := (*ps)[i], (*ps)[j]
	return a.Priority > b.Priority
}

func fnv32a(ruleName string) string {
	if prettyConfig() {
		return ruleName
	}
	algorithm := fnv.New32a()
	if _, err := algorithm.Write([]byte(ruleName)); err == nil {
		return fmt.Sprintf("%d|%d", algorithm.Sum32(), len(ruleName))
	} else {
		log.Err(err).Msgf("fnv32a[%s]", ruleName)
	}
	return ruleName
}
