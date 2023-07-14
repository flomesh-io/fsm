package catalog

import (
	"fmt"
	"net"
	"strings"

	mapset "github.com/deckarep/golang-set"

	policyV1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

// GetAccessControlTrafficPolicy returns the access control traffic policy for the given mesh service
// Depending on if the AccessControl API is enabled, the policies will be generated either from the AccessControl
// or Kubernetes AccessControl API.
func (mc *MeshCatalog) GetAccessControlTrafficPolicy(svc service.MeshService) (*trafficpolicy.AccessControlTrafficPolicy, error) {
	if !mc.configurator.GetFeatureFlags().EnableAccessControlPolicy {
		return nil, nil
	}

	aclPolicy := mc.policyController.GetAccessControlPolicy(svc)
	if aclPolicy == nil {
		log.Trace().Msgf("Did not find AccessControl policy for service %s", svc)
		return nil, nil
	}

	// The status field will be updated after the policy is processed.
	// Note: The original pointer returned by cache.Store must not be modified for thread safety.
	aclWithStatus := *aclPolicy

	sourcePrincipals := mapset.NewSet()
	var trafficRoutingRules []*trafficpolicy.Rule
	var trafficMatches []*trafficpolicy.AccessControlTrafficMatch
	if len(aclPolicy.Spec.Backends) > 0 {
		if err := mc.getStrictAccessControlTrafficPolicy(svc, aclPolicy, &aclWithStatus, sourcePrincipals, &trafficMatches, &trafficRoutingRules); err != nil {
			return nil, err
		}
	} else {
		if err := mc.getPermissiveAccessControlTrafficPolicy(svc, aclPolicy, &aclWithStatus, sourcePrincipals, &trafficMatches, &trafficRoutingRules); err != nil {
			return nil, err
		}
	}

	if len(trafficMatches) == 0 {
		// Since no trafficMatches exist for this AccessControl config, it implies that the given
		// MeshService does not map to this AccessControl config.
		log.Debug().Msgf("No acl traffic matches exist for MeshService %s, no acl config required", svc.SidecarLocalClusterName())
		return nil, nil
	}

	aclWithStatus.Status = policyV1alpha1.AccessControlStatus{
		CurrentStatus: "committed",
		Reason:        "successfully committed by the system",
	}
	if _, err := mc.kubeController.UpdateStatus(&aclWithStatus); err != nil {
		log.Error().Err(err).Msg("Error updating status for AccessControl")
	}

	// Create an inbound traffic policy from the routing rules
	// TODO(#3779): Implement HTTP route matching from AccessControl.Spec.Matches
	var httpRoutePolicy *trafficpolicy.InboundTrafficPolicy
	if trafficRoutingRules != nil {
		httpRoutePolicy = &trafficpolicy.InboundTrafficPolicy{
			Name:      fmt.Sprintf("%s_from_%s", svc, aclPolicy.Name),
			Hostnames: []string{"*"},
			Rules:     trafficRoutingRules,
		}
	}

	return &trafficpolicy.AccessControlTrafficPolicy{
		TrafficMatches:    trafficMatches,
		HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{httpRoutePolicy},
	}, nil
}

func (mc *MeshCatalog) getPermissiveAccessControlTrafficPolicy(svc service.MeshService, aclPolicy *policyV1alpha1.AccessControl, aclWithStatus *policyV1alpha1.AccessControl, sourcePrincipals mapset.Set, trafficMatches *[]*trafficpolicy.AccessControlTrafficMatch, trafficRoutingRules *[]*trafficpolicy.Rule) error {
	upstreamTrafficSetting := mc.policyController.GetUpstreamTrafficSetting(
		policy.UpstreamTrafficSettingGetOpt{MeshService: &svc})

	trafficMatch := &trafficpolicy.AccessControlTrafficMatch{
		Name:     service.AccessControlTrafficMatchName(svc.Name, svc.Namespace, svc.Port, svc.Protocol),
		Port:     uint32(svc.Port),
		Protocol: svc.Protocol,
	}

	if upstreamTrafficSetting != nil {
		trafficMatch.RateLimit = upstreamTrafficSetting.Spec.RateLimit
	}

	var sourceIPRanges []string
	sourceIPSet := mapset.NewSet() // Used to avoid duplicate IP ranges
	for _, source := range aclPolicy.Spec.Sources {
		switch source.Kind {
		case policyV1alpha1.KindService:
			sourceMeshSvc := service.MeshService{
				Name:      source.Name,
				Namespace: source.Namespace,
			}
			endpoints := mc.listEndpointsForService(sourceMeshSvc)
			if len(endpoints) == 0 {
				aclWithStatus.Status = policyV1alpha1.AccessControlStatus{
					CurrentStatus: "error",
					Reason:        fmt.Sprintf("endpoints not found for service %s/%s", source.Namespace, source.Name),
				}
				if _, err := mc.kubeController.UpdateStatus(&aclWithStatus); err != nil {
					log.Error().Err(err).Msg("Error updating status for AccessControl")
				}
				return fmt.Errorf("Could not list endpoints of the source service %s/%s specified in the AccessControl %s/%s",
					source.Namespace, source.Name, aclPolicy.Namespace, aclPolicy.Name)
			}

			for _, ep := range endpoints {
				sourceCIDR := ep.IP.String() + singeIPPrefixLen
				if sourceIPSet.Add(sourceCIDR) {
					sourceIPRanges = append(sourceIPRanges, sourceCIDR)
				}
			}

		case policyV1alpha1.KindIPRange:
			if _, _, err := net.ParseCIDR(source.Name); err != nil {
				// This should not happen because the validating webhook will prevent it. This check has
				// been added as a safety net to prevent invalid configs.
				log.Error().Err(err).Msgf("Invalid IP address range specified in AccessControl %s/%s: %s",
					aclPolicy.Namespace, aclPolicy.Name, source.Name)
				continue
			}
			sourceIPRanges = append(sourceIPRanges, source.Name)
		}
	}

	// If this acl is corresponding to an HTTP port, wildcard the downstream's identity
	// because the identity cannot be verified for HTTP traffic. HTTP based acl can
	// restrict downstreams based on their endpoint's IP address.
	if strings.EqualFold(svc.Protocol, constants.ProtocolHTTP) {
		sourcePrincipals.Add(identity.WildcardPrincipal)
	}

	trafficMatch.SourceIPRanges = sourceIPRanges
	*trafficMatches = append(*trafficMatches, trafficMatch)

	// Build the routing rule for this backend and source combination.
	// Currently AccessControl only supports a wildcard HTTP route. The
	// 'Matches' field in the spec can be used to extend this to perform
	// stricter enforcement.
	backendCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(svc.SidecarLocalClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
	protocol := strings.ToLower(trafficMatch.Protocol)
	if protocol == constants.ProtocolHTTP || protocol == constants.ProtocolGRPC {
		routingRule := &trafficpolicy.Rule{
			Route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch:   trafficpolicy.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(backendCluster),
			},
			AllowedPrincipals: sourcePrincipals,
		}
		*trafficRoutingRules = append(*trafficRoutingRules, routingRule)
	}
	return nil
}

func (mc *MeshCatalog) getStrictAccessControlTrafficPolicy(svc service.MeshService, aclPolicy *policyV1alpha1.AccessControl, aclWithStatus *policyV1alpha1.AccessControl, sourcePrincipals mapset.Set, trafficMatches *[]*trafficpolicy.AccessControlTrafficMatch, trafficRoutingRules *[]*trafficpolicy.Rule) error {
	for _, backend := range aclPolicy.Spec.Backends {
		if backend.Name != svc.Name || backend.Port.Number != int(svc.TargetPort) {
			continue
		}

		upstreamTrafficSetting := mc.policyController.GetUpstreamTrafficSetting(
			policy.UpstreamTrafficSettingGetOpt{MeshService: &svc})

		trafficMatch := &trafficpolicy.AccessControlTrafficMatch{
			Name:     service.AccessControlTrafficMatchName(svc.Name, svc.Namespace, uint16(backend.Port.Number), backend.Port.Protocol),
			Port:     uint32(backend.Port.Number),
			Protocol: backend.Port.Protocol,
			TLS:      backend.TLS,
		}

		if upstreamTrafficSetting != nil {
			trafficMatch.RateLimit = upstreamTrafficSetting.Spec.RateLimit
		}

		var sourceIPRanges []string
		sourceIPSet := mapset.NewSet() // Used to avoid duplicate IP ranges
		for _, source := range aclPolicy.Spec.Sources {
			switch source.Kind {
			case policyV1alpha1.KindService:
				sourceMeshSvc := service.MeshService{
					Name:      source.Name,
					Namespace: source.Namespace,
				}
				endpoints := mc.listEndpointsForService(sourceMeshSvc)
				if len(endpoints) == 0 {
					aclWithStatus.Status = policyV1alpha1.AccessControlStatus{
						CurrentStatus: "error",
						Reason:        fmt.Sprintf("endpoints not found for service %s/%s", source.Namespace, source.Name),
					}
					if _, err := mc.kubeController.UpdateStatus(&aclWithStatus); err != nil {
						log.Error().Err(err).Msg("Error updating status for AccessControl")
					}
					return fmt.Errorf("Could not list endpoints of the source service %s/%s specified in the AccessControl %s/%s",
						source.Namespace, source.Name, aclPolicy.Namespace, aclPolicy.Name)
				}

				for _, ep := range endpoints {
					sourceCIDR := ep.IP.String() + singeIPPrefixLen
					if sourceIPSet.Add(sourceCIDR) {
						sourceIPRanges = append(sourceIPRanges, sourceCIDR)
					}
				}

			case policyV1alpha1.KindIPRange:
				if _, _, err := net.ParseCIDR(source.Name); err != nil {
					// This should not happen because the validating webhook will prevent it. This check has
					// been added as a safety net to prevent invalid configs.
					log.Error().Err(err).Msgf("Invalid IP address range specified in AccessControl %s/%s: %s",
						aclPolicy.Namespace, aclPolicy.Name, source.Name)
					continue
				}
				sourceIPRanges = append(sourceIPRanges, source.Name)

			case policyV1alpha1.KindAuthenticatedPrincipal:
				if backend.TLS != nil && backend.TLS.SkipClientCertValidation {
					sourcePrincipals.Add(identity.WildcardServiceIdentity.String())
				} else {
					sourcePrincipals.Add(source.Name)
				}
			}
		}

		// If this acl is corresponding to an HTTP port, wildcard the downstream's identity
		// because the identity cannot be verified for HTTP traffic. HTTP based acl can
		// restrict downstreams based on their endpoint's IP address.
		if strings.EqualFold(backend.Port.Protocol, constants.ProtocolHTTP) {
			sourcePrincipals.Add(identity.WildcardPrincipal)
		}

		trafficMatch.SourceIPRanges = sourceIPRanges
		*trafficMatches = append(*trafficMatches, trafficMatch)

		// Build the routing rule for this backend and source combination.
		// Currently AccessControl only supports a wildcard HTTP route. The
		// 'Matches' field in the spec can be used to extend this to perform
		// stricter enforcement.
		backendCluster := service.WeightedCluster{
			ClusterName: service.ClusterName(svc.SidecarLocalClusterName()),
			Weight:      constants.ClusterWeightAcceptAll,
		}
		protocol := strings.ToLower(trafficMatch.Protocol)
		if protocol == constants.ProtocolHTTP || protocol == constants.ProtocolGRPC {
			routingRule := &trafficpolicy.Rule{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   trafficpolicy.WildCardRouteMatch,
					WeightedClusters: mapset.NewSet(backendCluster),
				},
				AllowedPrincipals: sourcePrincipals,
			}
			*trafficRoutingRules = append(*trafficRoutingRules, routingRule)
		}
	}
	return nil
}
