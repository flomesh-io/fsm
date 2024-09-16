package catalog

import (
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/service"
)

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (mc *MeshCatalog) listEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	isolationCidrs := mc.GetIsolationCidrs()
	for _, provider := range mc.endpointsProviders {
		eps := provider.ListEndpointsForService(svc)
		if len(eps) == 0 {
			log.Trace().Msgf("No endpoints found for service %s by endpoints provider %s", provider.GetID(), svc)
			continue
		}
		if len(isolationCidrs) > 0 {
			for _, ep := range eps {
				isolation := false
				for _, isolationCidr := range isolationCidrs {
					if isolationCidr.Has(ep.IP) {
						isolation = true
						break
					}
				}
				if !isolation {
					endpoints = append(endpoints, ep)
				}
			}
		} else {
			endpoints = append(endpoints, eps...)
		}
	}
	return endpoints
}

// getDNSResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN
func (mc *MeshCatalog) getDNSResolvableServiceEndpoints(svc service.MeshService) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	isolationCidrs := mc.GetIsolationCidrs()
	for _, provider := range mc.endpointsProviders {
		eps := provider.GetResolvableEndpointsForService(svc)
		if len(eps) == 0 {
			continue
		}
		if len(isolationCidrs) > 0 {
			for _, ep := range eps {
				isolation := false
				for _, isolationCidr := range isolationCidrs {
					if isolationCidr.Has(ep.IP) {
						isolation = true
						break
					}
				}
				if !isolation {
					endpoints = append(endpoints, ep)
				}
			}
		} else {
			endpoints = append(endpoints, eps...)
		}
	}
	return endpoints
}

// ListAllowedUpstreamEndpointsForService returns the list of endpoints over which the downstream client identity
// is allowed access the upstream service
func (mc *MeshCatalog) ListAllowedUpstreamEndpointsForService(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) []endpoint.Endpoint {
	outboundEndpoints := mc.listEndpointsForService(upstreamSvc)

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return outboundEndpoints
	}

	if len(outboundEndpoints) == 0 {
		return nil
	}

	// In SMI mode, the endpoints for an upstream service must be filtered based on the service account
	// associated with the endpoint. Only endpoints associated with authorized service accounts as referenced
	// in SMI TrafficTarget resources should be returned.
	//
	// The following code filters the upstream service's endpoints for this purpose.
	outboundEndpointsSet := make(map[string][]endpoint.Endpoint)
	for _, ep := range outboundEndpoints {
		ipStr := ep.IP.String()
		outboundEndpointsSet[ipStr] = append(outboundEndpointsSet[ipStr], ep)
	}

	// allowedEndpoints comprises of only those endpoints from outboundEndpoints that matches the endpoints from listEndpointsForServiceIdentity
	// i.e. only those intersecting endpoints are taken into cosideration
	var allowedEndpoints []endpoint.Endpoint
	for _, destSvcIdentity := range mc.ListOutboundServiceIdentities(downstreamIdentity) {
		for _, ep := range mc.listEndpointsForServiceIdentity(destSvcIdentity) {
			epIPStr := ep.IP.String()
			// check if endpoint IP is allowed
			if _, ok := outboundEndpointsSet[epIPStr]; ok {
				// add all allowed endpoints on the pod to result list
				allowedEndpoints = append(allowedEndpoints, outboundEndpointsSet[epIPStr]...)
			}
		}
	}

	return allowedEndpoints
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/flomesh-io/fsm/issues/3188]
func (mc *MeshCatalog) listEndpointsForServiceIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	isolationCidrs := mc.GetIsolationCidrs()
	for _, provider := range mc.endpointsProviders {
		eps := provider.ListEndpointsForIdentity(serviceIdentity)
		if len(eps) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service account=%s", provider.GetID(), serviceIdentity)
			continue
		}
		if len(isolationCidrs) > 0 {
			for _, ep := range eps {
				isolation := false
				for _, isolationCidr := range isolationCidrs {
					if isolationCidr.Has(ep.IP) {
						isolation = true
						break
					}
				}
				if !isolation {
					endpoints = append(endpoints, ep)
				}
			}
		} else {
			endpoints = append(endpoints, eps...)
		}
	}
	return endpoints
}
