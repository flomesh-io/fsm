package utils

import (
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
)

// FilterEndpointSliceList filters the given EndpointSliceList to only include EndpointSlices that have a matching port.
func FilterEndpointSliceList(endpointSliceList *discoveryv1.EndpointSliceList, port corev1.ServicePort) []*discoveryv1.EndpointSlice {
	filtered := make([]*discoveryv1.EndpointSlice, 0, len(endpointSliceList.Items))

	for _, endpointSlice := range endpointSliceList.Items {
		endpointSlice := endpointSlice
		if !ignoreEndpointSlice(&endpointSlice, port) {
			filtered = append(filtered, &endpointSlice)
		}
	}

	return filtered
}

func ignoreEndpointSlice(endpointSlice *discoveryv1.EndpointSlice, port corev1.ServicePort) bool {
	//if endpointSlice.AddressType != discoveryv1.AddressTypeIPv4 {
	//	return true
	//}

	// ignore endpoint slices that don't have a matching port.
	return FindEndpointSlicePort(endpointSlice.Ports, port) == 0
}

// FindEndpointSlicePort finds the port in the EndpointSlice that matches the ServicePort.
func FindEndpointSlicePort(ports []discoveryv1.EndpointPort, svcPort corev1.ServicePort) int32 {
	portName := svcPort.Name
	for _, p := range ports {
		if p.Port == nil {
			return GetDefaultPort(svcPort)
		}

		if p.Name != nil && *p.Name == portName {
			return *p.Port
		}
	}

	return 0
}

// IsEndpointReady returns true if the given Endpoint is ready.
func IsEndpointReady(ep discoveryv1.Endpoint) bool {
	return ep.Conditions.Ready != nil && *ep.Conditions.Ready
}
