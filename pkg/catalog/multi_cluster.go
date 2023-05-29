package catalog

import (
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/trafficpolicy"
)

// GetExportTrafficPolicy returns the export policy for the given mesh service
func (mc *MeshCatalog) GetExportTrafficPolicy(svc service.MeshService) (*trafficpolicy.ServiceExportTrafficPolicy, error) {
	exportedRule, err := mc.multiclusterController.GetExportedRule(svc)
	if err != nil {
		return nil, err
	}
	if exportedRule == nil {
		return nil, nil
	}

	var trafficMatches []*trafficpolicy.ServiceExportTrafficMatch

	trafficMatch := &trafficpolicy.ServiceExportTrafficMatch{
		Name:     service.ExportedServiceTrafficMatchName(svc.Name, svc.Namespace, uint16(exportedRule.PortNumber), svc.Protocol),
		Port:     uint32(exportedRule.PortNumber),
		Protocol: svc.Protocol,
	}

	controllerServices := mc.multiclusterController.GetIngressControllerServices()
	if len(controllerServices) > 0 {
		sourceIPSet := mapset.NewSet() // Used to avoid duplicate IP ranges
		for _, controllerService := range controllerServices {
			if endpoints := mc.listEndpointsForService(controllerService); len(endpoints) > 0 {
				for _, ep := range endpoints {
					sourceCIDR := ep.IP.String() + singeIPPrefixLen
					if sourceIPSet.Add(sourceCIDR) {
						trafficMatch.SourceIPRanges = append(trafficMatch.SourceIPRanges, sourceCIDR)
					}
				}
			}
		}
	}

	trafficMatches = append(trafficMatches, trafficMatch)

	var trafficRoutingRules []*trafficpolicy.Rule
	sourcePrincipals := mapset.NewSet()

	// If this acl is corresponding to an HTTP port, wildcard the downstream's identity
	// because the identity cannot be verified for HTTP traffic. HTTP based acl can
	// restrict downstreams based on their endpoint's IP address.
	if strings.EqualFold(svc.Protocol, constants.ProtocolHTTP) {
		sourcePrincipals.Add(identity.WildcardPrincipal)
	}

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
		trafficRoutingRules = append(trafficRoutingRules, routingRule)
	}

	// Create an inbound traffic policy from the routing rules
	// TODO(#3779): Implement HTTP route matching from AccessControl.Spec.Matches
	var httpRoutePolicy *trafficpolicy.InboundTrafficPolicy
	if trafficRoutingRules != nil {
		httpRoutePolicy = &trafficpolicy.InboundTrafficPolicy{
			Name:      fmt.Sprintf("%s_from_%s", svc, "IngressController"),
			Hostnames: []string{"*"},
			Rules:     trafficRoutingRules,
		}
	}

	return &trafficpolicy.ServiceExportTrafficPolicy{
		TrafficMatches:    trafficMatches,
		HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{httpRoutePolicy},
	}, nil
}
