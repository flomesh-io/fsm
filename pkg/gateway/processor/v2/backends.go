package v2

import (
	"context"
	"strconv"
	"strings"

	"github.com/flomesh-io/fsm/pkg/connector"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *ConfigGenerator) processBackends() []fgwv2.Resource {
	backends := make([]fgwv2.Resource, 0)
	for _, bk := range c.backends {
		backends = append(backends, bk)
	}

	return backends
}

func (c *ConfigGenerator) toFGWBackend(svcPort *fgwv2.ServicePortName) *fgwv2.Backend {
	if svcPort == nil {
		return nil
	}

	if bk, ok := c.backends[svcPort.String()]; ok {
		return bk
	}

	// don't create Backend resource if there are no endpoints
	targets := c.findFGWBackendTargets(svcPort)
	if len(targets) == 0 {
		return nil
	}

	bk := fgwv2.NewBackend(svcPort.String(), toFGWAppProtocol(svcPort.AppProtocol), targets)
	c.backends[svcPort.String()] = bk

	return bk
}

func (c *ConfigGenerator) findFGWBackendTargets(svcPort *fgwv2.ServicePortName) []fgwv2.BackendTarget {
	if svcPort == nil {
		return nil
	}

	svcKey := svcPort.NamespacedName
	svc, err := c.getServiceFromCache(svcKey)

	if err != nil {
		log.Error().Msgf("[GW] Failed to get Service %s: %s", svcKey, err)
		return nil
	}

	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		log.Warn().Msgf("[GW] Type of Service %s is %s, will be ignored", svcKey, corev1.ServiceTypeExternalName)
		return nil
	}

	return c.calculateEndpoints(svc, svcPort.Port)
}

func (c *ConfigGenerator) calculateEndpoints(svc *corev1.Service, port *int32) []fgwv2.BackendTarget {
	// If the Service is headless and has no selector, use Endpoints to get the list of backends
	if isHeadlessServiceWithoutSelector(svc) {
		return c.upstreamsByEndpoints(svc, port)
	}

	return c.upstreams(svc, port)
}

func (c *ConfigGenerator) upstreamsByEndpoints(svc *corev1.Service, port *int32) []fgwv2.BackendTarget {
	// cross cluster endpoints
	if len(svc.Annotations) > 0 {
		if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
			svcMeta := connector.Decode(svc, v)
			found := false

			targetPort := *port
			for _, portSpec := range svc.Spec.Ports {
				if portSpec.Port == *port {
					targetPort = portSpec.TargetPort.IntVal
					found = true
					break
				}
			}
			if found {
				endpointSet := make(map[endpointContext]struct{})
				for address, metadata := range svcMeta.Endpoints {
					if len(metadata.Native.ViaGatewayHTTP) == 0 {
						ep := endpointContext{address: string(address), port: targetPort}
						endpointSet[ep] = struct{}{}
					} else {
						if segs := strings.Split(metadata.Native.ViaGatewayHTTP, ":"); len(segs) == 2 {
							if portStr, convErr := strconv.Atoi(segs[1]); convErr == nil {
								viaPort := int32(portStr & 0xFFFF)
								viaAddr := segs[0]
								ep := endpointContext{address: viaAddr, port: viaPort}
								endpointSet[ep] = struct{}{}
							}
						}
					}
				}
				if len(endpointSet) > 0 {
					return toFGWBackendTargets(endpointSet)
				} else {
					log.Error().Msgf("[GW] no valid endpoints found for Service %s/%s and port %v", svc.Namespace, svc.Name, *port)
					return nil
				}
			}
		}
	}

	eps := &corev1.Endpoints{}
	if err := c.client.Get(context.TODO(), client.ObjectKeyFromObject(svc), eps); err != nil {
		log.Error().Msgf("[GW] Failed to get Endpoints of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(eps.Subsets) == 0 {
		return nil
	}

	svcPort, err := gwutils.GetServicePort(svc, port)
	if err != nil {
		log.Error().Msgf("[GW] Failed to get ServicePort: %s", err)
		return nil
	}

	endpointSet := make(map[endpointContext]struct{})
	for _, subset := range eps.Subsets {
		if endpointPort := gwutils.FindEndpointPort(subset.Ports, svcPort); endpointPort > 0 && endpointPort <= 65535 {
			for _, address := range subset.Addresses {
				ep := endpointContext{address: address.IP, port: endpointPort}
				endpointSet[ep] = struct{}{}
			}
		}
	}

	return toFGWBackendTargets(endpointSet)
}

func (c *ConfigGenerator) upstreamsByEndpointSlices(svc *corev1.Service, port *int32) []fgwv2.BackendTarget {
	// cross cluster endpoints
	if len(svc.Annotations) > 0 {
		if v, exists := svc.Annotations[connector.AnnotationMeshEndpointAddr]; exists {
			svcMeta := connector.Decode(svc, v)
			found := false

			targetPort := *port
			for _, portSpec := range svc.Spec.Ports {
				if portSpec.Port == *port {
					targetPort = portSpec.TargetPort.IntVal
					found = true
					break
				}
			}
			if found {
				endpointSet := make(map[endpointContext]struct{})
				for address, metadata := range svcMeta.Endpoints {
					if len(metadata.Native.ViaGatewayHTTP) > 0 {
						if segs := strings.Split(metadata.Native.ViaGatewayHTTP, ":"); len(segs) == 2 {
							if portStr, convErr := strconv.Atoi(segs[1]); convErr == nil {
								viaPort := int32(portStr & 0xFFFF)
								viaAddr := segs[0]
								ep := endpointContext{address: viaAddr, port: viaPort}
								endpointSet[ep] = struct{}{}
							}
						}
					} else if len(metadata.Native.ViaGatewayGRPC) > 0 {
						if segs := strings.Split(metadata.Native.ViaGatewayGRPC, ":"); len(segs) == 2 {
							if portStr, convErr := strconv.Atoi(segs[1]); convErr == nil {
								viaPort := int32(portStr & 0xFFFF)
								viaAddr := segs[0]
								ep := endpointContext{address: viaAddr, port: viaPort}
								endpointSet[ep] = struct{}{}
							}
						}
					} else {
						ep := endpointContext{address: string(address), port: targetPort}
						endpointSet[ep] = struct{}{}
					}
				}
				if len(endpointSet) > 0 {
					return toFGWBackendTargets(endpointSet)
				} else {
					log.Error().Msgf("[GW] no valid endpoints found for Service %s/%s and port %v", svc.Namespace, svc.Name, *port)
					return nil
				}
			}
		}
	}

	// in-cluster endpoints
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			discoveryv1.LabelServiceName: svc.Name,
		},
	})
	if err != nil {
		log.Error().Msgf("[GW] Failed to convert LabelSelector to Selector: %s", err)
		return nil
	}

	endpointSliceList := &discoveryv1.EndpointSliceList{}
	if err := c.client.List(context.TODO(), endpointSliceList, client.InNamespace(svc.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error().Msgf("[GW] Failed to list EndpointSlice of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(endpointSliceList.Items) == 0 {
		return nil
	}

	svcPort, err := gwutils.GetServicePort(svc, port)
	if err != nil {
		log.Error().Msgf("[GW] Failed to get ServicePort: %s", err)
		return nil
	}

	filteredSlices := gwutils.FilterEndpointSliceList(endpointSliceList, svcPort)
	if len(filteredSlices) == 0 {
		log.Error().Msgf("[GW] no valid endpoints found for Service %s/%s and port %v", svc.Namespace, svc.Name, svcPort)
		return nil
	}

	endpointSet := make(map[endpointContext]struct{})
	for _, eps := range filteredSlices {
		for _, endpoint := range eps.Endpoints {
			if !gwutils.IsEndpointReady(endpoint) {
				continue
			}

			if endpointPort := gwutils.FindEndpointSlicePort(eps.Ports, svcPort); endpointPort > 0 && endpointPort <= 65535 {
				for _, address := range endpoint.Addresses {
					ep := endpointContext{address: address, port: endpointPort}
					endpointSet[ep] = struct{}{}
				}
			}
		}
	}

	return toFGWBackendTargets(endpointSet)
}
