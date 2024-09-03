package catalog

import (
	"fmt"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha4"
	"k8s.io/apimachinery/pkg/types"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/errcode"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/smi"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

// GetOutboundMeshTrafficPolicy returns the outbound mesh traffic policy for the given downstream identity
//
// The function works as follows:
//  1. If permissive mode is enabled, builds outbound mesh traffic policies to reach every upstream service
//     discovered using service discovery, using wildcard routes.
//  2. In SMI mode, builds outbound mesh traffic policies to reach every upstream service corresponding
//     to every upstream service account that this downstream is authorized to access using SMI TrafficTarget
//     policies.
//  3. Process TraficSplit policies and update the weights for the upstream services based on the policies.
//
// The route configurations are consolidated per port, such that upstream services using the same port are a part
// of the same route configuration. This is required to avoid route conflicts that can occur when the same hostname
// needs to be routed differently based on the port used.
func (mc *MeshCatalog) GetOutboundMeshTrafficPolicy(downstreamIdentity identity.ServiceIdentity) *trafficpolicy.OutboundMeshTrafficPolicy {
	var trafficMatches []*trafficpolicy.TrafficMatch
	var clusterConfigs []*trafficpolicy.MeshClusterConfig
	routeConfigPerPort := make(map[int][]*trafficpolicy.OutboundTrafficPolicy)
	downstreamSvcAccount := downstreamIdentity.ToK8sServiceAccount()
	servicesResolvableSet := make(map[string][]interface{})

	var egressPolicy *trafficpolicy.EgressTrafficPolicy
	var egressPolicyGetted bool
	var egressEnabled bool

	// For each service, build the traffic policies required to access it.
	// It is important to aggregate HTTP route configs by the service's port.
	for _, meshSvc := range mc.ListOutboundServicesForIdentity(downstreamIdentity) {
		meshSvc := meshSvc // To prevent loop variable memory aliasing in for loop

		egressEnabled, egressPolicyGetted, egressPolicy = mc.enableEgressSrviceForIdentity(downstreamIdentity, egressPolicyGetted, egressPolicy, meshSvc)
		monitoredNamespace := mc.kubeController.IsMonitoredNamespace(meshSvc.Namespace)
		existIntraEndpoints := false

		// Retrieve the destination IP address from the endpoints for this service
		// IP range must not have duplicates, use a mapset to only add unique IP ranges
		var destinationIPRanges []string
		destinationIPSet := mapset.NewSet()
		endpoints := mc.getDNSResolvableServiceEndpoints(meshSvc)
		for _, endp := range endpoints {
			ipCIDR := endp.IP.String() + "/32"
			if added := destinationIPSet.Add(ipCIDR); added {
				destinationIPRanges = append(destinationIPRanges, ipCIDR)
			}
			if !existIntraEndpoints {
				if len(endp.ClusterKey) == 0 && (monitoredNamespace || egressEnabled) {
					existIntraEndpoints = true
				}
			}
		}

		if mc.configurator.IsLocalDNSProxyEnabled() && !mc.configurator.IsWildcardDNSProxyEnabled() {
			if !existIntraEndpoints {
				resolvableIPSet := mapset.NewSet()
				for _, endp := range endpoints {
					resolvableIPSet.Add(endp.IP.String())
				}
				if resolvableIPSet.Cardinality() > 0 {
					addrItems := resolvableIPSet.ToSlice()
					sort.SliceStable(addrItems, func(i, j int) bool {
						addr1 := addrItems[i].(string)
						addr2 := addrItems[j].(string)
						return addr1 < addr2
					})
					servicesResolvableSet[meshSvc.PolicyName(true)] = addrItems
				}
			}
		}

		// ---
		// Create the cluster config for this upstream service
		clusterConfigForServicePort := &trafficpolicy.MeshClusterConfig{
			Name:                            meshSvc.SidecarClusterName(),
			Service:                         meshSvc,
			EnableSidecarActiveHealthChecks: mc.configurator.GetFeatureFlags().EnableSidecarActiveHealthChecks,
			UpstreamTrafficSetting:          mc.policyController.GetUpstreamTrafficSetting(policy.UpstreamTrafficSettingGetOpt{MeshService: &meshSvc}),
		}
		clusterConfigs = append(clusterConfigs, clusterConfigForServicePort)

		hasTrafficSplitWildCard := false
		var routeMatches []*trafficpolicy.HTTPRouteMatchWithWeightedClusters
		// Check if there is a traffic split corresponding to this service.
		// The upstream clusters are to be derived from the traffic split backends
		// in that case.
		trafficSplits := mc.meshSpec.ListTrafficSplits(smi.WithTrafficSplitApexService(meshSvc))
		if len(trafficSplits) > 0 {
			// Program routes to the backends specified in the traffic split
			for _, split := range trafficSplits {
				routeMatch := new(trafficpolicy.HTTPRouteMatchWithWeightedClusters)
				if len(split.Spec.Matches) > 0 {
					routeMatch.HasSplitMatches = true
					routeMatch.RouteMatches = mc.getSplitRouteMatches(split)
				} else {
					hasTrafficSplitWildCard = true
				}

				for _, backend := range split.Spec.Backends {
					routeMatch.UpstreamClusters = mc.mergeSplitUpstreamClusters(meshSvc, backend, routeMatch.UpstreamClusters)
				}
				routeMatches = append(routeMatches, routeMatch)
			}
		} else {
			hasTrafficSplitWildCard = true
			routeMatch := new(trafficpolicy.HTTPRouteMatchWithWeightedClusters)
			routeMatch.UpstreamClusters = mc.mergeUpstreamClusters(meshSvc, routeMatch.UpstreamClusters)
			routeMatches = append(routeMatches, routeMatch)
		}

		// ---
		// Create a TrafficMatch for this upstream service and port combination.
		// The TrafficMatch will be used by LDS to program a filter chain match
		// for this upstream service, port, and destination IP ranges. This
		// will be programmed on the downstream client.
		for _, routeMatch := range routeMatches {
			trafficMatchForServicePort := &trafficpolicy.TrafficMatch{
				Name:                meshSvc.OutboundTrafficMatchName(),
				DestinationPort:     int(meshSvc.Port),
				DestinationProtocol: meshSvc.Protocol,
				DestinationIPRanges: destinationIPRanges,
				WeightedClusters:    routeMatch.UpstreamClusters,
			}
			trafficMatches = append(trafficMatches, trafficMatchForServicePort)
			log.Trace().Msgf("Built traffic match %s for downstream %s", trafficMatchForServicePort.Name, downstreamIdentity)
		}

		// Build the HTTP route configs for this service and port combination.
		// If the port's protocol corresponds to TCP, we can skip this step
		if meshSvc.Protocol == constants.ProtocolTCP || meshSvc.Protocol == constants.ProtocolTCPServerFirst {
			continue
		}

		// Create a route to access the upstream service via it's hostnames and upstream weighted clusters
		localNamespace := downstreamSvcAccount.Namespace == meshSvc.Namespace || len(meshSvc.CloudInheritedFrom) > 0
		httpHostNamesForServicePort := mc.getHostnamesForService(meshSvc, localNamespace, endpoints)
		outboundTrafficPolicy := trafficpolicy.NewOutboundTrafficPolicy(meshSvc.PolicyName(true), httpHostNamesForServicePort)
		retryPolicy := mc.GetRetryPolicy(downstreamIdentity, meshSvc)

		hasWildCardRoute := false
		for _, routeMatch := range routeMatches {
			for _, route := range routeMatch.RouteMatches {
				if route.Path == constants.RegexMatchAll {
					hasWildCardRoute = true
				}
				if err := outboundTrafficPolicy.AddRoute(route, retryPolicy, routeMatch.UpstreamClusters...); err != nil {
					log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrAddingRouteToOutboundTrafficPolicy)).
						Msgf("Error adding route to outbound mesh HTTP traffic policy for destination %s", meshSvc)
					continue
				}
			}
		}
		if !hasWildCardRoute {
			upstreamClusters := mc.getWildCardRouteUpstreamClusters(hasTrafficSplitWildCard, routeMatches)
			if err := outboundTrafficPolicy.AddRoute(trafficpolicy.WildCardRouteMatch, retryPolicy, upstreamClusters...); err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrAddingRouteToOutboundTrafficPolicy)).
					Msgf("Error adding route to outbound mesh HTTP traffic policy for destination %s", meshSvc)
				continue
			}
		}
		routeConfigPerPort[int(meshSvc.Port)] = append(routeConfigPerPort[int(meshSvc.Port)], outboundTrafficPolicy)
	}

	return &trafficpolicy.OutboundMeshTrafficPolicy{
		TrafficMatches:          trafficMatches,
		ClustersConfigs:         clusterConfigs,
		HTTPRouteConfigsPerPort: routeConfigPerPort,
		ServicesResolvableSet:   servicesResolvableSet,
	}
}

func (mc *MeshCatalog) getHostnamesForService(meshSvc service.MeshService, localNamespace bool, endpoints []endpoint.Endpoint) []string {
	var httpHostNamesForServicePort []string
	sam := mc.configurator.GetServiceAccessMode()
	san := mc.configurator.GetServiceAccessNames()
	if sam == configv1alpha3.ServiceAccessModeDomain || sam == configv1alpha3.ServiceAccessModeMixed {
		httpHostNamesForServicePort = k8s.GetHostnamesForService(meshSvc, san, localNamespace)
	}
	if sam == configv1alpha3.ServiceAccessModeIP || sam == configv1alpha3.ServiceAccessModeMixed {
		for _, endp := range endpoints {
			if !san.MustWithServicePort {
				httpHostNamesForServicePort = append(httpHostNamesForServicePort, endp.IP.String())
			}
			httpHostNamesForServicePort = append(httpHostNamesForServicePort, fmt.Sprintf("%s:%d", endp.IP.String(), meshSvc.Port))
		}
	}
	return httpHostNamesForServicePort
}

func (mc *MeshCatalog) getWildCardRouteUpstreamClusters(hasTrafficSplitWildCard bool, routeMatches []*trafficpolicy.HTTPRouteMatchWithWeightedClusters) []service.WeightedCluster {
	var upstreamClusters []service.WeightedCluster
	upstreamClusterMap := make(map[service.ClusterName]bool)
	if hasTrafficSplitWildCard {
		for _, routeMatch := range routeMatches {
			if routeMatch.HasSplitMatches {
				continue
			}
			for _, upstreamCluster := range routeMatch.UpstreamClusters {
				if _, exist := upstreamClusterMap[upstreamCluster.ClusterName]; !exist {
					weightedCluster := service.WeightedCluster{
						ClusterName: upstreamCluster.ClusterName,
						Weight:      upstreamCluster.Weight,
					}
					upstreamClusters = append(upstreamClusters, weightedCluster)
					upstreamClusterMap[upstreamCluster.ClusterName] = true
				}
			}
		}
	} else {
		for _, routeMatch := range routeMatches {
			for _, upstreamCluster := range routeMatch.UpstreamClusters {
				if _, exist := upstreamClusterMap[upstreamCluster.ClusterName]; !exist {
					weightedCluster := service.WeightedCluster{
						ClusterName: upstreamCluster.ClusterName,
						Weight:      constants.ClusterWeightAcceptAll,
					}
					upstreamClusters = append(upstreamClusters, weightedCluster)
					upstreamClusterMap[upstreamCluster.ClusterName] = true
				}
			}
		}
	}
	return upstreamClusters
}

func (mc *MeshCatalog) mergeUpstreamClusters(meshSvc service.MeshService, upstreamClusters []service.WeightedCluster) []service.WeightedCluster {
	wc := service.WeightedCluster{
		ClusterName: service.ClusterName(meshSvc.SidecarClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
	if meshSvc.IsMultiClusterService() {
		aa, fo, _, weight, _ := mc.multiclusterController.GetLbWeightForService(meshSvc)
		if aa && weight > 0 {
			wc.Weight = weight
		} else if fo {
			wc.Weight = constants.ClusterWeightFailOver
		}
	}
	// No TrafficSplit for this upstream service, so use a default weighted cluster
	upstreamClusters = append(upstreamClusters, wc)
	return upstreamClusters
}

func (mc *MeshCatalog) mergeSplitUpstreamClusters(meshSvc service.MeshService, backend split.TrafficSplitBackend, upstreamClusters []service.WeightedCluster) []service.WeightedCluster {
	backendNamespace := meshSvc.Namespace
	backendService := backend.Service
	if segs := strings.Split(backend.Service, "/"); len(segs) == 2 {
		backendNamespace = segs[0]
		backendService = segs[1]
	}
	cnsLocal := make(map[service.ClusterName]bool)
	var aas []service.ClusterName
	var fos []service.ClusterName
	{
		backendMeshSvc := service.MeshService{
			Namespace: backendNamespace, // Backends belong to the same namespace as the apex service
			Name:      backendService,
		}
		targetPort, err := mc.kubeController.GetTargetPortForServicePort(
			types.NamespacedName{Namespace: backendNamespace, Name: backendMeshSvc.Name}, meshSvc.Port)
		if err == nil {
			backendMeshSvc.TargetPort = targetPort
			aas = append(aas, service.ClusterName(backendMeshSvc.SidecarClusterName()))
			cnsLocal[service.ClusterName(backendMeshSvc.SidecarClusterName())] = true
		}
	}
	{
		backendMeshSvc := service.MeshService{
			Namespace: backendNamespace, // Backends belong to the same namespace as the apex service
			Name:      backendService,
			Port:      meshSvc.Port,
		}
		targetPorts := mc.multiclusterController.GetTargetPortForServicePort(
			types.NamespacedName{Namespace: backendMeshSvc.Namespace, Name: backendMeshSvc.Name}, meshSvc.Port)
		if len(targetPorts) > 0 {
			for targetPort, aa := range targetPorts {
				backendMeshSvc.TargetPort = targetPort
				if _, exists := cnsLocal[service.ClusterName(backendMeshSvc.SidecarClusterName())]; !exists {
					if aa {
						aas = append(aas, service.ClusterName(backendMeshSvc.SidecarClusterName()))
					} else {
						fos = append(fos, service.ClusterName(backendMeshSvc.SidecarClusterName()))
					}
				}
			}
		}
	}
	upstreamClusters = activeUpstreamClusters(aas, backend, upstreamClusters)
	upstreamClusters = failOverUpstreamClusters(fos, upstreamClusters)
	return upstreamClusters
}

func (mc *MeshCatalog) enableEgressSrviceForIdentity(downstreamIdentity identity.ServiceIdentity, egressPolicyGetted bool, egressPolicy *trafficpolicy.EgressTrafficPolicy, meshSvc service.MeshService) (bool, bool, *trafficpolicy.EgressTrafficPolicy) {
	egressEnabled := mc.configurator.IsEgressEnabled()
	if !egressEnabled {
		if !egressPolicyGetted {
			egressPolicy, _ = mc.GetEgressTrafficPolicy(downstreamIdentity)
			egressPolicyGetted = true
		}
		if egressPolicy != nil {
			egressEnabled = mc.isEgressService(meshSvc, egressPolicy)
		}
	}
	return egressEnabled, egressPolicyGetted, egressPolicy
}

func (mc *MeshCatalog) getSplitRouteMatches(split *split.TrafficSplit) (splitRouteMatches []trafficpolicy.HTTPRouteMatch) {
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		if trafficSpecs.Spec.Matches == nil {
			continue
		}
		for _, match := range split.Spec.Matches {
			if match.Name == trafficSpecs.Name {
				for _, trafficSpecsMatches := range trafficSpecs.Spec.Matches {
					serviceRoute := trafficpolicy.HTTPRouteMatch{
						Path:          trafficSpecsMatches.PathRegex,
						PathMatchType: trafficpolicy.PathMatchRegex,
						Methods:       trafficSpecsMatches.Methods,
						Headers:       trafficSpecsMatches.Headers,
					}

					// When pathRegex or/and methods are not defined, they will be wildcarded
					if serviceRoute.Path == "" {
						serviceRoute.Path = constants.RegexMatchAll
					}
					if len(serviceRoute.Methods) == 0 {
						serviceRoute.Methods = []string{constants.WildcardHTTPMethod}
					}
					splitRouteMatches = append(splitRouteMatches, serviceRoute)
				}
				break
			}
		}
	}
	return splitRouteMatches
}

func activeUpstreamClusters(aas []service.ClusterName, backend split.TrafficSplitBackend, upstreamClusters []service.WeightedCluster) []service.WeightedCluster {
	if len(aas) > 0 {
		totalWeight := backend.Weight
		for index, cn := range aas {
			weight := totalWeight / (len(aas) - index)
			totalWeight -= weight
			wc := service.WeightedCluster{
				ClusterName: cn,
				Weight:      weight,
			}
			upstreamClusters = append(upstreamClusters, wc)
		}
	}
	return upstreamClusters
}

func failOverUpstreamClusters(fos []service.ClusterName, upstreamClusters []service.WeightedCluster) []service.WeightedCluster {
	if len(fos) > 0 {
		for _, cn := range fos {
			wc := service.WeightedCluster{
				ClusterName: cn,
				Weight:      constants.ClusterWeightFailOver,
			}
			upstreamClusters = append(upstreamClusters, wc)
		}
	}
	return upstreamClusters
}

func (mc *MeshCatalog) isEgressService(meshSvc service.MeshService, egressPolicy *trafficpolicy.EgressTrafficPolicy) bool {
	egressEnabled := false
	san := mc.configurator.GetServiceAccessNames()
	hostnames := k8s.GetHostnamesForService(meshSvc, san, true)
	for _, routeConfigs := range egressPolicy.HTTPRouteConfigsPerPort {
		if egressEnabled {
			break
		}
		if len(routeConfigs) == 0 {
			continue
		}
		for _, routeConfig := range routeConfigs {
			if egressEnabled {
				break
			}
			if len(routeConfig.Hostnames) == 0 {
				continue
			}
			for _, host := range routeConfig.Hostnames {
				if egressEnabled {
					break
				}
				for _, hostname := range hostnames {
					if hostname == host {
						egressEnabled = true
						break
					}
				}
			}
		}
	}
	return egressEnabled
}

// ListOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/flomesh-io/fsm/issues/3188]
func (mc *MeshCatalog) ListOutboundServicesForIdentity(serviceIdentity identity.ServiceIdentity) []service.MeshService {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return mc.listMeshServices()
	}

	svcAccount := serviceIdentity.ToK8sServiceAccount()
	serviceSet := mapset.NewSet()
	var allowedServices []service.MeshService
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name != svcAccount.Name || source.Namespace != svcAccount.Namespace {
				// Source doesn't match the downstream's service identity
				continue
			}

			sa := identity.K8sServiceAccount{
				Name:      t.Spec.Destination.Name,
				Namespace: t.Spec.Destination.Namespace,
			}

			for _, destService := range mc.getServicesForServiceIdentity(sa.ToServiceIdentity()) {
				if added := serviceSet.Add(destService); added {
					allowedServices = append(allowedServices, destService)
				}
			}
			break
		}
	}

	return allowedServices
}
