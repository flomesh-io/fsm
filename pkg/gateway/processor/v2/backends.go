package v2

import (
	"context"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/flomesh-io/fsm/pkg/k8s"
)

func (c *ConfigGenerator) processBackends() []interface{} {
	//configs := make(map[string]fgw.ServiceConfig)
	backends := make([]interface{}, 0)
	for svcPortName, svcInfo := range c.services {
		svcKey := svcInfo.svcPortName.NamespacedName
		svc, err := c.getServiceFromCache(svcKey)

		if err != nil {
			log.Error().Msgf("Failed to get Service %s: %s", svcKey, err)
			continue
		}

		if svc.Spec.Type == corev1.ServiceTypeExternalName {
			log.Warn().Msgf("Type of Service %s is %s, will be ignored", svcKey, corev1.ServiceTypeExternalName)
			continue
		}

		// don't create Backend resource if there are no endpoints
		targets := c.calculateEndpoints(svc, svcInfo.svcPortName.Port)
		if len(targets) == 0 {
			continue
		}

		//for _, enricher := range c.getServicePolicyEnrichers(svc) {
		//    enricher.Enrich(svcPortName, svcCfg)
		//}

		backends = append(backends, fgwv2.NewBackend(svcPortName, targets))
	}

	return backends
}

func (c *ConfigGenerator) calculateEndpoints(svc *corev1.Service, port *int32) []fgwv2.BackendTarget {
	// If the Service is headless, use the Endpoints to get the list of backends
	if k8s.IsHeadlessService(*svc) {
		return c.upstreamsByEndpoints(svc, port)
	}

	return c.upstreams(svc, port)
}

func (c *ConfigGenerator) upstreamsByEndpoints(svc *corev1.Service, port *int32) []fgwv2.BackendTarget {
	eps := &corev1.Endpoints{}
	if err := c.client.Get(context.TODO(), client.ObjectKeyFromObject(svc), eps); err != nil {
		log.Error().Msgf("Failed to get Endpoints of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(eps.Subsets) == 0 {
		return nil
	}

	svcPort, err := gwutils.GetServicePort(svc, port)
	if err != nil {
		log.Error().Msgf("Failed to get ServicePort: %s", err)
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
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			discoveryv1.LabelServiceName: svc.Name,
		},
	})
	if err != nil {
		log.Error().Msgf("Failed to convert LabelSelector to Selector: %s", err)
		return nil
	}

	endpointSliceList := &discoveryv1.EndpointSliceList{}
	if err := c.client.List(context.TODO(), endpointSliceList, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error().Msgf("Failed to list EndpointSlice of Service %s/%s: %s", svc.Namespace, svc.Name, err)
		return nil
	}

	if len(endpointSliceList.Items) == 0 {
		return nil
	}

	svcPort, err := gwutils.GetServicePort(svc, port)
	if err != nil {
		log.Error().Msgf("Failed to get ServicePort: %s", err)
		return nil
	}

	filteredSlices := gwutils.FilterEndpointSliceList(endpointSliceList, svcPort)
	if len(filteredSlices) == 0 {
		log.Error().Msgf("no valid endpoints found for Service %s/%s and port %v", svc.Namespace, svc.Name, svcPort)
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
